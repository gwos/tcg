package main

import (
	"encoding/json"
	"github.com/gwos/tng/clients"
	"github.com/gwos/tng/log"
	"net/http"
	"strconv"
)

const (
	defaultKibanaApiPath = "http://localhost:5601/kibana/api/"
	savedObjectsPath     = "saved_objects/"
	findPath             = "_find"
	bulkGetPath          = "_bulk_get"
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

func retrieveStoredQueries(titles []string) []SavedObject {
	savedObjectType := StoredQuery
	savedObjectSearchField := Title
	return findSavedObjects(&savedObjectType, &savedObjectSearchField, titles)
}

func retrieveIndexTitles(ids []string) []string {
	var indexes []string

	var indexPatterns []SavedObject
	savedObjectType := IndexPattern
	indexPatterns = bulkGetSavedObjects(&savedObjectType, ids)
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

func findSavedObjects(savedObjectType *KibanaSavedObjectType, searchField *KibanaSavedObjectSearchField, searchValues []string) []SavedObject {
	var savedObjects []SavedObject

	page := 0
	perPage := 1000
	total := -1

	allSuccessful := true
	for total == -1 || total >= page*perPage {
		page = page + 1
		path := buildSavedObjectsFindPath(&page, &perPage, savedObjectType, searchField, searchValues)

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

		var savedObjectsResponse SavedObjectsResponse
		err = json.Unmarshal(response, &savedObjectsResponse)
		if err != nil {
			log.Error("Error parsing Kibana Saved Objects response: ", err)
			if total != -1 {
				// previous parsings were fine let's try to get remaining data
				allSuccessful = false
				continue
			} else {
				log.Error("Kibana Find Saved Objects failed.")
				return nil
			}
		}
		savedObjects = append(savedObjects, savedObjectsResponse.SavedObjects...)

		if total == -1 {
			total = savedObjectsResponse.Total
		}
	}
	if !allSuccessful && savedObjects != nil {
		log.Error("Failed to extract some of Kibana Saved Objects. The result is probably incomplete.")
	}
	return savedObjects
}

func bulkGetSavedObjects(savedObjectType *KibanaSavedObjectType, ids []string) []SavedObject {
	if savedObjectType == nil || ids == nil || len(ids) == 0 {
		log.Error("Error performing Kibana Bulk Get: type and at least one id required.")
		return nil
	}

	var requestBody []BulkGetRequest
	for _, id := range ids {
		requestBody = append(requestBody, BulkGetRequest{
			Type: string(*savedObjectType),
			Id:   id,
		})
	}

	path := buildBulkGetSavedObjectsPath()

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

	var savedObjectsResponse SavedObjectsResponse
	err = json.Unmarshal(response, &savedObjectsResponse)
	if err != nil {
		log.Error("Error parsing Kibana Bulk Get response: ", err)
		return nil
	}
	return savedObjectsResponse.SavedObjects
}

func buildSavedObjectsFindPath(page *int, perPage *int, savedObjectType *KibanaSavedObjectType, searchField *KibanaSavedObjectSearchField, searchValues []string) string {
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
	return getKibanaApiPath() + savedObjectsPath + findPath + params
}

func appendParamsSeparator(params string) string {
	if params != "" {
		params = params + "&"
	} else {
		params = "?"
	}
	return params
}

func buildBulkGetSavedObjectsPath() string {
	return getKibanaApiPath() + savedObjectsPath + bulkGetPath
}

func getKibanaApiPath() string {
	// TODO get from environment variables if not set return default
	return defaultKibanaApiPath
}

func extractIndexIds(storedQuery SavedObject) []string {
	indexIdsSet := make(map[string]struct{})
	for _, filter := range storedQuery.Attributes.Filters {
		if filter.Meta.Index != "" {
			indexIdsSet[filter.Meta.Index] = struct{}{}
		}
	}
	var indexIds []string
	for indexId := range indexIdsSet {
		indexIds = append(indexIds, indexId)
	}
	return indexIds
}

type SavedObjectsResponse struct {
	Page         int           `json:"page"`
	PerPage      int           `json:"per_page"`
	Total        int           `json:"total"`
	SavedObjects []SavedObject `json:"saved_objects"`
}

type SavedObject struct {
	Type       string     `json:"type"`
	ID         string     `json:"id"`
	Attributes Attributes `json:"attributes"`
}

type Attributes struct {
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Filters     []Filter    `json:"filters,omitempty"`
	Timefilter  *TimeFilter `json:"timefilter,omitempty"`
}

type Filter struct {
	Meta  Meta   `json:"meta"`
	Range *Range `json:"range,omitempty"`
}

type Meta struct {
	Index    string      `json:"index"`
	Negate   bool        `json:"negate"`
	Disabled bool        `json:"disabled"`
	Type     string      `json:"type"`
	Key      string      `json:"key"`
	Value    interface{} `json:"value"`
	Params   interface{} `json:"params"`
}

type Range struct {
	Timestamp *Timestamp `json:"@timestamp,omitempty"`
}

type Timestamp struct {
	From string `json:"gte"`
	To   string `json:"lt"`
}

type TimeFilter struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type BulkGetRequest struct {
	Type string `json:"type"`
	Id   string `json:"id"`
}
