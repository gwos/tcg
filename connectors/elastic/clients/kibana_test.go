package clients

import (
	"encoding/json"
	"testing"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestBulkGetSavedObjects(t *testing.T) {
	// NOTE: Saved objects that are unable to persist are replaced with an error object.
	// https://www.elastic.co/guide/en/kibana/current/saved-objects-api-bulk-get.html#saved-objects-api-bulk-get-response-body
	res := []byte(`{"saved_objects": [
  {
    "id": "my-pattern",
    "type": "index-pattern",
    "version": 1,
    "attributes": {
      "title": "my-pattern-*"
    }
  },
  {
    "id": "my-dashboard",
    "type": "dashboard",
    "error": {
      "statusCode": 404,
      "message": "Not found"
    }
  }
]}`)
	var bulkResponse KBulkGetSOResponse
	assert.NoError(t, json.Unmarshal(res, &bulkResponse))

	actualErrors, expectedErrors := 0, 1
	savedObjects := make([]KSavedObject, 0)
	for _, o := range bulkResponse.SavedObjects {
		if o.Error != nil {
			actualErrors++

			log.Warn().
				Interface("data", o).
				Msg("error in Kibana Bulk Get Saved Objects response")
			continue
		}
		savedObjects = append(savedObjects, KSavedObject{Type: o.Type, ID: o.ID, Attributes: o.Attributes})
	}
	assert.Equal(t, expectedErrors, actualErrors)
	assert.Equal(t, 1, len(savedObjects))
}
