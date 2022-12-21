package clients

import (
	"encoding/base64"
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

type KibanaClient struct {
	APIRoot  string
	Username string
	Password string

	headers map[string]string
}

func (client *KibanaClient) InitClient() error {
	client.headers = map[string]string{
		"Content-Type": "application/json",
		"kbn-xsrf":     "true",
	}
	if len(client.Password) != 0 {
		client.headers["Authorization"] = "Basic " +
			base64.StdEncoding.EncodeToString([]byte(client.Username+":"+client.Password))
	}
	return nil
}

// Extracts stored queries with provided titles
// If no titles provided extracts all stored queries
func (client *KibanaClient) RetrieveStoredQueries(titles []string) []KSavedObject {
	savedObjectType := StoredQuery
	savedObjectSearchField := Title
	return client.findSavedObjects(&savedObjectType, &savedObjectSearchField, titles)
}

// RetrieveIndexTitles extracts index patterns titles associated with provided stored query
func (client *KibanaClient) RetrieveIndexTitles(storedQuery KSavedObject) []string {
	var (
		indexPatterns []KSavedObject
		indexes       = make([]string, 0)
	)

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
		status, response, err := clients.SendRequest(http.MethodGet, path, client.headers, nil, nil)
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

	var requestBody = make([]KBulkGetRequest, 0, len(ids))
	for _, id := range ids {
		requestBody = append(requestBody, KBulkGetRequest{
			Type: string(*savedObjectType),
			ID:   id,
		})
	}

	path := client.buildBulkGetSavedObjectsPath()

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		log.Err(err).Msg("could not marshal Kibana Bulk Get request")
		return nil
	}
	status, response, err := clients.SendRequest(http.MethodPost, path, client.headers, nil, bodyBytes)
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
	return client.APIRoot + apiPath + savedObjectsPath + findPath + clients.BuildQueryParams(params)
}

func (client *KibanaClient) buildBulkGetSavedObjectsPath() string {
	return client.APIRoot + apiPath + savedObjectsPath + bulkGetPath
}
