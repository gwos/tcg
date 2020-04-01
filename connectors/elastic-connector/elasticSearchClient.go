package main

import (
	"encoding/json"
	"github.com/gwos/tng/clients"
	"github.com/gwos/tng/log"
	"net/http"
)

const (
	defaultElasticApiPath = "http://localhost:9200/"
	searchPath            = "_search"
)

var elasticHeaders = map[string]string{
	"Content-Type": "application/json",
}

func retrieveHits(storedQuery SavedObject) []Hit {
	searchRequest := buildSearchRequest(storedQuery, true)
	// indexes are presented in query's filters with their ids, but for search request we need their titles
	indexIds := extractIndexIds(storedQuery)
	indexes := retrieveIndexTitles(indexIds)
	path := buildSearchPath(indexes)

	var hits []Hit

	step := 1
	searchResponse := executeSearchRequest(searchRequest, path)
	if searchResponse == nil {
		return nil
	}
	hits = append(hits, searchResponse.Hits.Hits...)
	total := searchResponse.Hits.Total.Value

	allSuccessful := true
	for total > (*searchRequest.Size * step) {
		step = step + 1
		lastId := searchResponse.Hits.Hits[len(searchResponse.Hits.Hits)-1].ID
		searchRequest.SearchAfter = append(searchRequest.SearchAfter, lastId)
		searchResponse := executeSearchRequest(searchRequest, path)
		if searchResponse == nil {
			allSuccessful = false
			continue
		}
		hits = append(hits, searchResponse.Hits.Hits...)
	}

	if !allSuccessful && hits != nil {
		log.Error("Failed to extract some of Hits. The result is probably incomplete.")
	}

	return hits
}

func buildSearchPath(indexes []string) string {
	var indexesPath string
	if indexes != nil && len(indexes) > 0 {
		for i, index := range indexes {
			indexesPath = indexesPath + index
			if i != len(indexes)-1 {
				indexesPath = indexesPath + ","
			} else {
				indexesPath = indexesPath + "/"
			}
		}
	}
	return getElasticApiPath() + indexesPath + searchPath
}

func executeSearchRequest(searchRequest SearchRequest, path string) *SearchResponse {
	bodyBytes, err := json.Marshal(searchRequest)
	if err != nil {
		log.Error("Error marshalling Search request body: ", err)
		return nil
	}

	status, response, err := clients.SendRequest(http.MethodGet, path, elasticHeaders, nil, bodyBytes)
	if err != nil || status != 200 || response == nil {
		if err != nil {
			log.Error(err)
		}
		if status != 200 {
			log.Error("Failure response code: ", status)
		}
		if response == nil {
			log.Error("Search response is nil.")
		}
		return nil
	}

	var searchResponse SearchResponse
	err = json.Unmarshal(response, &searchResponse)
	if err != nil {
		log.Error("Error parsing Search response: ", err)
		return nil
	}

	return &searchResponse
}

func getElasticApiPath() string {
	// TODO get from environment variables if not set return default
	return defaultElasticApiPath
}

type SearchRequest struct {
	TrackTotalHits *bool               `json:"track_total_hits,omitempty"`
	Size           *int                `json:"size,omitempty"`
	Query          *Query              `json:"query,omitempty"`
	Sort           []map[string]string `json:"sort,omitempty"`
	SearchAfter    []interface{}       `json:"search_after,omitempty"`
}

type Query struct {
	Bool Bool `json:"bool"`
}

type Bool struct {
	Must               []Clause `json:"must,omitempty"`
	MustNot            []Clause `json:"must_not,omitempty"`
	Should             []Clause `json:"should,omitempty"`
	Filter             []Clause `json:"filter,omitempty"`
	MinimumShouldMatch *int     `json:"minimum_should_match,omitempty"`
}

type Clause struct {
	Match  *map[string]interface{} `json:"match,omitempty"`
	Range  *map[string]interface{} `json:"range,omitempty"`
	Exists *Exists                 `json:"exists,omitempty"`
	Bool   *Bool                   `json:"bool,omitempty"`
}

type Exists struct {
	Field string `json:"field,omitempty"`
}

type SearchResponse struct {
	Took int  `json:"took"`
	Hits Hits `json:"hits"`
}

type Hits struct {
	Total TotalHits `json:"total"`
	Hits  []Hit     `json:"hits"`
}

type Hit struct {
	Index  string                 `json:"_index"`
	Type   string                 `json:"_type"`
	ID     string                 `json:"_id"`
	Score  float64                `json:"_score"`
	Source map[string]interface{} `json:"_source"`
}

type TotalHits struct {
	Value int `json:"value"`
}
