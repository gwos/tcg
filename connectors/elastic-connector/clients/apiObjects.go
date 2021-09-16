package clients

import (
	"strconv"
	"strings"
	"time"

	"github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/transit"
	"github.com/rs/zerolog/log"
)

const (
	layout = "2006-01-02T15:04:05.000Z"
	now    = "now"
)

type KBulkGetRequest struct {
	Type string `json:"type"`
	Id   string `json:"id"`
}

type KSavedObjectsResponse struct {
	Page         int            `json:"page"`
	PerPage      int            `json:"per_page"`
	Total        int            `json:"total"`
	SavedObjects []KSavedObject `json:"saved_objects"`
}

type KSavedObject struct {
	Type       string       `json:"type"`
	ID         string       `json:"id"`
	Attributes *KAttributes `json:"attributes,omitempty"`
}

type KAttributes struct {
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Filters     []KFilter    `json:"filters,omitempty"`
	TimeFilter  *KTimeFilter `json:"timefilter,omitempty"`
}

type KFilter struct {
	Meta   *KMeta      `json:"meta,omitempty"`
	Query  interface{} `json:"query,omitempty"`
	Exists interface{} `json:"exists,omitempty"`
	Range  interface{} `json:"range,omitempty"`
}

type KMeta struct {
	Index    *string `json:"index,omitempty"`
	Negate   *bool   `json:"negate,omitempty"`
	Disabled *bool   `json:"disabled,omitempty"`
	Type     *string `json:"type,omitempty"`
}

type KTimeFilter struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Override *bool  `json:"override,omitempty"`
}

type Timestamp struct {
	From string `json:"gte"`
	To   string `json:"lt"`
}

type EsSearchBody struct {
	Query *EsQuery `json:"query,omitempty"`
	Aggs  *EsAggs  `json:"aggs,omitempty"`
}

type EsQuery struct {
	Bool EsQueryBool `json:"bool"`
}

type EsQueryBool struct {
	Must    []interface{} `json:"must"`
	Filter  []interface{} `json:"filter"`
	Should  []interface{} `json:"should"`
	MustNot []interface{} `json:"must_not"`
}

type EsAggs struct {
	Agg EsAggByHost `json:"_by_host"`
}

type EsAggByHost struct {
	Composite EsAggComposite `json:"composite"`
}

type EsAggComposite struct {
	Size    int           `json:"size"`
	Sources []interface{} `json:"sources"`
	After   interface{}   `json:"after,omitempty"`
}

type EsHostAggSource struct {
	HostTerm EsAggTerm `json:"host"`
}

type EsHostGroupAggSource struct {
	HostGroupTerm EsAggTerm `json:"host_group"`
}

type EsAggTerm struct {
	Term EsAggTermField `json:"terms"`
}

type EsAggTermField struct {
	Field string `json:"field"`
}

type EsSearchResponse struct {
	Hits         EsSearchHits        `json:"hits"`
	Aggregations EsAggregationByHost `json:"aggregations,omitempty"`
}

type EsSearchHits struct {
	Total EsSearchHitsTotal `json:"total"`
}

type EsSearchHitsTotal struct {
	Value int `json:"value"`
}

type EsAggregationByHost struct {
	Aggregation EsAggregation `json:"_by_host"`
}

type EsAggregation struct {
	AfterKey *EsAggregationKey     `json:"after_key,omitempty"`
	Buckets  []EsAggregationBucket `json:"buckets"`
}

type EsAggregationBucket struct {
	Key       EsAggregationKey `json:"key"`
	DocsCount int              `json:"doc_count"`
}

type EsAggregationKey struct {
	Host      string  `json:"host"`
	HostGroup *string `json:"host_group,omitempty"`
}

type EsFieldCapsResponse struct {
	Fields map[string]interface{} `json:"fields"`
}

