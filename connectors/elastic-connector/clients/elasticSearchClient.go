package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/rs/zerolog/log"
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
		log.Err(err).Msg("could not create ES client")
	} else {
		esClient.Client = client
	}
	return err
}

func (esClient *EsClient) GetHosts(hostField string, hostGroupField *string) ([]EsAggregationKey, error) {
	searchBody := EsSearchBody{
		Aggs: BuildAggregationsByHostNameAndHostGroup(hostField, hostGroupField),
	}
	response, err := esClient.doSearchRequest(searchBody, nil)
	if err != nil {
		return nil, err
	}
	searchResponse := parseSearchResponse(response)
	if searchResponse == nil {
		log.Error().Msg("could not get hosts: response is nil")
		return nil, nil
	}

	var keys []EsAggregationKey
	if searchResponse.Aggregations.Aggregation.Buckets != nil {
		for _, bucket := range searchResponse.Aggregations.Aggregation.Buckets {
			keys = append(keys, bucket.Key)
		}
	}

	afterKey := getAfterKey(searchResponse)
	for afterKey != nil {
		searchBody.Aggs.Agg.Composite.After = afterKey
		response, err = esClient.doSearchRequest(searchBody, nil)
		searchResponse := parseSearchResponse(response)
		if searchResponse == nil {
			log.Error().Msg("could not get hosts: response is nil")
			break
		}
		if searchResponse.Aggregations.Aggregation.Buckets != nil {
			for _, bucket := range searchResponse.Aggregations.Aggregation.Buckets {
				keys = append(keys, bucket.Key)
			}
		}
		afterKey = getAfterKey(searchResponse)
	}

	return keys, err
}

func (esClient EsClient) CountHits(hostField string, indexes []string, query *EsQuery) (map[string]int, error) {
	searchBody := EsSearchBody{
		Query: query,
		Aggs:  BuildAggregationsByHostNameAndHostGroup(hostField, nil),
	}

	response, err := esClient.doSearchRequest(searchBody, indexes)
	if err != nil {
		return nil, err
	}
	searchResponse := parseSearchResponse(response)
	if searchResponse == nil {
		log.Error().Msg("could not count hits: response is nil")
		return nil, nil
	}

	result := make(map[string]int)
	if searchResponse.Aggregations.Aggregation.Buckets != nil {
		for _, bucket := range searchResponse.Aggregations.Aggregation.Buckets {
			result[bucket.Key.Host] = bucket.DocsCount
		}
	}

	afterKey := getAfterKey(searchResponse)
	for afterKey != nil {
		searchBody.Aggs.Agg.Composite.After = afterKey
		response, err = esClient.doSearchRequest(searchBody, indexes)
		searchResponse := parseSearchResponse(response)
		if searchResponse == nil {
			log.Error().Msg("could not count hits: response is nil")
			break
		}
		if searchResponse.Aggregations.Aggregation.Buckets != nil {
			for _, bucket := range searchResponse.Aggregations.Aggregation.Buckets {
				result[bucket.Key.Host] = bucket.DocsCount
			}
		}
		afterKey = getAfterKey(searchResponse)
	}

	return result, err
}

func (esClient EsClient) CountHitsForHost(hostName string, hostNameField string, indexes []string, query *EsQuery) (int, error) {
	queryCopy := copyQuery(query)
	queryCopy.Bool.Filter = append(queryCopy.Bool.Filter, buildMatchPhraseFilter(hostNameField, hostName))
	searchBody := EsSearchBody{
		Query: queryCopy,
	}
	response, err := esClient.doSearchRequest(searchBody, indexes)
	if err != nil {
		return 0, err
	}
	searchResponse := parseSearchResponse(response)
	if searchResponse == nil {
		log.Error().Msg("could not count hits: response is nil")
		return 0, nil
	}
	return searchResponse.Hits.Total.Value, nil
}

func getAfterKey(searchResponse *EsSearchResponse) *EsAggregationKey {
	var afterKey *EsAggregationKey
	if searchResponse != nil {
		if searchResponse.Aggregations.Aggregation.AfterKey != nil {
			afterKey = searchResponse.Aggregations.Aggregation.AfterKey
		} else {
			if searchResponse.Aggregations.Aggregation.Buckets != nil && len(searchResponse.Aggregations.Aggregation.Buckets) > 0 {
				afterKey = &searchResponse.Aggregations.Aggregation.Buckets[len(searchResponse.Aggregations.Aggregation.Buckets)-1].Key
			}
		}
	}
	return afterKey
}

