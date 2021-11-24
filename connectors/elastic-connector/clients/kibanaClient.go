package clients

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gwos/tcg/sdk/clients"
	"github.com/rs/zerolog/log"
)

const (
	apiPath          = "api/"
	savedObjectsPath = "saved_objects/"
	findPath         = "_find"
	bulkGetPath      = "_bulk_get"
)

type KibanaSavedObjectType string

const (
	StoredQuery  KibanaSavedObjectType = "query"
	IndexPattern KibanaSavedObjectType = "index-pattern"
	// TODO add remaining when they will be supported
)

type KibanaSavedObjectSearchField string

const (
	Title KibanaSavedObjectSearchField = "title"
)

var kibanaHeaders = map[string]string{
	"Content-Type": "application/json",
	"kbn-xsrf":     "true",
}

type KibanaClient struct {
	ApiRoot  string
	Username string
	Password string
}

// Extracts stored queries with provided titles
// If no titles provided extracts all stored queries
func (client *KibanaClient) RetrieveStoredQueries(titles []string) []KSavedObject {
	savedObjectType := StoredQuery
	savedObjectSearchField := Title
	return client.findSavedObjects(&savedObjectType, &savedObjectSearchField, titles)
}

// Extracts index patterns titles associated with provided stored query
func (client *KibanaClient) RetrieveIndexTitles(storedQuery KSavedObject) []string {
	var indexes []string

	var indexPatterns []KSavedObject
	savedObjectType := IndexPattern
	ids := storedQuery.ExtractIndexIds()
	if ids == nil {
		log.Warn().Msgf("no index patterns linked to query: %s", storedQuery.Attributes.Title)
		return nil
	}
	indexPatterns = client.bulkGetSavedObjects(&savedObjectType, ids)
	if indexPatterns == nil {
		log.Error().Msg("could not get index patterns")
		return nil
	}

	indexSet := make(map[string]struct{})
	for _, indexPattern := range indexPatterns {
		title := indexPattern.Attributes.Title
		if title != "" {
			indexSet[title] = struct{}{}
		}
	}

	for index := range indexSet {
		indexes = append(indexes, index)
	}

	return indexes
}

// Finds saved objects of provided type
// and searchField matching searchValues if both searchField and searchValue set
func (client *KibanaClient) findSavedObjects(savedObjectType *KibanaSavedObjectType, searchField *KibanaSavedObjectSearchField, searchValues []string) []KSavedObject {
	var savedObjects []KSavedObject

	page := 0
	perPage := 10000
	total := -1

	for total == -1 || total >= page*perPage {
		page = page + 1
		path := client.buildSavedObjectsFindPath(&page, &perPage, savedObjectType, searchField, searchValues)

		log.Debug().Msgf("performing Kibana Find Saved Objects request: %s", path)
		status, response, err := clients.SendRequest(http.MethodGet, path, kibanaHeaders, nil, nil)
		log.Debug().Msgf("Kibana Find Saved Objects response: %s", string(response))

		if err != nil || status != 200 || response == nil {
			if err != nil {
				log.Err(err).Msg("failed to perform Kibana Find Saved Objects request")
			}
			if status != 200 {
				log.Error().
					Int("status", status).
					Msg("failure Kibana Find Saved Objects response")
			}
			if response == nil {
				log.Error().Msg("Kibana Find Saved Objects response is nil")
			}
			return nil
		}

		var savedObjectsResponse KSavedObjectsResponse
		err = json.Unmarshal(response, &savedObjectsResponse)
		if err != nil {
			log.Err(err).Msg("could not parse Kibana Find Saved Objects response")
			return savedObjects
		}
		savedObjects = append(savedObjects, savedObjectsResponse.SavedObjects...)

		if total == -1 {
			total = savedObjectsResponse.Total
		}
	}
	return savedObjects
}

// Performs bulk get of saved objects for provided type and ids
func (client *KibanaClient) bulkGetSavedObjects(savedObjectType *KibanaSavedObjectType, ids []string) []KSavedObject {
	if savedObjectType == nil || ids == nil || len(ids) == 0 {
		log.Warn().Msg("could not perform Kibana Bulk Get: type and at least one id required")
		return nil
	}

	var requestBody []KBulkGetRequest
	for _, id := range ids {
		requestBody = append(requestBody, KBulkGetRequest{
			Type: string(*savedObjectType),
			Id:   id,
		})
	}

	path := client.buildBulkGetSavedObjectsPath()

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		log.Err(err).Msg("could not marshal Kibana Bulk Get request")
		return nil
	}
	status, response, err := clients.SendRequest(http.MethodPost, path, kibanaHeaders, nil, bodyBytes)
	log.Debug().
		Err(err).
		Bytes("requestBody", bodyBytes).
		Bytes("response", response).
		Msgf("Kibana Bulk Get Saved Objects request: %s", path)

	if err != nil || status != 200 || response == nil {
		if err != nil {
			log.Err(err).Msg("could not perform Kibana Bulk Get Saved Objects request")
		}
		if status != 200 {
			log.Error().
				Int("status", status).
				Msg("failure Kibana Bulk Get Saved Objects response")
		}
		if response == nil {
			log.Error().Msg("Kibana Bulk Get Saved Objects response is nil")
		}
		return nil
	}

	var savedObjectsResponse KSavedObjectsResponse
	err = json.Unmarshal(response, &savedObjectsResponse)
	if err != nil {
		log.Err(err).Msg("could not parse Kibana Bulk Get Saved Objects response")
		return nil
	}
	return savedObjectsResponse.SavedObjects
}

func (client *KibanaClient) buildSavedObjectsFindPath(page *int, perPage *int, savedObjectType *KibanaSavedObjectType, searchField *KibanaSavedObjectSearchField, searchValues []string) string {
	params := make(map[string]string)
	if savedObjectType != nil {
		params["type"] = string(*savedObjectType)
	}
	if page != nil {
		params["page"] = strconv.Itoa(*page)
	}
	if perPage != nil {
		params["per_page"] = strconv.Itoa(*perPage)
	}
	if searchField != nil && searchValues != nil {
		params["search_fields"] = string(*searchField)
		var searchValue string
		for index, id := range searchValues {
			searchValue = searchValue + id
			if index != len(searchValues)-1 {
				searchValue = searchValue + "|"
			}
		}
		params["search"] = searchValue
	}
	return client.ApiRoot + apiPath + savedObjectsPath + findPath + clients.BuildQueryParams(params)
}

func (client *KibanaClient) buildBulkGetSavedObjectsPath() string {
	return client.ApiRoot + apiPath + savedObjectsPath + bulkGetPath
}
