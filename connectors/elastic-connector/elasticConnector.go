package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/gwos/tng/connectors"
	"github.com/gwos/tng/log"
	"github.com/gwos/tng/milliseconds"
	_ "github.com/gwos/tng/milliseconds"
	"github.com/gwos/tng/transit"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	kibanaApiSavedObjectsPath = "http://localhost:5601/kibana/api/saved_objects/"
)

const (
	typePhrase  = "phrase"
	typePhrases = "phrases"
	typeRange   = "range"
	typeExists  = "exists"
)

func CollectMetrics() ([]transit.MonitoredResource, []transit.InventoryResource, []transit.ResourceGroup, error) {
	// TODO: selected queries with overridden time intervals will be retrieved from configs, here we will need only their ids (titles) to retrieve filters
	storedQueries, _ := retrieveStoredQueries(nil)

	// TODO: these should come from environment variables
	cfg := elasticsearch.Config{
		Addresses: []string{ // TODO: multiple load balanced elastic search
			"http://localhost:9200",
			// "http://localhost:9201",
		},
	}
	esClient, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Error(err)
		if esClient == nil {
			return nil, nil, nil, err
		}
	}
	if esClient == nil {
		log.Error("Could not create client")
		return nil, nil, nil, fmt.Errorf("could not create client")
	}

	hosts, groups := make(map[string]hostStruct), make(map[string]map[string]struct{})
	for _, storedQuery := range storedQueries {
		var must, mustNot, should []interface{}
		indexSet := make(map[string]struct{})

		for _, filter := range storedQuery.filters {
			addQueryClause(filter, &must, &mustNot, &should)
			index := filter["index"].(string)
			indexPattern := retrieveIndexPattern(index)
			indexSet[indexPattern] = struct{}{}
		}

		var from, to = storedQuery.timeFilter.from, storedQuery.timeFilter.to
		filter := createTimeFilter(&from, &to)
		query := createQuery(must, mustNot, should, filter)

		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(query); err != nil {
			log.Error("Error encoding query: %s", err)
		}

		var indexes []string
		for index := range indexSet {
			indexes = append(indexes, index)
		}

		res, err := esClient.Search(
			esClient.Search.WithContext(context.Background()),
			esClient.Search.WithIndex(indexes...),
			esClient.Search.WithBody(&buf),
			esClient.Search.WithTrackTotalHits(true),
			esClient.Search.WithPretty(),
		)
		if err != nil {
			log.Error("Error getting response: %s", err)
		}

		if res.IsError() {
			var e map[string]interface{}
			if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
				log.Error("Error parsing the response body: %s", err)
			} else {
				// Print the response status and error information.
				log.Error("[%s] %s: %s",
					res.Status(),
					e["error"].(map[string]interface{})["type"],
					e["error"].(map[string]interface{})["reason"],
				)
			}
		}

		responseBody, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Error(err)
		}
		var result map[string]interface{}
		err = json.Unmarshal(responseBody, &result)
		if err != nil {
			log.Error(err)
		}

		var startTime, endTime = parseTime(from, true), parseTime(to, false)
		timeInterval := timeIntervalStruct{
			startTime: startTime,
			endTime:   endTime,
		}

		hits := result["hits"].(map[string]interface{})["hits"].([]interface{})
		for _, h := range hits {
			hit := h.(map[string]interface{})
			if hit["_source"].(map[string]interface{})["container"] != nil {
				container := hit["_source"].(map[string]interface{})["container"].(map[string]interface{})

				hostName := container["name"].(string)

				var groupName string
				if container["labels"] != nil && container["labels"].(map[string]interface{})["com_docker_compose_project"] != nil {
					groupName = container["labels"].(map[string]interface{})["com_docker_compose_project"].(string)
				}
				if group, exists := groups[groupName]; exists {
					group[hostName] = struct{}{}
				} else {
					group := make(map[string]struct{})
					group[hostName] = struct{}{}
					groups[groupName] = group
				}
				groups[groupName][hostName] = struct{}{}

				if host, exists := hosts[hostName]; exists {
					services := host.services
					var found = false
					for i := range services {
						if services[i].name == storedQuery.title {
							services[i].hits = services[i].hits + 1
							found = true
							break
						}
					}
					if !found {
						service := serviceStruct{name: storedQuery.title, hits: 1, timeInterval: timeInterval}
						services = append(services, service)
						host.services = services
					}
					hosts[hostName] = host
				} else {
					service := serviceStruct{name: storedQuery.title, hits: 1, timeInterval: timeInterval}
					host := hostStruct{name: hostName, services: []serviceStruct{service}, hostGroup: groupName}
					hosts[hostName] = host
				}
			}
		}
	}

	mrs := make([]transit.MonitoredResource, len(hosts))
	irs := make([]transit.InventoryResource, len(hosts))
	i := 0
	for _, host := range hosts {
		monitoredServices := make([]transit.MonitoredService, len(host.services))
		inventoryServices := make([]transit.InventoryService, len(host.services))

		for i, service := range host.services {
			timeInterval := &transit.TimeInterval{
				EndTime:   milliseconds.MillisecondTimestamp{Time: service.timeInterval.endTime},
				StartTime: milliseconds.MillisecondTimestamp{Time: service.timeInterval.startTime},
			}
			metric, _ := connectors.CreateMetric("hits", service.hits, timeInterval, transit.UnitCounter)
			monitoredService, _ := connectors.CreateService(service.name, host.name, []transit.TimeSeries{*metric})
			inventoryService := connectors.CreateInventoryService(service.name, host.name)
			monitoredServices[i] = *monitoredService
			inventoryServices[i] = inventoryService
		}
		monitoredResource, _ := connectors.CreateResource(host.name, monitoredServices)
		inventoryResource := connectors.CreateInventoryResource(host.name, inventoryServices)
		mrs[i] = *monitoredResource
		irs[i] = inventoryResource
		i++
	}

	rgs := make([]transit.ResourceGroup, len(groups))
	j := 0
	for group, hostsInGroup := range groups {
		monitoredResourceRefs := make([]transit.MonitoredResourceRef, len(hostsInGroup))
		k := 0
		for host := range hostsInGroup {
			monitoredResourceRef := connectors.CreateMonitoredResourceRef(host, "", transit.Host)
			monitoredResourceRefs[k] = monitoredResourceRef
			k++
		}
		resourceGroup := connectors.CreateResourceGroup(group, group, transit.HostGroup, monitoredResourceRefs)
		rgs[j] = resourceGroup
		j++
	}

	return mrs, irs, rgs, nil
}

