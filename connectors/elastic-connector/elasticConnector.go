package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-elasticsearch/v7"
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

func CollectMetrics() ([]transit.MonitoredResource, []transit.InventoryResource, []transit.ResourceGroup) {
	monitoredResources, inventoryResources, resourceGroups := make(map[string]transit.MonitoredResource), make(map[string]transit.InventoryResource),
		make(map[string]transit.ResourceGroup)

	storedQueries, _ := RetrieveStoredQueries(nil)

	// TODO: these should come from environment variables
	cfg := elasticsearch.Config{
		Addresses: []string{ // TODO: multiple load balanced elastic search
			"http://localhost:9200",
			// "http://localhost:9201",
		},
	}
	esClient, _ := elasticsearch.NewClient(cfg)

	indexPatterns := make(map[string]IndexPattern)

	for _, storedQuery := range storedQueries {
		query, queryBool := make(map[string]interface{}), make(map[string]interface{})
		var must, mustNot, should, filter []interface{}

		indexSet := make(map[string]struct{})
		for _, filter := range storedQuery.Filters {
			index, queryType, negate, key := filter["index"].(string), filter["type"].(string), filter["negate"].(bool),
				filter["key"].(string)

			indexPattern := RetrieveIndexPattern(index)
			indexPatterns[indexPattern.Id] = indexPattern
			indexSet[indexPattern.Title] = struct{}{}

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
		if strings.Contains(gte, "$interval") {
			gte = strings.ReplaceAll(gte, "$interval", "5d")
		}
		if strings.Contains(lt, "$interval") {
			lt = strings.ReplaceAll(lt, "$interval", "5d")
		}
		var startTime, endTime = parseTime(gte, true), parseTime(lt, false)
		filter = append(filter, map[string]interface{}{
			"range": map[string]interface{}{
				"@timestamp": map[string]interface{}{
					"gte": gte,
					"lt":  lt,
				},
			},
		})
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

				if monitoredResource, exists := monitoredResources[hostName]; exists {
					updServices := monitoredResource.Services
					var found = false
					for _, updService := range updServices {
						if updService.Name == storedQuery.Title {
							updMetric := updService.Metrics[0]
							updValue := updMetric.Value

							doubleValue := updValue.DoubleValue + 1
							integerValue := updValue.IntegerValue + 1

							updValue.DoubleValue = doubleValue
							updValue.IntegerValue = integerValue
							updValue.StringValue = fmt.Sprintf("%d", integerValue)

							updMetric.Value = updValue
							updService.Metrics = []transit.TimeSeries{updMetric}

							found = true
							break
						}
					}
					if !found {
						hitsValue := &transit.TypedValue{
							ValueType:    transit.IntegerType,
							BoolValue:    false,
							DoubleValue:  1,
							IntegerValue: 1,
							StringValue:  "1",
							TimeValue:    nil,
						}

						var timeInterval = &transit.TimeInterval{
							StartTime: milliseconds.MillisecondTimestamp{Time: startTime},
							EndTime:   milliseconds.MillisecondTimestamp{Time: endTime},
						}

						var hitsMetric = transit.TimeSeries{
							MetricName: "hits",
							SampleType: transit.Value,
							Interval:   timeInterval,
							Value:      hitsValue,
							Tags:       nil, // TODO
							Unit:       transit.UnitCounter,
							Thresholds: nil, // TODO
						}

						var metrics = []transit.TimeSeries{hitsMetric}

						var service = transit.MonitoredService{
							Name:             storedQuery.Title,
							Type:             transit.Service,
							Owner:            "",                                                  // TODO
							Status:           transit.ServiceOk,                                   // TODO
							LastCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now()}, // TODO
							NextCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now()}, // TODO
							LastPlugInOutput: "",                                                  // TODO
							Properties:       nil,                                                 // TODO
							Metrics:          metrics,
						}

						updServices = append(updServices, service)
						monitoredResource.Services = updServices
					}
					monitoredResources[hostName] = monitoredResource
				} else {
					hitsValue := &transit.TypedValue{
						ValueType:    transit.IntegerType,
						BoolValue:    false,
						DoubleValue:  1,
						IntegerValue: 1,
						StringValue:  "1",
						TimeValue:    nil,
					}
					var timeInterval = &transit.TimeInterval{
						StartTime: milliseconds.MillisecondTimestamp{Time: startTime},
						EndTime:   milliseconds.MillisecondTimestamp{Time: endTime},
					}
					var hitsMetric = transit.TimeSeries{
						MetricName: "hits",
						SampleType: transit.Value,
						Interval:   timeInterval,
						Value:      hitsValue,
						Tags:       nil, // TODO
						Unit:       transit.UnitCounter,
						Thresholds: nil, // TODO
					}
					var service = transit.MonitoredService{
						Name:             storedQuery.Title,
						Type:             transit.Service,
						Owner:            hostName,
						Status:           transit.ServiceOk,                                   // TODO
						LastCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now()}, // TODO
						NextCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now()}, // TODO
						LastPlugInOutput: "",                                                  // TODO
						Properties:       nil,                                                 // TODO
						Metrics:          []transit.TimeSeries{hitsMetric},
					}
					var monitoredResource = transit.MonitoredResource{
						Name:             hostName,
						Type:             transit.Host,
						Owner:            "",                                                  // TODO
						Status:           transit.HostUp,                                      // TODO
						LastCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now()}, // TODO
						NextCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now()}, // TODO
						LastPlugInOutput: "",                                                  // TODO
						Properties:       nil,                                                 // TODO
						Services:         []transit.MonitoredService{service},
					}
					monitoredResources[hostName] = monitoredResource
				}

				inventoryService := transit.InventoryService{
					Name:        storedQuery.Title,
					Type:        transit.Service,
					Owner:       hostName,
					Category:    "",  //TODO
					Description: "",  //TODO
					Properties:  nil, //TODO
				}
				if inventoryResource, hostExists := inventoryResources[hostName]; hostExists {
					inventoryServices := inventoryResource.Services
					var serviceExists = false
					for _, service := range inventoryServices {
						if service.Name == inventoryService.Name {
							serviceExists = true
							break
						}
					}
					if !serviceExists {
						inventoryServices = append(inventoryServices, inventoryService)
						inventoryResource.Services = inventoryServices
						inventoryResources[hostName] = inventoryResource
					}
				} else {
					inventoryResource := transit.InventoryResource{
						Name:        hostName,
						Type:        transit.Host,
						Owner:       "",  // TODO
						Category:    "",  // TODO
						Description: "",  // TODO
						Device:      "",  // TODO
						Properties:  nil, // TODO
						Services:    []transit.InventoryService{inventoryService},
					}
					inventoryResources[hostName] = inventoryResource
				}

				resourceRef := transit.MonitoredResourceRef{
					Name:  hostName,
					Type:  transit.Host,
					Owner: "", //TODO
				}
				var hostGroup string
				if container["labels"] != nil && container["labels"].(map[string]interface{})["com_docker_compose_project"] != nil {
					hostGroup = container["labels"].(map[string]interface{})["com_docker_compose_project"].(string)
				}
				if group, groupExists := resourceGroups[hostGroup]; groupExists {
					resources := group.Resources
					resources = append(resources, resourceRef)
					group.Resources = resources
					resourceGroups[hostGroup] = group
				} else {
					group := transit.ResourceGroup{
						GroupName:   hostGroup,
						Type:        transit.HostGroup,
						Description: hostGroup,
						Resources:   []transit.MonitoredResourceRef{resourceRef},
					}
					resourceGroups[hostGroup] = group
				}
			}
		}
	}

	mrs := make([]transit.MonitoredResource, len(monitoredResources))
	i := 0
	for _, value := range monitoredResources {
		mrs[i] = value
		i++
	}
	irs := make([]transit.InventoryResource, len(inventoryResources))
	j := 0
	for _, value := range inventoryResources {
		irs[j] = value
		j++
	}
	rgs := make([]transit.ResourceGroup, len(resourceGroups))
	k := 0
	for _, value := range resourceGroups {
		rgs[k] = value
		k++
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
	expression := timeExpression[4:len(timeExpression)]
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

const (
	TypePhrase  = "phrase"
	TypePhrases = "phrases"
	TypeRange   = "range"
	TypeExists  = "exists"
)
