package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/gwos/tcg/log"
	"io/ioutil"
)

const (
	trackTotalHits = true
	perPage        = 10000
	sortById       = "_id:asc"
)

type EsClient struct {
	Servers []string
	Client  *elasticsearch.Client
}

func (esClient *EsClient) InitEsClient() error {
	cfg := elasticsearch.Config{
		Addresses: esClient.Servers,
	}
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Error(err)
	} else {
		esClient.Client = client
	}
	return err
}

// Retrieves all documents matching query's filters
// Error is being returned only in case of non-working client
func (esClient EsClient) RetrieveHits(indexes []string, storedQuery SavedObject) ([]Hit, error) {
	var hits []Hit

	var searchBody SearchBody
	searchBody.FromStoredQuery(storedQuery)

	step := 1
	searchResponse, err := esClient.retrieveSingleSearchWindow(indexes, searchBody)
	if err != nil {
		return nil, err
	}
	if searchResponse == nil {
		return nil, nil
	}
	hits = append(hits, searchResponse.Hits.Hits...)

	total := searchResponse.Hits.Total.Value
	for total > (step * perPage) {
		step = step + 1
		lastId := hits[len(hits)-1].ID
		searchBody.WithSingleSearchAfter(lastId)
		searchResponse, err := esClient.retrieveSingleSearchWindow(indexes, searchBody)
		if err != nil {
			return nil, err
		}
		if searchResponse == nil {
			log.Error("Failed to extract remaining Hits. The result is incomplete.")
			return hits, nil
		}
		hits = append(hits, searchResponse.Hits.Hits...)
	}

	return hits, nil
}

// Retrieves single window of documents matching query's filters
// Error is being returned only in case of non-working client
func (esClient EsClient) retrieveSingleSearchWindow(indexes []string, searchBody SearchBody) (*SearchResponse, error) {
	if esClient.Client == nil {
		err := esClient.InitEsClient()
		if err != nil {
			log.Error("ES client was not initialized")
			return nil, err
		}
	}
	client := esClient.Client

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(searchBody); err != nil {
		log.Error("Error encoding ES Search Body: ", err)
		return nil, nil
	}

	log.Debug("Performing ES search request with body: ", body.String())
	response, err := client.Search(
		client.Search.WithContext(context.Background()),
		client.Search.WithIndex(indexes...),
		client.Search.WithBody(&body),
		client.Search.WithTrackTotalHits(trackTotalHits),
		client.Search.WithSize(perPage),
		client.Search.WithSort(sortById),
	)

	if err != nil {
		log.Error("Error getting Search response: ", err)
		return nil, nil
	}

	log.Debug("ES Search response: ", response)

	if response.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(response.Body).Decode(&e); err != nil {
			log.Error("Error parsing Search response body: ", err)
		} else {
			// Print the response status and error information.
			log.Error(response.Status(),
				e["error"].(map[string]interface{})["type"],
				e["error"].(map[string]interface{})["reason"],
			)
		}
		return nil, nil
	}

	responseBody, err := ioutil.ReadAll(response.Body)

	if err != nil {
		log.Error("Error reading ES Search response body: ", err)
		return nil, nil
	}

	var searchResponse SearchResponse
	err = json.Unmarshal(responseBody, &searchResponse)
	if err != nil {
		log.Error("Error parsing ES Search response body: ", err)
		return nil, nil
	}

	return &searchResponse, nil
}
