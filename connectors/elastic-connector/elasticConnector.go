package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
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
	KibanaApiSavedObjectsPath = "http://localhost:5601/kibana/api/saved_objects/"
)

const (
	TypePhrase  = "phrase"
	TypePhrases = "phrases"
	TypeRange   = "range"
	TypeExists  = "exists"
)

const defaultInterval = 5

func CollectMetrics() ([]transit.MonitoredResource, []transit.InventoryResource, []transit.ResourceGroup) {
	hosts, groups := make(map[string]Host), make(map[string][]string)

	// TODO: selected queries titles and time intervals will be retrieved from stored configs, here only queries filters will be retrieved
	storedQueries, _ := RetrieveStoredQueries(nil)

	// TODO: these should come from environment variables
	cfg := elasticsearch.Config{
		Addresses: []string{ // TODO: multiple load balanced elastic search
			"http://localhost:9200",
			// "http://localhost:9201",
		},
	}
	esClient, _ := elasticsearch.NewClient(cfg)

	for _, storedQuery := range storedQueries {
		query, queryBool, indexesSet := make(map[string]interface{}), make(map[string]interface{}), make(map[string]struct{})
		var must, mustNot, should, filter []interface{}

		for _, filter := range storedQuery.Filters {
			index, queryType, negate, key := filter["index"].(string), filter["type"].(string), filter["negate"].(bool), filter["key"].(string)

			indexPattern := RetrieveIndexPattern(index)
			indexesSet[indexPattern.Title] = struct{}{}

			switch queryType {
			case TypePhrase:
				q := map[string]interface{}{
					"match": map[string]interface{}{
						key: filter["value"].(string),
					}}
				if !negate {
					must = append(must, q)
				} else {
					mustNot = append(mustNot, q)
				}
				break
			case TypePhrases:
				params := filter["params"].([]interface{})
				for _, param := range params {
					param := param.(string)
					if !negate {
						should = append(should, map[string]interface{}{
							"match": map[string]interface{}{
								key: param,
							}})
					} else {
						should = append(should, map[string]interface{}{
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
			case TypeRange:
				params := filter["params"].(map[string]interface{})
				r := map[string]interface{}{
					"range": map[string]interface{}{
						key: params,
					}}
				if !negate {
					must = append(must, r)
				} else {
					mustNot = append(mustNot, r)
				}
				break
			case TypeExists:
				q := map[string]interface{}{
					"exists": map[string]interface{}{
						"field": key,
					}}
				if !negate {
					must = append(must, q)
				} else {
					mustNot = append(mustNot, q)
				}
				break
			}
		}
		var gte, lt = storedQuery.TimeFilter.From, storedQuery.TimeFilter.To
		interval := strconv.Itoa(defaultInterval) + "m"
		if strings.Contains(gte, "$interval") {
			gte = strings.ReplaceAll(gte, "$interval", interval)
		}
		if strings.Contains(lt, "$interval") {
			lt = strings.ReplaceAll(lt, "$interval", interval)
		}
		filter = append(filter, map[string]interface{}{
			"range": map[string]interface{}{
				"@timestamp": map[string]interface{}{
					"gte": gte,
					"lt":  lt,
				},
			},
		})

		var startTime, endTime = parseTime(gte, true), parseTime(lt, false)
		timeInterval := TimeInterval{
			StartTime: startTime,
			EndTime:   endTime,
		}

		queryBool["must"], queryBool["must_not"], queryBool["should"], queryBool["filter"] = must, mustNot, should, filter
		if should != nil {
			queryBool["minimum_should_match"] = 1
		}
		query["bool"] = queryBool
		queryBody := map[string]interface{}{
			"query": query,
		}

		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(queryBody); err != nil {
			log.Error("Error encoding query: %s", err)
		}

		var indexes []string
		for index := range indexesSet {
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
		var result map[string]interface{}
		err = json.Unmarshal(responseBody, &result)
		if err != nil {
			log.Error(err.Error())
		}

		hits := result["hits"].(map[string]interface{})["hits"].([]interface{})
		for _, h := range hits {
			hit := h.(map[string]interface{})
			if hit["_source"].(map[string]interface{})["container"] != nil {
				container := hit["_source"].(map[string]interface{})["container"].(map[string]interface{})

				hostName := container["name"].(string)

				var hostGroup string
				if container["labels"] != nil && container["labels"].(map[string]interface{})["com_docker_compose_project"] != nil {
					hostGroup = container["labels"].(map[string]interface{})["com_docker_compose_project"].(string)
				}
				if hosts, exists := groups[hostGroup]; exists {
					hosts = append(hosts, hostName)
					groups[hostGroup] = hosts
				} else {
					groups[hostGroup] = []string{hostName}
				}

				if host, exists := hosts[hostName]; exists {
					services := host.Services
					var found = false
					for _, service := range services {
						if service.Name == storedQuery.Title {
							service.Hits = service.Hits + 1
							found = true
							break
						}
					}
					if !found {
						service := Service{Name: storedQuery.Title, Hits: 1, TimeInterval: timeInterval}
						services = append(services, service)
						host.Services = services
					}
					hosts[hostName] = host
				} else {
					service := Service{Name: storedQuery.Title, Hits: 1, TimeInterval: timeInterval}
					host := Host{Name: hostName, Services: []Service{service}, HostGroup: hostGroup}
					hosts[hostName] = host
				}
			}
		}
	}

	mrs := make([]transit.MonitoredResource, len(hosts))
	irs := make([]transit.InventoryResource, len(hosts))
	i := 0
	for _, host := range hosts {
		monitoredServices := make([]transit.MonitoredService, len(host.Services))
		inventoryServices := make([]transit.InventoryService, len(host.Services))

		for i, service := range host.Services {
			timeInterval := &transit.TimeInterval{
				EndTime:   milliseconds.MillisecondTimestamp{Time: service.TimeInterval.EndTime},
				StartTime: milliseconds.MillisecondTimestamp{Time: service.TimeInterval.StartTime},
			}
			metric, _ := connectors.CreateMetric("hits", service.Hits, timeInterval, transit.UnitCounter)
			monitoredService, _ := connectors.CreateService(service.Name, host.Name, []transit.TimeSeries{*metric})
			inventoryService := connectors.CreateInventoryService(service.Name, host.Name)
			monitoredServices[i] = *monitoredService
			inventoryServices[i] = inventoryService
		}
		monitoredResource, _ := connectors.CreateResource(host.Name, monitoredServices)
		inventoryResource := connectors.CreateInventoryResource(host.Name, inventoryServices)
		mrs[i] = *monitoredResource
		irs[i] = inventoryResource
		i++
	}

	rgs := make([]transit.ResourceGroup, len(groups))
	j := 0
	for group, hostsInGroup := range groups {
		monitoredResourceRefs := make([]transit.MonitoredResourceRef, len(hostsInGroup))
		for i, host := range hostsInGroup {
			monitoredResourceRef := connectors.CreateMonitoredResourceRef(host, "", transit.Host)
			monitoredResourceRefs[i] = monitoredResourceRef
		}
		resourceGroup := connectors.CreateResourceGroup(group, group, transit.HostGroup, monitoredResourceRefs)
		rgs[j] = resourceGroup
		j++
	}

	return mrs, irs, rgs
}

func RetrieveStoredQueries(ids []string) ([]StoredQuery, int) {
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

	var result = RetrieveSavedObjects("_find?type=query"+search, nil)
	savedObjects := result["saved_objects"].([]interface{})
	var storedQueries []StoredQuery
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
		var tFilter TimeFilter
		if savedObject["attributes"].(map[string]interface{})["timefilter"] != nil {
			timeFilter := savedObject["attributes"].(map[string]interface{})["timefilter"].(map[string]interface{})
			tFilter.From = timeFilter["from"].(string)
			tFilter.To = timeFilter["to"].(string)
		} else {
			tFilter.From = "now-$interval"
			tFilter.To = "now"
		}
		storedQueries = append(storedQueries, StoredQuery{id, name, description,
			tFilter, filters})
	}

	return storedQueries, len(storedQueries)
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
	period := expression[len(expression)-1 : len(expression)]
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

func RetrieveIndexPattern(id string) IndexPattern {
	var result = RetrieveSavedObjects("index-pattern/"+id, nil)
	title := result["attributes"].(map[string]interface{})["title"].(string)
	return IndexPattern{id, title}
}

func RetrieveSavedObjects(path string, body io.Reader) map[string]interface{} {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := http.Client{Transport: tr}
	var request *http.Request
	var response *http.Response
	var err error

	request, err = http.NewRequest(http.MethodGet, KibanaApiSavedObjectsPath+path, body)
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

type StoredQuery struct {
	Id          string
	Title       string
	Description string
	TimeFilter  TimeFilter
	Filters     []map[string]interface{}
}

type TimeFilter struct {
	From string
	To   string
}

type IndexPattern struct {
	Id    string
	Title string
}

type TimeInterval struct {
	StartTime time.Time
	EndTime   time.Time
}

type Service struct {
	Name         string
	Hits         int
	TimeInterval TimeInterval
}

type Host struct {
	Name      string
	Services  []Service
	HostGroup string
}
