package main

import (
	"bytes"
	"encoding/json"
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
	searchRequest := buildSearchRequest(storedQuery)
	// indexes are presented in query's filters with their ids, but for search request we need their titles
	indexIds := extractIndexIds(storedQuery)
	indexes := retrieveIndexTitles(indexIds)
	path := buildSearchPath(indexes)

	var hits []Hit

	offset := 0
	perPage := 1000
	total := -1

	trackTotalHits := true
	searchRequest.TrackTotalHits = &trackTotalHits

	allSuccessful := true
	for total == -1 || total > offset {
		searchRequest.Size = &perPage
		searchRequest.From = &offset

		var body bytes.Buffer
		if err := json.NewEncoder(&body).Encode(searchRequest); err != nil {
			log.Error("Error encoding Search request body: ", err)
			return nil
		}
		log.Info(body.String())
		response, successful := executeRequest(http.MethodGet, path, &body, elasticHeaders)
		if !successful {
			if total != -1 {
				// previous calls were fine let's try to get remaining data
				allSuccessful = false
				offset = offset + perPage
				if offset+perPage > total {
					perPage = total - offset
				}
				continue
			} else {
				log.Error("Search failed.")
				return nil
			}
		}

		var searchResponse SearchResponse
		err := json.Unmarshal(response, &searchResponse)
		if err != nil {
			log.Error("Error parsing Search response: ", err)
			if total != -1 {
				// previous parsings were fine let's try to get remaining data
				allSuccessful = false
				offset = offset + perPage
				if offset+perPage > total {
					perPage = total - offset
				}
				continue
			} else {
				log.Error("Search failed.")
				return nil
			}
		}

		hits = append(hits, searchResponse.Hits.Hits...)

		total = searchResponse.Hits.Total.Value
		offset = offset + perPage
		if offset+perPage > total {
			perPage = total - offset
		}
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

func getElasticApiPath() string {
	// TODO get from environment variables if not set return default
	return defaultElasticApiPath
}

type SearchRequest struct {
	TrackTotalHits *bool  `json:"track_total_hits,omitempty"`
	Size           *int   `json:"size,omitempty"`
	From           *int   `json:"from,omitempty"`
	Query          *Query `json:"query,omitempty"`
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
