package main

import (
	"github.com/gwos/tng/log"
	"time"
)

func buildSearchRequest(savedObject SavedObject, simpleHttpRequest bool) SearchRequest {
	var searchRequest SearchRequest

	for _, filter := range savedObject.Attributes.Filters {
		if filter.Meta.Disabled {
			continue
		}
		switch filter.Meta.Type {
		case "phrase":
			addPhraseFilterClause(filter, &searchRequest)
			break
		case "phrases":
			addPhrasesFilterClause(filter, &searchRequest)
			break
		case "range":
			addRangeFilterClause(filter, &searchRequest)
			break
		case "exists":
			addExistsFilterClause(filter, &searchRequest)
			break
		default:
			log.Error("Could not add query clause. Unknown filter type: ", filter.Meta.Type)
			break
		}
	}

	if savedObject.Attributes.Timefilter != nil {
		addTimeFilter(savedObject, &searchRequest)
	}

	if simpleHttpRequest {
		trackTotalHits := true
		searchRequest.TrackTotalHits = &trackTotalHits

		perPage := 10000
		searchRequest.Size = &perPage

		var sortByIdAsc = map[string]string{
			"_id": "asc",
		}
		searchRequest.Sort = append(searchRequest.Sort, sortByIdAsc)
	}

	return searchRequest
}

// "phrase" filters are: "is", "is not"
// so simply add a "must" or "must not" clause
func addPhraseFilterClause(filter Filter, request *SearchRequest) {
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
	appendClause(clauseType, clause, request)
}

// "phrases" filters are: "is one of", "is not one of"
func addPhrasesFilterClause(filter Filter, request *SearchRequest) {
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
			appendClause("must_not", clause, request)
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
		appendClause("must", clause, request)
	}
}

func addRangeFilterClause(filter Filter, request *SearchRequest) {
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
	appendClause(clauseType, clause, request)
}

func addExistsFilterClause(filter Filter, request *SearchRequest) {
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
	appendClause(clauseType, clause, request)
}

func addTimeFilter(savedObject SavedObject, request *SearchRequest) {
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
	appendClause("filter", clause, request)
}

func appendClause(clauseType string, clause Clause, request *SearchRequest) {
	var query Query
	if request.Query == nil {
		request.Query = &query
	}
	switch clauseType {
	case "must":
		if request.Query.Bool.Must == nil {
			request.Query.Bool.Must = []Clause{clause}
		} else {
			request.Query.Bool.Must = append(request.Query.Bool.Must, clause)
		}
		break
	case "must_not":
		if request.Query.Bool.MustNot == nil {
			request.Query.Bool.MustNot = []Clause{clause}
		} else {
			request.Query.Bool.MustNot = append(request.Query.Bool.MustNot, clause)
		}
		break
	case "should":
		if request.Query.Bool.Should == nil {
			request.Query.Bool.Should = []Clause{clause}
		} else {
			request.Query.Bool.Should = append(request.Query.Bool.Should, clause)
		}
		break
	case "filter":
		if request.Query.Bool.Filter == nil {
			request.Query.Bool.Filter = []Clause{clause}
		} else {
			request.Query.Bool.Filter = append(request.Query.Bool.Filter, clause)
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
