package clients

// = buildEsQuery method in /kibana/src/plugins/data/common/es_query/es_query/build_es_query.ts
func BuildEsQuery(storedQuery KSavedObject) EsQuery {
	var esQuery EsQuery

	// TODO kueryQuery := buildQueryFromKuery; append
	// TODO luceneQuery := buildQueryFromLucene; append

	if storedQuery.Attributes == nil {
		return esQuery
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

	// remove disabled filters
	var enabledFilters []KFilter
	for _, filter := range filters {
		if !isFilterDisabled(filter) {
			enabledFilters = append(enabledFilters, filter)
		}
	}

	esQueryBool.Filter = filtersToESQueries(filters, false)
	esQueryBool.MustNot = filtersToESQueries(filters, true)
	return esQueryBool
}

// = buildQueryFromFilters.filtersToESQueries in /kibana/src/plugins/data/common/es_query/es_query/from_filters.ts
func filtersToESQueries(filters []KFilter, negate bool) []interface{} {
	// filter negate
	var negateFilters []KFilter
	for _, filter := range filters {
		if filterNegate(filter, negate) {
			negateFilters = append(negateFilters, filter)
		}
	}

	// no need to filter by index

	// migrate filters
	var migratedFilters []KFilter
	for _, filter := range negateFilters {
		migratedFilters = append(migratedFilters, migrateDeprecatedPhraseFilter(filter))
	}
	negateFilters = nil

	// TODO nested filters are not supported

	var translatedFilters []interface{}
	for _, filter := range migratedFilters {
		translatedFilters = append(translatedFilters, translateToQuery(filter))
	}
	migratedFilters = nil

	var cleanedFilters []interface{}
	for _, filter := range translatedFilters {
		cleanedFilters = append(cleanedFilters, cleanFilter(filter))
	}

	return cleanedFilters
}

func isFilterDisabled(filter KFilter) bool {
	if filter.Meta != nil && filter.Meta.Disabled != nil {
		return *filter.Meta.Disabled
	}
	return false
}

func filterNegate(filter KFilter, reverse bool) bool {
	if filter.Meta == nil || filter.Meta.Negate == nil {
		return !reverse
	}
	return *filter.Meta.Negate == reverse
}

// combined migrateFilter and isDeprecatedPhraseFilter in
//    /kibana/src/plugins/data/common/es_query/es_query/migrate_filter.ts
func migrateDeprecatedPhraseFilter(filter KFilter) KFilter {
	if filter.Query != nil {
		switch filter.Query.(type) {
		case map[string]interface{}:
			filterQuery := filter.Query.(map[string]interface{})
			if matchValue, has := filterQuery["match"]; has {
				switch matchValue.(type) {
				case map[string]interface{}:
					queryBody := matchValue.(map[string]interface{})
					var fieldName string
					for k := range queryBody {
						fieldName = k
						break
					}
					switch queryBody[fieldName].(type) {
					case map[string]interface{}:
						field := queryBody[fieldName].(map[string]interface{})
						if fieldType, has := field["type"]; has {
							switch fieldType.(type) {
							case string:
								if fieldType.(string) == "phrase" {
									query := queryBody[fieldName].(map[string]interface{})["query"]
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
func translateToQuery(filter KFilter) interface{} {
	if filter.Query != nil {
		return filter.Query
	}
	return filter
}

// = cleanFilter in /kibana/src/plugins/data/common/es_query/filters/index.ts
func cleanFilter(filter interface{}) interface{} {
	switch filter.(type) {
	case KFilter:
		cleanedFilter := filter.(KFilter)
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
	rangeFilter := map[string]interface{}{
		"@timestamp": timestamp.toAbsoluteUtcTime(),
	}
	return KFilter{
		Range: rangeFilter,
	}
}

// to replace deprecated match phrase filter and assign our hostName filter
func buildMatchPhraseFilter(fieldName string, value interface{}) interface{} {
	query := map[string]interface{}{
		"match_phrase": map[string]interface{}{
			fieldName: map[string]interface{}{
				"query": value,
			},
		},
	}
	return query
}