func (esClient EsClient) doSearchRequest(searchBody EsSearchBody, indexes []string) (*esapi.Response, error) {
	if esClient.Client == nil {
		err := esClient.InitEsClient()
		if err != nil {
			log.Err(err).Msg("ES client was not initialized")
			return nil, err
		}
	}
	client := esClient.Client
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(searchBody); err != nil {
		log.Err(err).Msg("could not encode ES Search Body")
		return nil, nil
	}

	log.Debug().
		Bytes("body", body.Bytes()).
		Msg("performing ES search request with body")

	response, err := client.Search(
		client.Search.WithContext(context.Background()),
		client.Search.WithIndex(indexes...),
		client.Search.WithBody(&body),
		client.Search.WithTrackTotalHits(true),
		client.Search.WithSize(0),
	)

	if err != nil {
		log.Err(err).Msg("could not get Search response")
		return nil, nil
	}

	return response, nil
}

func parseSearchResponse(response *esapi.Response) *EsSearchResponse {
	if response == nil {
		log.Error().Msg("ES Search response is nil")
		return nil
	}

	log.Debug().
		Str("response", response.String()).
		Msg("ES Search response")

	if response.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(response.Body).Decode(&e); err != nil {
			log.Err(err).Msg("could not parse Search response")
		} else {
			// Print the response status and error information.
			log.Error().Msgf("response is error: %s: %s: %s",
				response.Status(),
				e["error"].(map[string]interface{})["type"],
				e["error"].(map[string]interface{})["reason"],
			)
		}
		return nil
	}

	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Err(err).Msg("could not read ES Search response")
		return nil
	}

	defer response.Body.Close()

	var searchResponse EsSearchResponse
	if err := json.Unmarshal(responseBody, &searchResponse); err != nil {
		log.Err(err).Msg("could not parse ES Search response")
		return nil
	}

	return &searchResponse
}

func (esClient EsClient) IsAggregatable(fieldNames []string, indexes []string) (map[string]bool, error) {
	result := make(map[string]bool)
	for _, fieldName := range fieldNames {
		result[fieldName] = false
	}

	if esClient.Client == nil {
		err := esClient.InitEsClient()
		if err != nil {
			log.Err(err).Msg("ES client was not initialized")
			return result, err
		}
	}
	client := esClient.Client

	log.Debug().
		Strs("fieldNames", fieldNames).
		Msg("Performing ES FieldCaps request for fields")

	response, err := client.FieldCaps(
		client.FieldCaps.WithFields(fieldNames...),
		client.FieldCaps.WithIndex(),
	)
	if err != nil {
		log.Err(err).Msg("could not get ES FieldCaps response")
		return result, nil
	}

	if response == nil {
		log.Error().Msg("ES FieldCaps response is nil")
		return result, nil
	}

	log.Debug().
		Str("response", response.String()).
		Msg("ES FieldCaps response")

	if response.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(response.Body).Decode(&e); err != nil {
			log.Err(err).Msg("could not parse ES FieldCaps response")
		} else {
			// Print the response status and error information.
			log.Error().Msgf("response is error: %s: %s: %s",
				response.Status(),
				e["error"].(map[string]interface{})["type"],
				e["error"].(map[string]interface{})["reason"],
			)
		}
		return result, nil
	}

	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Err(err).Msg("could not read ES FieldCaps response")
		return result, nil
	}

	defer response.Body.Close()

	var fieldCapsResponse EsFieldCapsResponse
	if err := json.Unmarshal(responseBody, &fieldCapsResponse); err != nil {
		log.Err(err).Msg("could not parse ES FieldCaps response")
		return result, nil
	}

	// TODO improve this parsing once link metrics to index patterns
	if fieldCapsResponse.Fields != nil {
		for _, fieldName := range fieldNames {
			if field, exists := fieldCapsResponse.Fields[fieldName]; exists {
				switch field.(type) {
				case map[string]interface{}:
					fieldCaps := field.(map[string]interface{})
					for _, v := range fieldCaps {
						switch v.(type) {
						case map[string]interface{}:
							fieldCap := v.(map[string]interface{})
							if aggregatable, exists := fieldCap["aggregatable"]; exists {
								switch aggregatable.(type) {
								case bool:
									if aggregatable.(bool) {
										result[fieldName] = true
										break
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return result, nil
}
