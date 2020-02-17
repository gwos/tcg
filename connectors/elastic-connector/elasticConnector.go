package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-elasticsearch/v7"
	_ "github.com/gwos/tng/milliseconds"
	"github.com/gwos/tng/transit"
	"io/ioutil"
	"log"
	"net/http"
)

const (
	KibanaApiSavedObjectsPath = "http://localhost:5601/kibana/api/saved_objects/"
)

func CollectMetrics() []transit.MonitoredResource {
	storedQueries, _ := RetrieveStoredQueries(nil)

	// TODO: these should come from environment variables
	cfg := elasticsearch.Config{
		Addresses: []string{ // TODO: multiple load balanced elastic search
			"http://localhost:9200",
			// "http://localhost:9201",
		},
	}
	esClient, _ := elasticsearch.NewClient(cfg)

	monitoredResources := make(map[string]transit.MonitoredResource)

	indexPatterns := make(map[string]IndexPattern)

	for _, storedQuery := range storedQueries {
		indexSet := make(map[string]struct{})
		query := make(map[string]interface{})
		queryBool := make(map[string]interface{})
		var must []interface{}
		var mustNot []interface{}
		var should []interface{}
		for _, filter := range storedQuery.Filters {
			index := filter["index"].(string)
			queryType := filter["type"].(string)
			negate := filter["negate"].(bool)
			key := filter["key"].(string)

			indexPattern := RetrieveIndexPattern(index)
			indexPatterns[indexPattern.Id] = indexPattern
			indexSet[indexPattern.Title] = struct{}{}
			switch queryType {
			case TypePhrase:
				q := map[string]interface{}{
					"match": map[string]interface{}{
						filter["key"].(string): filter["value"].(string),
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
						"field": filter["key"].(string),
					}}
				if !negate {
					must = append(must, q)
				} else {
					mustNot = append(mustNot, q)
				}
				break
			}
		}
		queryBool["must"] = must
		queryBool["must_not"] = mustNot
		queryBool["should"] = should
		if should != nil {
			queryBool["minimum_should_match"] = 1
		}
		query["bool"] = queryBool
		queryBody := map[string]interface{}{
			"query": query,
		}

		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(queryBody); err != nil {
			log.Fatalf("Error encoding query: %s", err)
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
			log.Fatalf("Error getting response: %s", err)
		}

		if res.IsError() {
			var e map[string]interface{}
			if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
				log.Fatalf("Error parsing the response body: %s", err)
			} else {
				// Print the response status and error information.
				log.Fatalf("[%s] %s: %s",
					res.Status(),
					e["error"].(map[string]interface{})["type"],
					e["error"].(map[string]interface{})["reason"],
				)
			}
		}

		responseBody, err := ioutil.ReadAll(res.Body)
		var result map[string]interface{}
		json.Unmarshal(responseBody, &result)
		took := result["took"].(float64)
		hits := result["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64)

		log.Print(storedQuery.Name)
		log.Print(fmt.Sprintf("%f", took))
		log.Print(fmt.Sprintf("%f", hits))

		//hitsValue := &transit.TypedValue{
		//	ValueType:    "", // TODO
		//	BoolValue:    false,
		//	DoubleValue:  hits,
		//	IntegerValue: int64(hits),
		//	StringValue:  fmt.Sprintf("%f", hits),
		//	TimeValue:    nil, // TODO
		//}
		//
		//var hitsMetric = transit.TimeSeries{
		//	MetricName: "hits", // TODO
		//	SampleType: "",     // TODO
		//	Interval:   nil,    // TODO
		//	Value:      hitsValue,
		//	Tags:       nil, // TODO
		//	Unit:       "",  // TODO
		//	Thresholds: nil, // TODO
		//}
		//
		//var tookValue = &transit.TypedValue{
		//	ValueType:    "", // TODO
		//	BoolValue:    false,
		//	DoubleValue:  took,
		//	IntegerValue: int64(took),
		//	StringValue:  fmt.Sprintf("%f", took),
		//	TimeValue:    nil, // TODO
		//}
		//
		//var tookMetric = transit.TimeSeries{
		//	MetricName: "execution_time", // TODO
		//	SampleType: "",               // TODO
		//	Interval:   nil,              // TODO
		//	Value:      tookValue,
		//	Tags:       nil,  // TODO
		//	Unit:       "ms", // TODO
		//	Thresholds: nil,  // TODO
		//}
		//
		//var metrics = []transit.TimeSeries{tookMetric, hitsMetric}
		//
		//var service = transit.MonitoredService{
		//	Name:             storedQuery.Name,
		//	Type:             transit.Service,                     // TODO
		//	Owner:            "",                                  // TODO
		//	Status:           "",                                  // TODO
		//	LastCheckTime:    milliseconds.MillisecondTimestamp{}, // TODO
		//	NextCheckTime:    milliseconds.MillisecondTimestamp{}, // TODO
		//	LastPlugInOutput: "",                                  // TODO
		//	Properties:       nil,                                 // TODO
		//	Metrics:          metrics,
		//}
		//
		//var services = []transit.MonitoredService{service}
		//for _, indexPattern := range indexPatterns {
		//	hostName := indexPattern.Id // TODO tag!!!!!
		//	if host, exists := monitoredResources[hostName]; exists {
		//		existingServices := host.Services
		//		existingServices = append(existingServices, service)
		//		host.Services = existingServices
		//		monitoredResources[hostName] = host
		//	} else {
		//		var monitoredResource = transit.MonitoredResource{
		//			Name:             hostName,
		//			Type:             transit.Host,
		//			Owner:            "",                                  // TODO
		//			Status:           "",                                  // TODO
		//			LastCheckTime:    milliseconds.MillisecondTimestamp{}, // TODO
		//			NextCheckTime:    milliseconds.MillisecondTimestamp{}, // TODO
		//			LastPlugInOutput: "",                                  // TODO
		//			Properties:       nil,                                 // TODO
		//			Services:         services,
		//		}
		//		monitoredResources[hostName] = monitoredResource
		//	}
		//}
	}

	result := make([]transit.MonitoredResource, len(monitoredResources))
	i := 0
	for _, value := range monitoredResources {
		result[i] = value
		i++
	}
	return result
}