// Extracts indexes's ids linked to stored query's filters
func (storedQuery *KSavedObject) ExtractIndexIds() []string {
	indexIdsSet := make(map[string]struct{})
	for _, filter := range storedQuery.Attributes.Filters {
		if filter.Meta.Index != nil {
			indexIdsSet[*filter.Meta.Index] = struct{}{}
		}
	}
	var indexIds []string
	for indexId := range indexIdsSet {
		indexIds = append(indexIds, indexId)
	}
	return indexIds
}

func BuildAggregationsByHostNameAndHostGroup(hostNameField string, hostGroupField *string) *EsAggs {
	sources := []interface{}{EsHostAggSource{
		HostTerm: EsAggTerm{
			Term: EsAggTermField{
				Field: hostNameField,
			},
		},
	}}
	if hostGroupField != nil {
		sources = append(sources, EsHostGroupAggSource{
			HostGroupTerm: EsAggTerm{
				Term: EsAggTermField{
					Field: *hostGroupField,
				},
			},
		})
	}

	return &EsAggs{
		Agg: EsAggByHost{
			Composite: EsAggComposite{
				Size:    1000,
				Sources: sources,
			},
		},
	}
}

func copyQuery(query *EsQuery) *EsQuery {
	if query != nil {
		queryCopy := &EsQuery{}
		if query.Bool.Must != nil {
			queryCopy.Bool.Must = make([]interface{}, len(query.Bool.Must))
			copy(queryCopy.Bool.Must, query.Bool.Must)
		}
		if query.Bool.MustNot != nil {
			queryCopy.Bool.MustNot = make([]interface{}, len(query.Bool.MustNot))
			copy(queryCopy.Bool.MustNot, query.Bool.MustNot)
		}
		if query.Bool.Should != nil {
			queryCopy.Bool.Should = make([]interface{}, len(query.Bool.Should))
			copy(queryCopy.Bool.Should, query.Bool.Should)
		}
		if query.Bool.Filter != nil {
			queryCopy.Bool.Filter = make([]interface{}, len(query.Bool.Filter))
			copy(queryCopy.Bool.Filter, query.Bool.Filter)
		}
		return queryCopy
	}
	return nil
}

// Modifies timestamp's from/to values to absolute time in UTC time zone
func (timestamp *Timestamp) toAbsoluteUtcTime() *Timestamp {
	from, err := toAbsoluteTime(timestamp.From, true)
	if err != nil {
		log.Err(err).Msg("could not parse 'from' timestamp")
	}
	to, err := toAbsoluteTime(timestamp.To, false)
	if err != nil {
		log.Err(err).Msg("could not parse 'to' timestamp")
	}
	timestamp.From = from.In(time.UTC).Format(layout)
	timestamp.To = to.In(time.UTC).Format(layout)
	return timestamp
}

// ToTimeInterval converts timeFilter's from/to values to Time in local time zone and returns TimeInterval with appropriate StartTime/EndTime
func (timeFilter *KTimeFilter) ToTimeInterval() *transit.TimeInterval {
	startTime, err := toAbsoluteTime(timeFilter.From, true)
	if err != nil {
		log.Err(err).Msg("could not parse time filter's 'from'")
	}
	endTime, err := toAbsoluteTime(timeFilter.To, false)
	if err != nil {
		log.Err(err).Msg("could not parse time filter's 'to'")
	}
	timeInterval := &transit.TimeInterval{
		StartTime: &milliseconds.MillisecondTimestamp{Time: startTime},
		EndTime:   &milliseconds.MillisecondTimestamp{Time: endTime},
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
	period := strings.ToLower(relativePart[len(relativePart)-1:])
	i, err := strconv.Atoi(interval)
	if operator == "-" {
		i = -i
	}
	if err != nil {
		log.Err(err).Msg("could not parse time filterClause expression")
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
		log.Error().Msgf("could not parse time filterClause expression: unknown period format '%s'", period)
	}
	return result, nil
}
