package main

//
//
//import (
//	"bytes"
//	"context"
//	"encoding/json"
//	"github.com/elastic/go-elasticsearch/v7"
//	"github.com/gwos/tng/log"
//	"io/ioutil"
//)
//
//var esClient *elasticsearch.Client
//
//func retrieveHits(storedQuery SavedObject) ([]Hit) {
//	if esClient == nil {
//		initEsClient()
//	}
//	if esClient == nil {
//		log.Error("cannot create ElasticSearch Client")
//		return nil
//	}
//
//	searchRequest := buildSearchRequest(storedQuery)
//	indexIds := extractIndexIds(storedQuery)
//	indexes := retrieveIndexTitles(indexIds)
//
//	var hits []Hit
//
//	offset := 0
//	perPage := 1000
//	total := 1
//	firstCall := true
//
//	var body bytes.Buffer
//	if err := json.NewEncoder(&body).Encode(searchRequest); err != nil {
//		log.Error("Error encoding Search request body: ", err)
//	}
//	log.Info(body.String())
//
//	for total > offset {
//		// TODO
//		if offset + perPage > 10000 {
//			// 400 Bad Request search_phase_execution_exceptionall shards failed
//			// elasticsearch log says: "from + size must be less than or equal to: [10000]"
//			log.Error("Total hits count is %d. Only the first 10000 will be processed.", total)
//			break
//		}
//
//		response, err := esClient.Search(
//			esClient.Search.WithContext(context.Background()),
//			esClient.Search.WithIndex(indexes...),
//			esClient.Search.WithBody(&body),
//			esClient.Search.WithTrackTotalHits(true),
//			esClient.Search.WithPretty(),
//			esClient.Search.WithSize(perPage),
//			esClient.Search.WithFrom(offset),
//		)
//
//		if err != nil {
//			log.Error("Error getting response: ", err)
//			return nil
//		}
//		if response.IsError() {
//			var e map[string]interface{}
//			if err := json.NewDecoder(response.Body).Decode(&e); err != nil {
//				log.Error("Error parsing the response body: ", err)
//			} else {
//				// Print the response status and error information.
//				log.Error(response.Status(),
//					e["error"].(map[string]interface{})["type"],
//					e["error"].(map[string]interface{})["reason"],
//				)
//			}
//			return nil
//		}
//
//		responseBody, err := ioutil.ReadAll(response.Body)
//		//s := string(responseBody)
//		//log.Info(s)
//		if err != nil {
//			log.Error(err)
//			return nil
//		}
//		var searchResponse SearchResponse
//		err = json.Unmarshal(responseBody, &searchResponse)
//		if err != nil {
//			log.Error(err)
//		}
//
//		hits = append(hits, searchResponse.Hits.Hits...)
//
//		if firstCall {
//			total = searchResponse.Hits.Total.Value
//			firstCall = false
//		}
//
//		offset = offset + perPage
//
//		if offset + perPage > total {
//			perPage = total - offset
//		}
//	}
//	return hits
//}
//
//func initEsClient()  {
//	// TODO: these should come from environment variables
//	cfg := elasticsearch.Config{
//		Addresses: []string{ // TODO: multiple load balanced elastic search
//			"http://localhost:9200",
//			// "http://localhost:9201",
//		},
//	}
//	var err error
//	esClient, err = elasticsearch.NewClient(cfg)
//	if err != nil {
//		log.Error(err)
//	}
//}
//
//type SearchRequest struct {
//	Query *Query `json:"query,omitempty"`
//}
//
//type Query struct {
//	Bool Bool `json:"bool"`
//}
//
//type Bool struct {
//	Must               []Clause `json:"must,omitempty"`
//	MustNot            []Clause `json:"must_not,omitempty"`
//	Should             []Clause `json:"should,omitempty"`
//	Filter             []Clause `json:"filter,omitempty"`
//	MinimumShouldMatch *int      `json:"minimum_should_match,omitempty"`
//}
//
//type Clause struct {
//	Match  *map[string]interface{} `json:"match,omitempty"`
//	Range  *map[string]interface{} `json:"range,omitempty"`
//	Exists *Exists                 `json:"exists,omitempty"`
//	Bool   *Bool                   `json:"bool,omitempty"`
//}
//
//type Exists struct {
//	Field string `json:"field,omitempty"`
//}
//
//type SearchResponse struct {
//	Took int  `json:"took"`
//	Hits Hits `json:"hits"`
//}
//
//type Hits struct {
//	Total TotalHits `json:"total"`
//	Hits  []Hit     `json:"hits"`
//}
//
//type Hit struct {
//	Index  string                 `json:"_index"`
//	Type   string                 `json:"_type"`
//	ID     string                 `json:"_id"`
//	Score  float64                `json:"_score"`
//	Source map[string]interface{} `json:"_source"`
//}
//
//type TotalHits struct {
//	Value int `json:"value"`
//}