func RetrieveStoredQueries(ids []string) ([]StoredQuery, int) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := http.Client{Transport: tr}
	var request *http.Request
	var response *http.Response
	var err error

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

	request, err = http.NewRequest(http.MethodGet, KibanaApiSavedObjectsPath+"_find?type=query"+search, nil)
	if err != nil {
		log.Fatalf("Error getting response: %s", err)
	}
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("kbn-xsrf", "true")

	response, err = client.Do(request)
	if response.StatusCode == 400 {
		log.Fatalf("Not Found!")
	}
	responseBody, err := ioutil.ReadAll(response.Body)
	response.Body.Close()

	var result map[string]interface{}
	json.Unmarshal(responseBody, &result)
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
		storedQueries = append(storedQueries, StoredQuery{id, name, description, filters})
	}

	return storedQueries, len(storedQueries)
}

func RetrieveIndexPattern(id string) IndexPattern {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := http.Client{Transport: tr}
	var request *http.Request
	var response *http.Response
	var err error

	request, err = http.NewRequest(http.MethodGet, KibanaApiSavedObjectsPath+"index-pattern/"+id, nil)
	if err != nil {
		log.Fatalf("Error getting response: %s", err)
	}
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("kbn-xsrf", "true")

	response, err = client.Do(request)
	if response.StatusCode == 400 {
		log.Fatalf("Not Found!")
	}
	responseBody, err := ioutil.ReadAll(response.Body)
	response.Body.Close()

	var result map[string]interface{}
	json.Unmarshal(responseBody, &result)

	title := result["attributes"].(map[string]interface{})["title"].(string)
	return IndexPattern{id, title}
}

type StoredQuery struct {
	Id          string
	Name        string
	Description string
	Filters     []map[string]interface{}
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
