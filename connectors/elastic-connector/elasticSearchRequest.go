package main

import (
	"github.com/gwos/tng/log"
	"time"
)

const (
	trackTotalHits = true
	perPage        = 10000
	sortById       = "_id:asc"
	from           = 0
)

func buildSearchBody(savedObject SavedObject) SearchBody {
	var searchBody SearchBody

	for _, filter := range savedObject.Attributes.Filters {
		if filter.Meta.Disabled {
			continue
		}
		switch filter.Meta.Type {
		case "phrase":
			addPhraseFilterClause(filter, &searchBody)
			break
		case "phrases":
			addPhrasesFilterClause(filter, &searchBody)
			break
		case "range":
			addRangeFilterClause(filter, &searchBody)
			break
		case "exists":
			addExistsFilterClause(filter, &searchBody)
			break
		default:
			log.Error("Could not add query clause. Unknown filter type: ", filter.Meta.Type)
			break
		}
	}

	if savedObject.Attributes.Timefilter != nil {
		addTimeFilter(savedObject, &searchBody)
	}

	return searchBody
}

// "phrase" filters are: "is", "is not"
// so simply add a "must" or "must not" clause
func addPhraseFilterClause(filter Filter, body *SearchBody) {
	match := map[string]interface{}{
		filter.Meta.Key: filter.Meta.Value,
	}
	clause := Clause{Match: &match}
	var clauseType string
	if !filter.Meta.Negate {
		clauseType = "must"
	} else {
		clauseType = "must_not"
	}
	appendClause(clauseType, clause, body)
}

// "phrases" filters are: "is one of", "is not one of"
func addPhrasesFilterClause(filter Filter, body *SearchBody) {
	var key = filter.Meta.Key
	params := filter.Meta.Params.([]interface{})
	if filter.Meta.Negate {
		// if filter is "is NOT one of" that means that the value must not be any of them
		// so add all of them as "must not" clause
		for _, param := range params {
			match := map[string]interface{}{
				key: param,
			}
			clause := Clause{Match: &match}
			appendClause("must_not", clause, body)
		}
	} else {
		// if filter is "is one of" that means that if value is just one of them that's fine
		// so add them to "should" clause with minimum should match = 1
		var clauses []Clause
		for _, param := range params {
			match := map[string]interface{}{
				key: param,
			}
			clauses = append(clauses, Clause{Match: &match})
		}
		minimumShouldMatch := 1
		boolClause := Bool{Should: clauses, MinimumShouldMatch: &minimumShouldMatch}
		clause := Clause{Bool: &boolClause}
		appendClause("must", clause, body)
	}
}

func addRangeFilterClause(filter Filter, body *SearchBody) {
	var key = filter.Meta.Key
	var rangeClause map[string]interface{}
	// time ranges must be adjusted to timezone, other kinds of ranges can be simply copied
	if key == "@timestamp" {
		if filter.Range == nil || filter.Range.Timestamp == nil {
			log.Error("Could not add range clause for @timestamp: could not retrieve appropriate value from filter.")
			return
		}
		timestamp := filter.Range.Timestamp
		adjustTimestamp(timestamp)
		rangeClause = map[string]interface{}{
			key: timestamp,
		}
	} else {
		params := filter.Meta.Params
		rangeClause = map[string]interface{}{
			key: params,
		}
	}
	clause := Clause{Range: &rangeClause}
	var clauseType string
	if !filter.Meta.Negate {
		clauseType = "must"
	} else {
		clauseType = "must_not"
	}
	appendClause(clauseType, clause, body)
}

func addExistsFilterClause(filter Filter, body *SearchBody) {
	exists := Exists{Field: filter.Meta.Key}
	clause := Clause{
		Exists: &exists,
	}
	var clauseType string
	if !filter.Meta.Negate {
		clauseType = "must"
	} else {
		clauseType = "must_not"
	}
	appendClause(clauseType, clause, body)
}

func addTimeFilter(savedObject SavedObject, body *SearchBody) {
	if savedObject.Attributes.Timefilter == nil {
		return
	}
	timestamp := Timestamp{
		From: savedObject.Attributes.Timefilter.From,
		To:   savedObject.Attributes.Timefilter.To,
	}
	adjustTimestamp(&timestamp)
	rangeClause := map[string]interface{}{
		"@timestamp": timestamp,
	}
	clause := Clause{Range: &rangeClause}
	appendClause("filter", clause, body)
}

func appendClause(clauseType string, clause Clause, body *SearchBody) {
	var query Query
	if body.Query == nil {
		body.Query = &query
	}
	switch clauseType {
	case "must":
		if body.Query.Bool.Must == nil {
			body.Query.Bool.Must = []Clause{clause}
		} else {
			body.Query.Bool.Must = append(body.Query.Bool.Must, clause)
		}
		break
	case "must_not":
		if body.Query.Bool.MustNot == nil {
			body.Query.Bool.MustNot = []Clause{clause}
		} else {
			body.Query.Bool.MustNot = append(body.Query.Bool.MustNot, clause)
		}
		break
	case "should":
		if body.Query.Bool.Should == nil {
			body.Query.Bool.Should = []Clause{clause}
		} else {
			body.Query.Bool.Should = append(body.Query.Bool.Should, clause)
		}
		break
	case "filter":
		if body.Query.Bool.Filter == nil {
			body.Query.Bool.Filter = []Clause{clause}
		} else {
			body.Query.Bool.Filter = append(body.Query.Bool.Filter, clause)
		}
		break
	default:
		log.Error("Could not add query clause. Unknown clause type: ", clauseType)
		break
	}
}

// converts timestamp to absolute if relative and shifts if to UTC timezone
func adjustTimestamp(timestamp *Timestamp) {
	from := timestamp.From
	to := timestamp.To
	timeFrom := parseTime(from, true, time.UTC)
	timeTo := parseTime(to, false, time.UTC)
	timestamp.From = timeFrom.Format(layout)
	timestamp.To = timeTo.Format(layout)
}

func setSingleSearchAfter(v interface{}, searchBody *SearchBody) {
	var searchAfter []interface{}
	searchAfter = append(searchAfter, v)
	searchBody.SearchAfter = searchAfter
}

type SearchBody struct {
	Query       *Query        `json:"query,omitempty"`
	SearchAfter []interface{} `json:"search_after,omitempty"`
}

type Query struct {
	Bool Bool `json:"bool"`
}

type Bool struct {
	Must               []Clause `json:"must,omitempty"`
	MustNot            []Clause `json:"must_not,omitempty"`
	Should             []Clause `json:"should,omitempty"`
	Filter             []Clause `json:"filter,omitempty"`
	MinimumShouldMatch *int     `json:"minimum_should_match,omitempty"`
}

type Clause struct {
	Match  *map[string]interface{} `json:"match,omitempty"`
	Range  *map[string]interface{} `json:"range,omitempty"`
	Exists *Exists                 `json:"exists,omitempty"`
	Bool   *Bool                   `json:"bool,omitempty"`
}

type Exists struct {
	Field string `json:"field,omitempty"`
}

type SearchResponse struct {
	Took int  `json:"took"`
	Hits Hits `json:"hits"`
}

type Hits struct {
	Total TotalHits `json:"total"`
	Hits  []Hit     `json:"hits"`
}

type Hit struct {
	Index  string                 `json:"_index"`
	Type   string                 `json:"_type"`
	ID     string                 `json:"_id"`
	Score  float64                `json:"_score"`
	Source map[string]interface{} `json:"_source"`
}

type TotalHits struct {
	Value int `json:"value"`
}