func retrieveStoredQueries(ids []string) ([]storedQueryStruct, int) {
	var storedQueries []storedQueryStruct

	var search string
	if ids != nil {
		search = "&search_fields=title&search="
		for index, id := range ids {
			search = search + id
			if index != len(ids) {
				search = search + "|"
			}
		}
	}

	page := 0
	perPage := 1000
	total := 0
	firstCall := true

	for total >= page*perPage {
		page = page + 1
		result := retrieveSavedObjects("_find?page="+strconv.Itoa(page)+"&per_page="+strconv.Itoa(perPage)+"&type=query"+search, nil)
		savedObjects := result["saved_objects"].([]interface{})
		extractStoredQueries(savedObjects, &storedQueries)
		if firstCall {
			total = int(result["total"].(float64))
			firstCall = false
		}
	}

	return storedQueries, len(storedQueries)
}

func retrieveIndexPattern(id string) string {
	var result = retrieveSavedObjects("index-pattern/"+id, nil)
	title := result["attributes"].(map[string]interface{})["title"].(string)
	return title
}

func retrieveSavedObjects(path string, body io.Reader) map[string]interface{} {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := http.Client{Transport: tr}
	var request *http.Request
	var response *http.Response
	var err error

	request, err = http.NewRequest(http.MethodGet, kibanaApiSavedObjectsPath+path, body)
	if err != nil {
		log.Error("Error getting response: %s", err)
	}
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("kbn-xsrf", "true")

	response, err = client.Do(request)
	if err != nil {
		log.Error("Error getting response: %s", err)
	}
	if response == nil {
		log.Error("Error getting response: response is nil")
	}
	if response.StatusCode == 400 {
		log.Error("Not Found!")
	}
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Error("Error reading response: %s", err)
	}
	err = response.Body.Close()
	if err != nil {
		log.Error("Error processing response: %s", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(responseBody, &result)
	if err != nil {
		log.Error("Error parsing response: %s", err)
	}
	return result
}

func extractStoredQueries(savedObjects []interface{}, storedQueries *[]storedQueryStruct) {
	for _, so := range savedObjects {
		savedObject := so.(map[string]interface{})

		id := savedObject["id"].(string)
		name := savedObject["attributes"].(map[string]interface{})["title"].(string)
		description := savedObject["attributes"].(map[string]interface{})["description"].(string)

		filtersAttribute := savedObject["attributes"].(map[string]interface{})["filters"].([]interface{})
		var filters []map[string]interface{}
		for _, f := range filtersAttribute {
			filter := f.(map[string]interface{})
			disabled := filter["meta"].(map[string]interface{})["disabled"].(bool)
			if disabled {
				continue
			}
			filters = append(filters, filter["meta"].(map[string]interface{}))
		}
		var tFilter timeFilterStruct
		if savedObject["attributes"].(map[string]interface{})["timefilter"] != nil {
			timeFilter := savedObject["attributes"].(map[string]interface{})["timefilter"].(map[string]interface{})
			tFilter.from = timeFilter["from"].(string)
			tFilter.to = timeFilter["to"].(string)
		} else {
			tFilter.from = "now-$interval"
			tFilter.to = "now"
		}
		*storedQueries = append(*storedQueries, storedQueryStruct{id, name, description,
			tFilter, filters})
	}
}

func addQueryClause(filter map[string]interface{}, must *[]interface{}, mustNot *[]interface{}, should *[]interface{}) {
	queryType, negate, key := filter["type"].(string), filter["negate"].(bool), filter["key"].(string)
	switch queryType {
	case typePhrase:
		q := map[string]interface{}{
			"match": map[string]interface{}{
				key: filter["value"].(string),
			}}
		if !negate {
			*must = append(*must, q)
		} else {
			*mustNot = append(*mustNot, q)
		}
		break
	case typePhrases:
		params := filter["params"].([]interface{})
		for _, param := range params {
			param := param.(string)
			if !negate {
				*should = append(*should, map[string]interface{}{
					"match": map[string]interface{}{
						key: param,
					}})
			} else {
				*should = append(*should, map[string]interface{}{
					"bool": map[string]interface{}{
						"must_not": map[string]interface{}{
							"match": map[string]interface{}{
								key: param,
							},
						},
					}})
			}
		}
		break
	case typeRange:
		params := filter["params"].(map[string]interface{})
		r := map[string]interface{}{
			"range": map[string]interface{}{
				key: params,
			}}
		if !negate {
			*must = append(*must, r)
		} else {
			*mustNot = append(*mustNot, r)
		}
		break
	case typeExists:
		q := map[string]interface{}{
			"exists": map[string]interface{}{
				"field": key,
			}}
		if !negate {
			*must = append(*must, q)
		} else {
			*mustNot = append(*mustNot, q)
		}
		break
	default:
		log.Error("Could not add query clause: unknown type '%s'", queryType)
		break
	}
}

func createTimeFilter(from *string, to *string) []interface{} {
	interval := strconv.Itoa(connectors.Timer) + "s"
	if strings.Contains(*from, "$interval") {
		*from = strings.ReplaceAll(*from, "$interval", interval)
	}
	if strings.Contains(*to, "$interval") {
		*to = strings.ReplaceAll(*to, "$interval", interval)
	}
	var filter []interface{}
	filter = append(filter, map[string]interface{}{
		"range": map[string]interface{}{
			"@timestamp": map[string]interface{}{
				"gte": from,
				"lt":  to,
			},
		},
	})
	return filter
}

func createQuery(must []interface{}, mustNot []interface{}, should []interface{}, filter []interface{}) map[string]interface{} {
	query, boolClause := make(map[string]interface{}), make(map[string]interface{})
	boolClause["must"], boolClause["must_not"], boolClause["should"], boolClause["filter"] = must, mustNot, should, filter
	if should != nil {
		boolClause["minimum_should_match"] = 1
	}
	query["bool"] = boolClause
	queryBody := map[string]interface{}{
		"query": query,
	}
	return queryBody
}

func parseTime(timeString string, isFrom bool) time.Time {
	if strings.Contains(timeString, "now") {
		return parseTimeExpression(timeString, isFrom)
	}
	layout := "2006-01-02T15:04:05.000Z"
	result, err := time.Parse(layout, timeString)
	if err != nil {
		log.Error(err)
	}
	return result
}

// converts relative expressions such as "now-5d" to Time
func parseTimeExpression(timeExpression string, isFrom bool) time.Time {
	now := time.Now()
	if timeExpression == "now" {
		return now
	}
	operator := timeExpression[3:4]
	expression := timeExpression[4:]
	var rounded = false
	if strings.Contains(expression, "/") {
		expression = expression[:len(expression)-2]
		rounded = true
	}
	interval := expression[:len(expression)-1]
	period := expression[len(expression)-1:]
	i, err := strconv.Atoi(interval)
	if operator == "-" {
		i = -i
	}
	if err != nil {
		log.Error("Error parsing time filter expression: %s", err)
	}
	var result time.Time
	switch period {
	case "y":
		result = now.AddDate(i, 0, 0)
		if rounded {
			if isFrom {
				result = time.Date(result.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
			} else {
				result = time.Date(result.Year()+1, 1, 1, 0, 0, 0, 0, time.UTC)
				result = result.Add(-1 * time.Millisecond)
			}
		}
		break
	case "M":
		result = now.AddDate(0, i, 0)
		if rounded {
			if isFrom {
				result = time.Date(result.Year(), result.Month(), 1, 0, 0, 0, 0, time.UTC)
			} else {
				result = time.Date(result.Year(), result.Month()+1, 1, 0, 0, 0, 0, time.UTC)
				result = result.Add(-1 * time.Millisecond)
			}
		}
		break
	case "w":
		dayOfDesiredWeek := now.AddDate(0, 0, 7*i)
		if rounded {
			var offsetFromSunday int
			var offsetToSaturday int
			switch dayOfDesiredWeek.Weekday() {
			case time.Monday:
				offsetFromSunday = 1
				offsetToSaturday = 5
				break
			case time.Tuesday:
				offsetFromSunday = 2
				offsetToSaturday = 4
				break
			case time.Wednesday:
				offsetFromSunday = 3
				offsetToSaturday = 3
				break
			case time.Thursday:
				offsetFromSunday = 4
				offsetToSaturday = 2
				break
			case time.Friday:
				offsetFromSunday = 5
				offsetToSaturday = 1
				break
			case time.Saturday:
				offsetFromSunday = 6
				offsetToSaturday = 0
				break
			case time.Sunday:
				offsetFromSunday = 0
				offsetToSaturday = 6
				break
			}
			if isFrom {
				result = time.Date(dayOfDesiredWeek.Year(), dayOfDesiredWeek.Month(), dayOfDesiredWeek.Day()-offsetFromSunday, 0, 0, 0, 0, time.UTC)
			} else {
				result = time.Date(dayOfDesiredWeek.Year(), dayOfDesiredWeek.Month(), dayOfDesiredWeek.Day()+offsetToSaturday+1, 0, 0, 0, 0, time.UTC)
				result = result.Add(-1 * time.Millisecond)
			}
		} else {
			result = dayOfDesiredWeek
		}
		break
	case "d":
		result = now.AddDate(0, 0, i)
		if rounded {
			if isFrom {
				result = time.Date(result.Year(), result.Month(), result.Day(), 0, 0, 0, 0, time.UTC)
			} else {
				result = time.Date(result.Year(), result.Month(), result.Day()+1, 0, 0, 0, 0, time.UTC)
				result = result.Add(-1 * time.Millisecond)
			}
		}
	case "h":
		result = now.Add(time.Duration(i) * time.Hour)
		if rounded {
			if isFrom {
				result = time.Date(result.Year(), result.Month(), result.Day(), result.Hour()+1, 0, 0, 0, time.UTC)
			} else {
				result = result.Add(-1 * time.Millisecond)
			}
		}
		break
	case "m":
		result = now.Add(time.Duration(i) * time.Minute)
		if rounded {
			if isFrom {
				result = time.Date(result.Year(), result.Month(), result.Day(), result.Hour(), result.Minute(), 0, 0, time.UTC)
			} else {
				result = time.Date(result.Year(), result.Month(), result.Day(), result.Hour(), result.Minute()+1, 0, 0, time.UTC)
				result = result.Add(-1 * time.Millisecond)
			}
		}
		break
	case "s":
		result = now.Add(time.Duration(i) * time.Second)
		if rounded {
			if isFrom {
				result = time.Date(result.Year(), result.Month(), result.Day(), result.Hour(), result.Minute(), result.Second(), 0, time.UTC)
			} else {
				result = time.Date(result.Year(), result.Month(), result.Day(), result.Hour(), result.Minute(), result.Second()+1, 0, time.UTC)
				result = result.Add(-1 * time.Millisecond)
			}
		}
		break
	default:
		log.Error("Error parsing time filter expression: unknown period format '" + period + "'")
	}
	return result
}

type storedQueryStruct struct {
	id          string
	title       string
	description string
	timeFilter  timeFilterStruct
	filters     []map[string]interface{}
}

type timeFilterStruct struct {
	from string
	to   string
}

type timeIntervalStruct struct {
	startTime time.Time
	endTime   time.Time
}

type serviceStruct struct {
	name         string
	hits         int
	timeInterval timeIntervalStruct
}

type hostStruct struct {
	name      string
	services  []serviceStruct
	hostGroup string
}
