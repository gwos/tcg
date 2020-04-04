package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/gwos/tng/log"
	"io/ioutil"
)

func retrieveHits(servers []string, indexes []string, storedQuery SavedObject) ([]Hit, error) {
	cfg := elasticsearch.Config{
		Addresses: servers,
	}
	esClient, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Error(err)
		return nil, errors.New("could not initialize ES client")
	}

	searchBody := buildSearchBody(storedQuery)

	var hits []Hit

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(searchBody); err != nil {
		log.Error("Error encoding Search request body: ", err)
	}
	log.Info(body.String())

	step := 1

	searchResponse := performSearch(esClient, indexes, searchBody)
	if searchResponse == nil {
		return nil, nil
	}
	hits = append(hits, searchResponse.Hits.Hits...)
	total := searchResponse.Hits.Total.Value
	allSuccessful := true
	for total > (step * perPage) {
		step = step + 1
		lastId1 := searchResponse.Hits.Hits[len(searchResponse.Hits.Hits)-1].ID
		log.Info(lastId1)
		lastId := hits[len(hits)-1].ID
		setSingleSearchAfter(lastId, &searchBody)
		searchResponse := performSearch(esClient, indexes, searchBody)
		if searchResponse == nil {
			allSuccessful = false
			continue
		}
		hits = append(hits, searchResponse.Hits.Hits...)
	}

	if !allSuccessful && hits != nil {
		log.Error("Failed to extract some of Hits. The result is probably incomplete.")
	}
	return hits, nil
}

func performSearch(esClient *elasticsearch.Client, indexes []string, searchBody SearchBody) *SearchResponse {
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(searchBody); err != nil {
		log.Error("Error encoding Search Body: ", err)
		return nil
	}
	log.Info(body.String())

	response, err := esClient.Search(
		esClient.Search.WithContext(context.Background()),
		esClient.Search.WithIndex(indexes...),
		esClient.Search.WithBody(&body),
		esClient.Search.WithTrackTotalHits(trackTotalHits),
		esClient.Search.WithPretty(),
		esClient.Search.WithSize(perPage),
		esClient.Search.WithFrom(from),
		esClient.Search.WithSort(sortById),
	)

	if err != nil {
		log.Error("Error getting Search response: ", err)
		return nil
	}

	if response.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(response.Body).Decode(&e); err != nil {
			log.Error("Error parsing the response body: ", err)
		} else {
			// Print the response status and error information.
			log.Error(response.Status(),
				e["error"].(map[string]interface{})["type"],
				e["error"].(map[string]interface{})["reason"],
			)
		}
		return nil
	}

	responseBody, err := ioutil.ReadAll(response.Body)
	str := string(responseBody)
	if str == "" {
		log.Error("test")
	}

	if err != nil {
		log.Error("Error reading Search response body: ", err)
		return nil
	}

	var searchResponse SearchResponse
	err = json.Unmarshal(responseBody, &searchResponse)
	if err != nil {
		log.Error("Error parsing Search response body: ", err)
		return nil
	}

	return &searchResponse
}
