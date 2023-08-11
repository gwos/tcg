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
	defaultIndexPath = "index_patterns/default"
	savedObjectsPath = "saved_objects/"
	findPath         = "_find"
	bulkGetPath      = "_bulk_get"
	bulkResolvePath  = "_bulk_resolve"
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
	return client.FindSO(StoredQuery, Title, titles)
}

// RetrieveIndexTitles extracts index patterns titles associated with provided stored query
func (client *KibanaClient) RetrieveIndexTitles(storedQuery KSavedObject) []string {
	ids := storedQuery.ExtractIndexIds()
	if len(ids) == 0 {
		// Kibana UI uses default index pattern unless user picks a different one.
		defaultIndexID := client.GetDefaultIndexID()
		if defaultIndexID == "" {
			log.Warn().
				Interface("storedQuery", storedQuery).
				Msg("no index patterns linked to query and no default index pattern")
			return nil
		}
		ids = append(ids, defaultIndexID)
	}

	indexPatterns := client.BulkResolveSO(IndexPattern, ids)
	if len(indexPatterns) == 0 {
		log.Warn().Msg("could not resolve index patterns")
		return nil
	}

	indexSet := make(map[string]bool)
	for _, indexPattern := range indexPatterns {
		if title := indexPattern.Attributes.Title; title != "" {
			indexSet[title] = true
		}
	}

	indexes := make([]string, 0, len(indexSet))
	for index := range indexSet {
		indexes = append(indexes, index)
	}

	return indexes
}

func (client *KibanaClient) GetDefaultIndexID() string {
	path := client.APIRoot + apiPath + defaultIndexPath
	status, response, err := clients.SendRequest(http.MethodGet, path, client.headers, nil, nil)

	if err != nil || status != 200 || len(response) == 0 {
		if err != nil {
			log.Err(err).Msg("failed to perform Kibana Get Default Index request")
		}
		if status != 200 {
			log.Error().
				Int("status", status).
				Msg("failure Kibana Get Default Index response")
		}
		if len(response) == 0 {
			log.Error().Msg("Kibana Get Default Index response is empty")
		}
		return ""
	} else {
		log.Debug().
			RawJSON("response", response).
			Msgf("Kibana Get Default Index request: %s", path)
	}

	p := new(struct {
		IndexPatternID string `json:"index_pattern_id"`
	})
	if err := json.Unmarshal(response, p); err != nil {
		log.Err(err).Msg("could not parse Kibana Get Default Index response")
		return ""
	}
	return p.IndexPatternID
}

