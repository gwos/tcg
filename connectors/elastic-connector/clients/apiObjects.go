package clients

import (
	"fmt"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/transit"
	"strconv"
	"strings"
	"time"
)

type FilterType string

const (
	typePhrase  FilterType = "phrase"
	typePhrases FilterType = "phrases"
	typeRange   FilterType = "range"
	typeExist   FilterType = "exists"
)

type clauseType string

const (
	mustClause    clauseType = "must"
	mustNotClause clauseType = "must_not"
	shouldClause  clauseType = "should"
	filterClause  clauseType = "filter"
)

const (
	layout = "2006-01-02T15:04:05.000Z"
	now    = "now"
)

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
	TimeFilter  *TimeFilter `json:"timefilter,omitempty"`
}

type Filter struct {
	Meta  Meta   `json:"meta"`
	Range *Range `json:"range,omitempty"`
}

type Meta struct {
	Index    string      `json:"index"`
	Negate   bool        `json:"negate"`
	Disabled bool        `json:"disabled"`
	Type     FilterType  `json:"type"`
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

// Extracts indexes's ids linked to stored query's filters
func (storedQuery *SavedObject) ExtractIndexIds() []string {
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

// Builds search request query corresponding to the stored query's filters
func (searchBody *SearchBody) FromStoredQuery(storedQuery SavedObject) {
	for _, filter := range storedQuery.Attributes.Filters {
		if filter.Meta.Disabled {
			continue
		}
		switch filter.Meta.Type {
		case typePhrase:
			searchBody.addPhraseFilterClause(filter)
			break
		case typePhrases:
			searchBody.addPhrasesFilterClause(filter)
			break
		case typeRange:
			searchBody.addRangeFilterClause(filter)
			break
		case typeExist:
			searchBody.addExistsFilterClause(filter)
			break
		default:
			log.Error("Could not add query clause. Unknown filterClause type: ", filter.Meta.Type)
			break
		}
	}

	if storedQuery.Attributes.TimeFilter != nil {
		searchBody.withTimeFilter(storedQuery)
	}
}

// Sets "search after" with a single value to request body
func (searchBody *SearchBody) WithSingleSearchAfter(v interface{}) {
	var searchAfter []interface{}
	searchAfter = append(searchAfter, v)
	searchBody.SearchAfter = searchAfter
}

// Adds query clause corresponding to the "is", "is not" filter to search request body
func (searchBody *SearchBody) addPhraseFilterClause(filter Filter) {
	match := map[string]interface{}{
		filter.Meta.Key: filter.Meta.Value,
	}
	clause := Clause{Match: &match}
	var clauseType clauseType
	if !filter.Meta.Negate {
		clauseType = mustClause
	} else {
		clauseType = mustNotClause
	}
	searchBody.appendClause(clauseType, clause)
}

// Adds query clause corresponding to the "is one of", "is not one of" filter to search request body
func (searchBody *SearchBody) addPhrasesFilterClause(filter Filter) {
	var key = filter.Meta.Key
	params := filter.Meta.Params.([]interface{})
	if filter.Meta.Negate {
		// if filterClause is "is NOT one of" that means that the value mustClause not be any of them
		// so add all of them as "mustClause not" clause
		for _, param := range params {
			match := map[string]interface{}{
				key: param,
			}
			clause := Clause{Match: &match}
			searchBody.appendClause(mustNotClause, clause)
		}
	} else {
		// if filterClause is "is one of" that means that if value is just one of them that's fine
		// so add them to "shouldClause" clause with minimum shouldClause match = 1
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
		searchBody.appendClause(mustClause, clause)
	}
}

// Adds query clause corresponding to the "range" filter to search request body
func (searchBody *SearchBody) addRangeFilterClause(filter Filter) {
	var key = filter.Meta.Key
	var rangeClause map[string]interface{}
	// time ranges mustClause be adjusted to timezone, other kinds of ranges can be simply copied
	if key == "@timestamp" {
		if filter.Range == nil || filter.Range.Timestamp == nil {
			log.Error("Could not add range clause for @timestamp: could not retrieve appropriate value from filterClause.")
			return
		}
		timestamp := filter.Range.Timestamp
		rangeClause = map[string]interface{}{
			key: timestamp.toAbsoluteUtcTime(),
		}
	} else {
		params := filter.Meta.Params
		rangeClause = map[string]interface{}{
			key: params,
		}
	}
	clause := Clause{Range: &rangeClause}
	var clauseType clauseType
	if !filter.Meta.Negate {
		clauseType = mustClause
	} else {
		clauseType = mustNotClause
	}
	searchBody.appendClause(clauseType, clause)
}

// Adds query clause corresponding to the "exists" filter to search request body
func (searchBody *SearchBody) addExistsFilterClause(filter Filter) {
	exists := Exists{Field: filter.Meta.Key}
	clause := Clause{
		Exists: &exists,
	}
	var clauseType clauseType
	if !filter.Meta.Negate {
		clauseType = mustClause
	} else {
		clauseType = mustNotClause
	}
	searchBody.appendClause(clauseType, clause)
}

// Adds query clause corresponding to the time filter to request body
func (searchBody *SearchBody) withTimeFilter(savedObject SavedObject) {
	if savedObject.Attributes.TimeFilter == nil {
		return
	}
	timestamp := Timestamp{
		From: savedObject.Attributes.TimeFilter.From,
		To:   savedObject.Attributes.TimeFilter.To,
	}
	rangeClause := map[string]interface{}{
		"@timestamp": timestamp.toAbsoluteUtcTime(),
	}
	clause := Clause{Range: &rangeClause}
	searchBody.appendClause(filterClause, clause)
}

// Adds query clause of appropriate type to request body
func (searchBody *SearchBody) appendClause(clauseType clauseType, clause Clause) {
	var query Query
	if searchBody.Query == nil {
		searchBody.Query = &query
	}
	switch clauseType {
	case mustClause:
		if searchBody.Query.Bool.Must == nil {
			searchBody.Query.Bool.Must = []Clause{clause}
		} else {
			searchBody.Query.Bool.Must = append(searchBody.Query.Bool.Must, clause)
		}
		break
	case mustNotClause:
		if searchBody.Query.Bool.MustNot == nil {
			searchBody.Query.Bool.MustNot = []Clause{clause}
		} else {
			searchBody.Query.Bool.MustNot = append(searchBody.Query.Bool.MustNot, clause)
		}
		break
	case shouldClause:
		if searchBody.Query.Bool.Should == nil {
			searchBody.Query.Bool.Should = []Clause{clause}
		} else {
			searchBody.Query.Bool.Should = append(searchBody.Query.Bool.Should, clause)
		}
		break
	case filterClause:
		if searchBody.Query.Bool.Filter == nil {
			searchBody.Query.Bool.Filter = []Clause{clause}
		} else {
			searchBody.Query.Bool.Filter = append(searchBody.Query.Bool.Filter, clause)
		}
		break
	default:
		log.Error("Could not add query clause. Unknown clause type: ", clauseType)
		break
	}
}

// Modifies timestamp's from/to values to absolute time in UTC time zone
func (timestamp *Timestamp) toAbsoluteUtcTime() *Timestamp {
	from, err := toAbsoluteTime(timestamp.From, true)
	if err != nil {
		log.Error("Cannot parse 'from' timestamp: ", err)
	}
	to, err := toAbsoluteTime(timestamp.To, false)
	if err != nil {
		log.Error("Cannot parse 'to' timestamp: ", err)
	}
	timestamp.From = from.In(time.UTC).Format(layout)
	timestamp.To = to.In(time.UTC).Format(layout)
	return timestamp
}

// Converts timeFilter's from/to values to Time in local time zone and returns TimeInterval with appropriate StartTime/EndTime
func (timeFilter *TimeFilter) ToTimeInterval() *transit.TimeInterval {
	startTime, err := toAbsoluteTime(timeFilter.From, true)
	if err != nil {
		log.Error("Cannot parse time filter's 'from': ", err)
	}
	endTime, err := toAbsoluteTime(timeFilter.To, false)
	if err != nil {
		log.Error("Cannot parse time filter's 'to': ", err)
	}
	timeInterval := &transit.TimeInterval{
		StartTime: milliseconds.MillisecondTimestamp{Time: startTime},
		EndTime:   milliseconds.MillisecondTimestamp{Time: endTime},
	}
	return timeInterval
}

// Converts relative expressions such as "now-5d" to Time in local time zone
func toAbsoluteTime(expression string, isStartTime bool) (time.Time, error) {
	if !strings.Contains(expression, now) {
		return time.Parse(layout, expression)
	}
	timeNow := time.Now()
	location := timeNow.Location()
	if expression == now {
		return timeNow, nil
	}
	// character after "now" is operator (+/-)
	operator := expression[len(now) : len(now)+1]
	// everything after "now" and operator is a relative part
	relativePart := expression[len(now)+1:]
	var rounded = false
	// rounded expression ends with "/y", "/M", "/w", "/d", "/h", "/m", "/s"
	if strings.Contains(relativePart, "/") {
		// remove these last 2 characters from relative part, keep in mind that expression is rounded
		relativePart = relativePart[:len(relativePart)-2]
		rounded = true
	}
	// the last character of the relative part defines period ("y", "M", "w", "d", "h", "m", "s"), everything else is interval
	interval := relativePart[:len(relativePart)-1]
	period := relativePart[len(relativePart)-1:]
	i, err := strconv.Atoi(interval)
	if operator == "-" {
		i = -i
	}
	if err != nil {
		log.Error(fmt.Sprintf("Error parsing time filterClause expression: %s", err.Error()))
	}
	var result time.Time
	switch period {
	case "y":
		result = timeNow.AddDate(i, 0, 0)
		if rounded {
			if isStartTime {
				// StartTime is being rounded to the beginning of period
				result = time.Date(result.Year(), 1, 1, 0, 0, 0, 0, location)
			} else {
				// EndTime is being rounded to the last millisecond of period
				// to achieve this we subtract one millisecond from the next period
				result = time.Date(result.Year()+1, 1, 1, 0, 0, 0, 0, location)
				result = result.Add(-1 * time.Millisecond)
			}
		}
		break
	case "M":
		result = timeNow.AddDate(0, i, 0)
		if rounded {
			if isStartTime {
				result = time.Date(result.Year(), result.Month(), 1, 0, 0, 0, 0, location)
			} else {
				result = time.Date(result.Year(), result.Month()+1, 1, 0, 0, 0, 0, location)
				result = result.Add(-1 * time.Millisecond)
			}
		}
		break
	case "w":
		dayOfDesiredWeek := timeNow.AddDate(0, 0, 7*i)
		if rounded {
			// week is being rounded to the beginning of past Sunday for StartTime filterClause and to the end of next Saturday for EndTime filterClause
			var offsetFromSunday int
			var offsetToSaturday int
			switch dayOfDesiredWeek.Weekday() {
			case time.Monday:
				offsetFromSunday = 1
				offsetToSaturday = 5
				break
			case time.Tuesday:
				offsetFromSunday = 2
				offsetToSaturday = 4
				break
			case time.Wednesday:
				offsetFromSunday = 3
				offsetToSaturday = 3
				break
			case time.Thursday:
				offsetFromSunday = 4
				offsetToSaturday = 2
				break
			case time.Friday:
				offsetFromSunday = 5
				offsetToSaturday = 1
				break
			case time.Saturday:
				offsetFromSunday = 6
				offsetToSaturday = 0
				break
			case time.Sunday:
				offsetFromSunday = 0
				offsetToSaturday = 6
				break
			}
			if isStartTime {
				result = time.Date(dayOfDesiredWeek.Year(), dayOfDesiredWeek.Month(), dayOfDesiredWeek.Day()-offsetFromSunday, 0, 0, 0, 0, location)
			} else {
				result = time.Date(dayOfDesiredWeek.Year(), dayOfDesiredWeek.Month(), dayOfDesiredWeek.Day()+offsetToSaturday+1, 0, 0, 0, 0, location)
				result = result.Add(-1 * time.Millisecond)
			}
		} else {
			result = dayOfDesiredWeek
		}
		break
	case "d":
		result = timeNow.AddDate(0, 0, i)
		if rounded {
			if isStartTime {
				result = time.Date(result.Year(), result.Month(), result.Day(), 0, 0, 0, 0, location)
			} else {
				result = time.Date(result.Year(), result.Month(), result.Day()+1, 0, 0, 0, 0, location)
				result = result.Add(-1 * time.Millisecond)
			}
		}
	case "h":
		result = timeNow.Add(time.Duration(i) * time.Hour)
		if rounded {
			if isStartTime {
				result = time.Date(result.Year(), result.Month(), result.Day(), result.Hour(), 0, 0, 0, location)
			} else {
				result = time.Date(result.Year(), result.Month(), result.Day(), result.Hour()+1, 0, 0, 0, location)
				result = result.Add(-1 * time.Millisecond)
			}
		}
		break
	case "m":
		result = timeNow.Add(time.Duration(i) * time.Minute)
		if rounded {
			if isStartTime {
				result = time.Date(result.Year(), result.Month(), result.Day(), result.Hour(), result.Minute(), 0, 0, location)
			} else {
				result = time.Date(result.Year(), result.Month(), result.Day(), result.Hour(), result.Minute()+1, 0, 0, location)
				result = result.Add(-1 * time.Millisecond)
			}
		}
		break
	case "s":
		result = timeNow.Add(time.Duration(i) * time.Second)
		if rounded {
			if isStartTime {
				result = time.Date(result.Year(), result.Month(), result.Day(), result.Hour(), result.Minute(), result.Second(), 0, location)
			} else {
				result = time.Date(result.Year(), result.Month(), result.Day(), result.Hour(), result.Minute(), result.Second()+1, 0, location)
				result = result.Add(-1 * time.Millisecond)
			}
		}
		break
	default:
		log.Error("Error parsing time filterClause expression: unknown period format '" + period + "'")
	}
	return result, nil
}
