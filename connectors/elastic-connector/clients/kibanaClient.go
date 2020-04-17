package clients

import (
	"encoding/json"
	"github.com/gwos/tng/clients"
	"github.com/gwos/tng/connectors/elastic-connector/model"
	"github.com/gwos/tng/log"
	"net/http"
	"strconv"
)

const (
	apiPath          = "kibana/api/"
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
func (client *KibanaClient) RetrieveStoredQueries(titles []string) []model.SavedObject {
	savedObjectType := StoredQuery
	savedObjectSearchField := Title
	return client.findSavedObjects(&savedObjectType, &savedObjectSearchField, titles)
}

// Extracts index patterns titles associated with provided stored query
func (client *KibanaClient) RetrieveIndexTitles(storedQuery model.SavedObject) []string {
	var indexes []string

	var indexPatterns []model.SavedObject
	savedObjectType := IndexPattern
	ids := storedQuery.ExtractIndexIds()
	indexPatterns = client.bulkGetSavedObjects(&savedObjectType, ids)
	if indexPatterns == nil {
		log.Error("Cannot get index patterns.")
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
func (client *KibanaClient) findSavedObjects(savedObjectType *KibanaSavedObjectType, searchField *KibanaSavedObjectSearchField, searchValues []string) []model.SavedObject {
	var savedObjects []model.SavedObject

	page := 0
	perPage := 10000
	total := -1

	for total == -1 || total >= page*perPage {
		page = page + 1
		path := client.buildSavedObjectsFindPath(&page, &perPage, savedObjectType, searchField, searchValues)

		status, response, err := clients.SendRequest(http.MethodGet, path, kibanaHeaders, nil, nil)
		if err != nil || status != 200 || response == nil {
			if err != nil {
				log.Error(err)
			}
			if status != 200 {
				log.Error("Failure response code: ", status)
			}
			if response == nil {
				log.Error("Find response is nil.")
			}
			return nil
		}

		var savedObjectsResponse model.SavedObjectsResponse
		err = json.Unmarshal(response, &savedObjectsResponse)
		if err != nil {
			log.Error("Error parsing Kibana Saved Objects response: ", err)
			log.Error("Failed to extract remaining Kibana Saved Objects. The result is incomplete.")
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
func (client *KibanaClient) bulkGetSavedObjects(savedObjectType *KibanaSavedObjectType, ids []string) []model.SavedObject {
	if savedObjectType == nil || ids == nil || len(ids) == 0 {
		log.Error("Error performing Kibana Bulk Get: type and at least one id required.")
		return nil
	}

	var requestBody []model.BulkGetRequest
	for _, id := range ids {
		requestBody = append(requestBody, model.BulkGetRequest{
			Type: string(*savedObjectType),
			Id:   id,
		})
	}

	path := client.buildBulkGetSavedObjectsPath()

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		log.Error("Error marshalling Bulk Get request body: ", err)
		return nil
	}

	status, response, err := clients.SendRequest(http.MethodPost, path, kibanaHeaders, nil, bodyBytes)
	if err != nil || status != 200 || response == nil {
		if err != nil {
			log.Error(err)
		}
		if status != 200 {
			log.Error("Failure response code: ", status)
		}
		if response == nil {
			log.Error("Bulk Get response is nil.")
		}
		return nil
	}

	var savedObjectsResponse model.SavedObjectsResponse
	err = json.Unmarshal(response, &savedObjectsResponse)
	if err != nil {
		log.Error("Error parsing Kibana Bulk Get response: ", err)
		return nil
	}
	return savedObjectsResponse.SavedObjects
}

func (client *KibanaClient) buildSavedObjectsFindPath(page *int, perPage *int, savedObjectType *KibanaSavedObjectType, searchField *KibanaSavedObjectSearchField, searchValues []string) string {
	var params string
	if savedObjectType != nil {
		params = "?type=" + string(*savedObjectType)
	}
	if page != nil {
		params = appendParamsSeparator(params) + "page=" + strconv.Itoa(*page)
	}
	if perPage != nil {
		params = appendParamsSeparator(params) + "per_page=" + strconv.Itoa(*perPage)
	}
	if searchField != nil && searchValues != nil {
		params = appendParamsSeparator(params) + "search_fields=" + string(*searchField) + "&search="
		for index, id := range searchValues {
			params = params + id
			if index != len(searchValues)-1 {
				params = params + "|"
			}
		}
	}
	return client.ApiRoot + apiPath + savedObjectsPath + findPath + params
}

func (client *KibanaClient) buildBulkGetSavedObjectsPath() string {
	return client.ApiRoot + apiPath + savedObjectsPath + bulkGetPath
}

func appendParamsSeparator(params string) string {
	if params != "" {
		params = params + "&"
	} else {
		params = "?"
	}
	return params
}