// FindSO finds saved objects of provided type
// and searchField matching searchValues if both searchField and searchValue set
func (client *KibanaClient) FindSO(savedObjectType KibanaSavedObjectType, searchField KibanaSavedObjectSearchField, searchValues []string) []KSavedObject {
	var savedObjects []KSavedObject

	for page, perPage, total := 1, 1000, -1; total == -1 || total >= page*perPage; page++ {
		path := client.buildFindSOPath(&page, &perPage, &savedObjectType, &searchField, searchValues)
		status, response, err := clients.SendRequest(http.MethodGet, path, client.headers, nil, nil)

		if err != nil || status != 200 || len(response) == 0 {
			if err != nil {
				log.Err(err).Msg("failed to perform Kibana Find Saved Objects request")
			}
			if status != 200 {
				log.Error().
					Int("status", status).
					Msg("failure Kibana Find Saved Objects response")
			}
			if len(response) == 0 {
				log.Error().Msg("Kibana Find Saved Objects response is empty")
			}
			return nil
		} else {
			log.Debug().
				RawJSON("response", response).
				Msgf("Kibana Find Saved Objects request: %s", path)
		}

		var savedObjectsResponse KSavedObjectsResponse
		if err := json.Unmarshal(response, &savedObjectsResponse); err != nil {
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

// BulkGetSO performs bulk get of saved objects for provided type and ids
func (client *KibanaClient) BulkGetSO(savedObjectType KibanaSavedObjectType, ids []string) []KSavedObject {
	if len(ids) == 0 {
		log.Warn().Msg("could not perform Kibana Bulk Get: at least one id required")
		return nil
	}

	var requestBody = make([]KBulkGetSORequestItem, 0, len(ids))
	for _, id := range ids {
		requestBody = append(requestBody, KBulkGetSORequestItem{
			Type: string(savedObjectType),
			ID:   id,
		})
	}

	path := client.buildBulkGetSOPath()

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		log.Err(err).Msg("could not marshal Kibana Bulk Get request")
		return nil
	}
	status, response, err := clients.SendRequest(http.MethodPost, path, client.headers, nil, bodyBytes)

	if err != nil || status != 200 || len(response) == 0 {
		if err != nil {
			log.Err(err).Msg("could not perform Kibana Bulk Get Saved Objects request")
		}
		if status != 200 {
			log.Error().
				Int("status", status).
				Msg("failure Kibana Bulk Get Saved Objects response")
		}
		if len(response) == 0 {
			log.Error().Msg("Kibana Bulk Get Saved Objects response is empty")
		}
		return nil
	} else {
		log.Debug().
			Err(err).
			RawJSON("request", bodyBytes).
			RawJSON("response", response).
			Msgf("Kibana Bulk Get Saved Objects request: %s", path)
	}

	bulkResponse := new(KBulkGetSOResponse)
	if err := json.Unmarshal(response, bulkResponse); err != nil {
		log.Err(err).Msg("could not parse Kibana Bulk Get Saved Objects response")
		return nil
	}
	savedObjects := make([]KSavedObject, 0)
	for _, o := range bulkResponse.SavedObjects {
		if o.Error != nil {
			log.Warn().
				Interface("data", o).
				Msg("error in Kibana Bulk Get Saved Objects response")
			continue
		}
		savedObjects = append(savedObjects, KSavedObject{Type: o.Type, ID: o.ID, Attributes: o.Attributes})
	}

	return savedObjects
}

// BulkResolveSO performs bulk resolve of saved objects for provided type and ids
func (client *KibanaClient) BulkResolveSO(savedObjectType KibanaSavedObjectType, ids []string) []KSavedObject {
	if len(ids) == 0 {
		log.Warn().Msg("could not perform Kibana Bulk Resolve: at least one id required")
		return nil
	}

	var requestBody = make([]KBulkGetSORequestItem, 0, len(ids))
	for _, id := range ids {
		requestBody = append(requestBody, KBulkGetSORequestItem{
			Type: string(savedObjectType),
			ID:   id,
		})
	}

	path := client.buildBulkResolveSOPath()

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		log.Err(err).Msg("could not marshal Kibana Bulk Resolve request")
		return nil
	}
	status, response, err := clients.SendRequest(http.MethodPost, path, client.headers, nil, bodyBytes)

	if err != nil || status != 200 || len(response) == 0 {
		if err != nil {
			log.Err(err).Msg("could not perform Kibana Bulk Resolve Saved Objects request")
		}
		if status != 200 {
			log.Error().
				Int("status", status).
				Msg("failure Kibana Bulk Resolve Saved Objects response")
		}
		if len(response) == 0 {
			log.Error().Msg("Kibana Bulk Resolve Saved Objects response is empty")
		}
		return nil
	} else {
		log.Debug().
			Err(err).
			RawJSON("request", bodyBytes).
			RawJSON("response", response).
			Msgf("Kibana Bulk Resolve Saved Objects request: %s", path)
	}

	bulkResponse := new(KBulkResolveSOResponse)
	if err := json.Unmarshal(response, bulkResponse); err != nil {
		log.Err(err).Msg("could not parse Kibana Bulk Resolve Saved Objects response")
		return nil
	}
	savedObjects := make([]KSavedObject, 0)
	for _, item := range bulkResponse.ResolvedObjects {
		if item.SavedObject.Error != nil {
			log.Warn().
				Interface("data", item).
				Msg("error in Kibana Bulk Resolve Saved Objects response")
			continue
		}
		savedObjects = append(savedObjects,
			KSavedObject{Type: item.SavedObject.Type, ID: item.SavedObject.ID, Attributes: item.SavedObject.Attributes})
	}

	return savedObjects
}

func (client *KibanaClient) buildFindSOPath(page *int, perPage *int, savedObjectType *KibanaSavedObjectType, searchField *KibanaSavedObjectSearchField, searchValues []string) string {
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
	return client.APIRoot + apiPath + savedObjectsPath + findPath + clients.BuildQueryParams(params) +
		`&fields=filters&fields=title&fields=typeMeta`
}

func (client *KibanaClient) buildBulkGetSOPath() string {
	return client.APIRoot + apiPath + savedObjectsPath + bulkGetPath
}

func (client *KibanaClient) buildBulkResolveSOPath() string {
	return client.APIRoot + apiPath + savedObjectsPath + bulkResolvePath
}
