package clients

import (
	"github.com/markel1974/gokuery/src/config"
	"github.com/markel1974/gokuery/src/context"
	"github.com/markel1974/gokuery/src/nodes"
	"github.com/rs/zerolog/log"
)

// = buildEsQuery method in /kibana/src/plugins/data/common/es_query/es_query/build_es_query.ts
func BuildEsQuery(storedQuery KSavedObject) EsQuery {
	esQuery := EsQuery{Bool: &EsQueryBool{}}
	if storedQuery.Attributes == nil {
		return esQuery
	}

	if FilterHostsWithLucene != "" {
		esQuery.Bool.Must = append(esQuery.Bool.Must, EsQuery{Str: &EsQueryStr{
			Query:           FilterHostsWithLucene,
			AnalyzeWildcard: true,
		}})
	}

	if storedQuery.Attributes.Query != nil &&
		storedQuery.Attributes.Query.Query != "" {
		switch storedQuery.Attributes.Query.Language {
		case "kuery":
			/* similar to Kibana, parse KQL string and use as 1st filter member */
			q, err := nodes.ParseKueryString(storedQuery.Attributes.Query.Query,
				nil, config.New(), context.New())
			if err == nil {
				/* got parsed query organized in nested map[string]any similar to EsQuery type
				with `.bool.minimum_should_match:1` and filled `.bool.should` members */
				esQuery.Bool.Filter = append(esQuery.Bool.Filter, q)
			} else {
				log.Err(err).
					Str("title", storedQuery.Attributes.Title).
					Str("kuery", storedQuery.Attributes.Query.Query).
					Msg("could not parse KQL")
			}
		case "lucene":
			/* similar to Kibana, create query_string and use as 1st must member */
			qs := EsQuery{Str: &EsQueryStr{
				Query:           storedQuery.Attributes.Query.Query,
				AnalyzeWildcard: true,
			}}
			esQuery.Bool.Must = append(esQuery.Bool.Must, qs)
		}
	}

	if storedQuery.Attributes.Filters != nil {
		queryFromFilters := buildQueryFromFilters(storedQuery.Attributes.Filters)
		esQuery.Bool.Must = append(esQuery.Bool.Must, queryFromFilters.Must...)
		esQuery.Bool.Filter = append(esQuery.Bool.Filter, queryFromFilters.Filter...)
		esQuery.Bool.Should = append(esQuery.Bool.Should, queryFromFilters.Should...)
		esQuery.Bool.MustNot = append(esQuery.Bool.MustNot, queryFromFilters.MustNot...)
	}

	if storedQuery.Attributes.TimeFilter != nil {
		esQuery.Bool.Filter = append(esQuery.Bool.Filter,
			buildRangeFilterFromTimeFilter(*storedQuery.Attributes.TimeFilter))
	}

	return esQuery
}

// = buildQueryFromFilters method in /kibana/src/plugins/data/common/es_query/es_query/from_filters.ts
func buildQueryFromFilters(filters []KFilter) EsQueryBool {
	var esQueryBool EsQueryBool

	// TODO: check if it's needed
	// remove disabled filters
	// var enabledFilters []KFilter
	// for _, filter := range filters {
	// 	 if !isFilterDisabled(filter) {
	//	 	 enabledFilters = append(enabledFilters, filter)
	//	 }
	// }

	esQueryBool.Filter = filtersToESQueries(filters, false)
	esQueryBool.MustNot = filtersToESQueries(filters, true)
	return esQueryBool
}

// = buildQueryFromFilters.filtersToESQueries in /kibana/src/plugins/data/common/es_query/es_query/from_filters.ts
func filtersToESQueries(filters []KFilter, negate bool) []any {
	// filter negate
	var negateFilters []KFilter
	for _, filter := range filters {
		if filterNegate(filter, negate) {
			negateFilters = append(negateFilters, filter)
		}
	}

	// no need to filter by index

	// migrate filters
	var migratedFilters = make([]KFilter, 0, len(negateFilters))
	for _, filter := range negateFilters {
		migratedFilters = append(migratedFilters, migrateDeprecatedPhraseFilter(filter))
	}

	// TODO nested filters are not supported

	var translatedFilters = make([]any, 0, len(migratedFilters))
	for _, filter := range migratedFilters {
		translatedFilters = append(translatedFilters, translateToQuery(filter))
	}

	var cleanedFilters = make([]any, 0, len(translatedFilters))
	for _, filter := range translatedFilters {
		cleanedFilters = append(cleanedFilters, cleanFilter(filter))
	}

	return cleanedFilters
}

// func isFilterDisabled(filter KFilter) bool {
// 	if filter.Meta != nil && filter.Meta.Disabled != nil {
// 		return *filter.Meta.Disabled
// 	}
// 	return false
// }

func filterNegate(filter KFilter, reverse bool) bool {
	if filter.Meta == nil || filter.Meta.Negate == nil {
		return !reverse
	}
	return *filter.Meta.Negate == reverse
}

// combined migrateFilter and isDeprecatedPhraseFilter in
// /kibana/src/plugins/data/common/es_query/es_query/migrate_filter.ts
func migrateDeprecatedPhraseFilter(filter KFilter) KFilter {
	if filter.Query != nil {
		switch filter.Query.(type) {
		case map[string]any:
			filterQuery := filter.Query.(map[string]any)
			if matchValue, has := filterQuery["match"]; has {
				switch matchValue.(type) {
				case map[string]any:
					queryBody := matchValue.(map[string]any)
					var fieldName string
					for k := range queryBody {
						fieldName = k
						break
					}
					switch queryBody[fieldName].(type) {
					case map[string]any:
						field := queryBody[fieldName].(map[string]any)
						if fieldType, has := field["type"]; has {
							switch fieldType.(type) {
							case string:
								if fieldType.(string) == "phrase" {
									query := queryBody[fieldName].(map[string]any)["query"]
									filter.Query = buildMatchPhraseFilter(fieldName, query)
								}
							}
						}
					}
				}
			}
		}
	}
	return filter
}

// = translateToQuery in /kibana/src/plugins/data/common/es_query/es_query/from_filters.ts
func translateToQuery(filter KFilter) any {
	if filter.Query != nil {
		return filter.Query
	}
	return filter
}

// = cleanFilter in /kibana/src/plugins/data/common/es_query/filters/index.ts
func cleanFilter(filter any) any {
	switch filter := filter.(type) {
	case KFilter:
		cleanedFilter := filter
		cleanedFilter.Meta = nil
		return cleanedFilter
	}
	return filter
}

// to convert timefilter to UTC
func buildRangeFilterFromTimeFilter(timeFilter KTimeFilter) KFilter {
	timestamp := Timestamp{
		From: timeFilter.From,
		To:   timeFilter.To,
	}
	rangeFilter := map[string]any{
		"@timestamp": timestamp.toAbsoluteUtcTime(),
	}
	return KFilter{
		Range: rangeFilter,
	}
}

// to replace deprecated match phrase filter and assign our hostName filter
func buildMatchPhraseFilter(fieldName string, value any) any {
	query := map[string]any{
		"match_phrase": map[string]any{
			fieldName: map[string]any{
				"query": value,
			},
		},
	}
	return query
}
