package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/gwos/tcg/log"
	"io/ioutil"
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

func (esClient *EsClient) GetHosts(hostField string, hostGroupField string) []HostBucket {
	if esClient.Client == nil {
		err := esClient.InitEsClient()
		if err != nil {
			log.Error("Failed to retrieve hosts from ElasticSearch. ElasticSearch client not initialized")
			return nil
		}
	}
	client := esClient.Client

	var aggregationsBody AggregationsBody
	aggregationsBody.byHostNameAndHostGroup(hostField, hostGroupField)

	var body bytes.Buffer
	var err error
	if err = json.NewEncoder(&body).Encode(aggregationsBody); err != nil {
		log.Error("Failed to retrieve hosts from ElasticSearch. Error encoding request body: ", err)
		return nil
	}

	log.Debug("Retrieving hosts from ElasticSearch. Performing ES search request with body: ", body.String())
	response, err := client.Search(
		client.Search.WithContext(context.Background()),
		client.Search.WithSize(0),
		client.Search.WithBody(&body),
	)

	if err != nil {
		log.Error("Failed to retrieve hosts from ElasticSearch. Error getting response: ", err)
		return nil
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
		return nil
	}

	responseBody, err := ioutil.ReadAll(response.Body)

	if err != nil {
		log.Error("Failed to retrieve hosts from ElasticSearch. Error reading response body: ", err)
		return nil
	}
	defer response.Body.Close()

	var aggregations AggregationsResponse
	err = json.Unmarshal(responseBody, &aggregations)
	if err != nil {
		log.Error("Error parsing ES Search response body: ", err)
		return nil
	}

	return aggregations.Aggregations.HostBuckets.Buckets
}

func (esClient EsClient) CountHitsForHost(hostName string, hostNameField string, indexes []string, query *Query) (int, error) {
	if esClient.Client == nil {
		err := esClient.InitEsClient()
		if err != nil {
			log.Error("ES client was not initialized")
			return 0, err
		}
	}
	client := esClient.Client

	searchBody := SearchBody{
		Query: copyQuery(query),
	}
	searchBody.ForHost(hostName, hostNameField)

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(searchBody); err != nil {
		log.Error("Error encoding ES Search Body: ", err)
		return 0, nil
	}

	log.Debug("Performing ES search request with body: ", body.String())
	var response *esapi.Response
	response, err := client.Search(
		client.Search.WithContext(context.Background()),
		client.Search.WithIndex(indexes...),
		client.Search.WithBody(&body),
		client.Search.WithTrackTotalHits(true),
		client.Search.WithSize(0),
	)

	if err != nil {
		log.Error("Error getting Search response: ", err)
		return 0, nil
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
		return 0, nil
	}

	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Error("Error reading ES Search response body: ", err)
		return 0, nil
	}

	defer response.Body.Close()

	var searchResponse SearchResponse
	if err := json.Unmarshal(responseBody, &searchResponse); err != nil {
		log.Error("Error parsing ES Search response body: ", err)
		return 0, nil
	}

	return searchResponse.Hits.Total.Value, nil
}

// ElasticSearch _search aggregated by hostname keyword request
type AggregationsBody struct {
	Aggs struct {
		ByHostname struct {
			Terms struct {
				Field string `json:"field"`
			} `json:"terms"`
			Aggs *AggsByHostGroup `json:"aggs,omitempty"`
		} `json:"_by_hostname"`
	} `json:"aggs"`
}

type AggsByHostGroup struct {
	ByHostgroup struct {
		Terms struct {
			Field string `json:"field"`
		} `json:"terms"`
	} `json:"_by_hostgroup"`
}

func (body *AggregationsBody) byHostNameAndHostGroup(hostField string, hostGroupField string) {
	body.Aggs.ByHostname.Terms.Field = hostField
	if hostGroupField != "" {
		var aggsByHostGroup AggsByHostGroup
		aggsByHostGroup.ByHostgroup.Terms.Field = hostGroupField
		body.Aggs.ByHostname.Aggs = &aggsByHostGroup
	}
}

// ElasticSearch _search aggregated by hostname keyword response
type AggregationsResponse struct {
	Aggregations struct {
		HostBuckets struct {
			Buckets []HostBucket `json:"buckets"`
		} `json:"_by_hostname"`
	} `json:"aggregations"`
}

type HostBucket struct {
	Key              string            `json:"key"`
	HostGroupBuckets *HostGroupBuckets `json:"_by_hostgroup,omitempty"`
}

type HostGroupBuckets struct {
	Buckets []struct {
		Key string `json:"key"`
	} `json:"buckets"`
}

type SearchResponse struct {
	Hits struct {
		Total struct {
			Value int `json:"value"`
		} `json:"total"`
	} `json:"hits"`
}
