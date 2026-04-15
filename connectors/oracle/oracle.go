package oracle

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	ociCom "github.com/oracle/oci-go-sdk/v65/common"
	ociIde "github.com/oracle/oci-go-sdk/v65/identity"
	ociMon "github.com/oracle/oci-go-sdk/v65/monitoring"
	"github.com/rs/zerolog/log"

	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/oracle/utils"
	"github.com/gwos/tcg/sdk/transit"
)

type serviceMetricState struct {
	metricBuilder connectors.MetricBuilder
	dimensions    map[string]string
	noData        bool
	endTime       time.Time
}

var hostDimensionKeys = map[string]struct{}{
	"resourcedisplayname": {},
	"resourcename":        {},
	"hostname":            {},
	"host":                {},
	"nodename":            {},
	"instanceid":          {},
	"resourceid":          {},
}

func collectMetrics() {
	cfg, cfgVersion, ctx := extConfig, configVersion.Load(), ctxCancel

	if cfg.OracleTenancyOCID == "" || cfg.OracleUserOCID == "" ||
		cfg.OraclePrivateKey == "" || cfg.OracleFingerprint == "" || cfg.OracleRegion == "" {
		log.Error().Msg("failed to create oracle identity client: missing required config parameters")
		return
	}

	if len(cfg.GWMapping.Host) == 0 || len(cfg.GWMapping.Service) == 0 {
		return
	}

	provider := ociCom.NewRawConfigurationProvider(
		cfg.OracleTenancyOCID,
		cfg.OracleUserOCID,
		cfg.OracleRegion,
		cfg.OracleFingerprint,
		cfg.OraclePrivateKey,
		nil,
	)

	ideClient, err := ociIde.NewIdentityClientWithConfigurationProvider(provider)
	if err != nil {
		log.Error().Err(err).Msg("failed to create oracle identity client")
		return
	}
	ideClient.SetRegion(cfg.OracleRegion)

	monClient, err := ociMon.NewMonitoringClientWithConfigurationProvider(provider)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			log.Error().Err(err).Msg("failed to create oracle monitoring client")
		}
		return
	}
	monClient.SetRegion(cfg.OracleRegion)

	compartments, err := utils.ListCompartments(ctx, ideClient, cfg.OracleTenancyOCID)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			log.Error().Err(err).Msg("failed to list oracle compartments")
		}
		return
	}

	serviceMetricsByResource := make(map[string]map[string]map[string]serviceMetricState)
	resourceGroupByHost := make(map[string]string)
	for _, compartment := range compartments {
		definitions, err := utils.ListDefinitions(ctx, monClient, compartment.ID)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			log.Error().Err(err).
				Str("compartment_id", compartment.ID).
				Str("compartment_name", compartment.Name).
				Msg("failed to list oracle metric definitions")
			continue
		}

		for _, definition := range definitions {
			if !cfg.GWMapping.Service.MatchString(definition.Name) {
				continue
			}

			samples, err := utils.ListSamples(ctx, monClient, compartment, definition, cfg.CheckInterval)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				log.Error().Err(err).
					Str("compartment_id", compartment.ID).
					Str("namespace", definition.Namespace).
					Str("metric_name", definition.Name).
					Msg("failed to list oracle metric samples")
				continue
			}

			for _, sample := range samples {
				if !cfg.GWMapping.Host.MatchString(sample.HostName) {
					continue
				}

				seriesKey := buildDimensionSeriesKey(sample.Dimensions)
				metricBuilder := connectors.MetricBuilder{
					Name:       sample.ServiceName,
					CustomName: sample.ServiceName,
					Value:      sample.Value,
					UnitType:   transit.UnitCounter,
					StartTimestamp: &transit.Timestamp{
						Time: sample.StartTime,
					},
					EndTimestamp: &transit.Timestamp{
						Time: sample.EndTime,
					},
				}

				hostServices, exists := serviceMetricsByResource[sample.HostName]
				if !exists {
					hostServices = make(map[string]map[string]serviceMetricState)
					serviceMetricsByResource[sample.HostName] = hostServices
				}
				serviceMetrics, exists := hostServices[sample.ServiceName]
				if !exists {
					serviceMetrics = make(map[string]serviceMetricState)
					hostServices[sample.ServiceName] = serviceMetrics
				}
				state, exists := serviceMetrics[seriesKey]
				if !exists || sample.EndTime.After(state.endTime) ||
					(sample.EndTime.Equal(state.endTime) && state.noData && !sample.NoData) {
					serviceMetrics[seriesKey] = serviceMetricState{
						metricBuilder: metricBuilder,
						dimensions:    sample.Dimensions,
						noData:        sample.NoData,
						endTime:       sample.EndTime,
					}
				}

				if _, exists := resourceGroupByHost[sample.HostName]; !exists {
					groupName := strings.TrimSpace(compartment.Name)
					if groupName == "" {
						groupName = strings.TrimSpace(compartment.ID)
					}
					resourceGroupByHost[sample.HostName] = groupName
				}
			}
		}
	}

	resourceNames := make([]string, 0, len(serviceMetricsByResource))
	for resourceName := range serviceMetricsByResource {
		resourceNames = append(resourceNames, resourceName)
	}
	sort.Strings(resourceNames)

	mResources := make([]transit.MonitoredResource, 0, len(resourceNames))
	resourceRefsByGroup := make(map[string][]transit.ResourceRef)
	for _, resourceName := range resourceNames {
		hostServices := serviceMetricsByResource[resourceName]
		if len(hostServices) == 0 {
			continue
		}

		serviceNames := make([]string, 0, len(hostServices))
		for serviceName := range hostServices {
			serviceNames = append(serviceNames, serviceName)
		}
		sort.Strings(serviceNames)

		services := make([]transit.MonitoredService, 0, len(serviceNames))
		for _, serviceName := range serviceNames {
			metricsBySeriesKey := hostServices[serviceName]
			if len(metricsBySeriesKey) == 0 {
				continue
			}

			seriesKeys := make([]string, 0, len(metricsBySeriesKey))
			for seriesKey := range metricsBySeriesKey {
				seriesKeys = append(seriesKeys, seriesKey)
			}
			sort.Strings(seriesKeys)

			metricBuilders := make([]connectors.MetricBuilder, 0, len(seriesKeys))
			noDataCount := 0
			multiSeries := len(seriesKeys) > 1
			usedMetricNames := make(map[string]int)
			for _, seriesKey := range seriesKeys {
				state := metricsBySeriesKey[seriesKey]
				metricBuilder := state.metricBuilder
				metricName := serviceName
				if multiSeries {
					metricName = buildDimensionMetricName(serviceName, state.dimensions)
					if metricName == "" {
						metricName = serviceName
					}
				}
				if count := usedMetricNames[metricName]; count > 0 {
					usedMetricNames[metricName] = count + 1
					metricName = fmt.Sprintf("%s_%d", metricName, count+1)
				} else {
					usedMetricNames[metricName] = 1
				}
				metricBuilder.Name = metricName
				metricBuilder.CustomName = metricName
				metricBuilders = append(metricBuilders, metricBuilder)
				if state.noData {
					noDataCount++
				}
			}

			service, err := connectors.BuildServiceForMetrics(serviceName, resourceName, metricBuilders)
			if err != nil {
				log.Error().Err(err).
					Str("host", resourceName).
					Str("service", serviceName).
					Msg("failed to build service for metrics")
				continue
			}
			service.LastPluginOutput = buildServiceLastPluginOutput(serviceName, cfg.CheckInterval, metricBuilders, noDataCount)
			services = append(services, *service)
		}

		if len(services) == 0 {
			continue
		}

		mResource, err := connectors.CreateResource(resourceName, services)
		if err != nil {
			log.Error().Err(err).
				Str("resource_name", resourceName).
				Msg("failed to create oracle resource")
			continue
		}
		mResources = append(mResources, *mResource)
		groupName := resourceGroupByHost[resourceName]
		if groupName == "" {
			continue
		}
		resourceRefsByGroup[groupName] = append(
			resourceRefsByGroup[groupName],
			connectors.CreateResourceRef(resourceName, "", transit.ResourceTypeHost),
		)
	}

	if len(mResources) == 0 {
		log.Debug().Msg("oracle connector collected no resources, skip sending empty payload")
		return
	}

	if cfgVersion != configVersion.Load() {
		log.Debug().
			Uint64("thread_config_version", cfgVersion).
			Uint64("current_config_version", configVersion.Load()).
			Msg("skip sending stale oracle metrics")
		return
	}

	groupNames := make([]string, 0, len(resourceRefsByGroup))
	for groupName := range resourceRefsByGroup {
		groupNames = append(groupNames, groupName)
	}
	sort.Strings(groupNames)

	resourceGroups := make([]transit.ResourceGroup, 0, len(groupNames))
	for _, groupName := range groupNames {
		refs := resourceRefsByGroup[groupName]
		if len(refs) == 0 {
			continue
		}
		sort.Slice(refs, func(i, j int) bool {
			return refs[i].Name < refs[j].Name
		})
		resourceGroups = append(
			resourceGroups,
			connectors.CreateResourceGroup(groupName, "", transit.HostGroup, refs),
		)
	}

	if err = connectors.SendMetrics(ctx, mResources, &resourceGroups); err != nil {
		if !errors.Is(err, context.Canceled) {
			log.Error().Err(err).Msg("failed to send oracle metrics")
		}
	}
}

func buildServiceLastPluginOutput(serviceName string, interval time.Duration, metricBuilders []connectors.MetricBuilder, noDataCount int) string {
	metricCount := len(metricBuilders)
	if metricCount == 0 {
		return fmt.Sprintf("%s sum(%dm)=0 (no metric series)", serviceName, int(interval.Minutes()))
	}
	if metricCount == 1 {
		valueText := formatMetricBuilderValue(metricBuilders[0].Value)
		if noDataCount == 1 {
			return fmt.Sprintf(
				"%s sum(%dm)=%s (no metrics found for the selected period; defaulting to %s)",
				serviceName, int(interval.Minutes()), valueText, valueText,
			)
		}
		return fmt.Sprintf("%s sum(%dm)=%s", serviceName, int(interval.Minutes()), valueText)
	}
	if noDataCount == metricCount {
		return fmt.Sprintf(
			"%s series(%dm)=%d (all series have no data; defaulting to 0)",
			serviceName, int(interval.Minutes()), metricCount,
		)
	}
	if noDataCount > 0 {
		return fmt.Sprintf(
			"%s series(%dm)=%d (%d no-data series defaulted to 0)",
			serviceName, int(interval.Minutes()), metricCount, noDataCount,
		)
	}
	return fmt.Sprintf("%s series(%dm)=%d", serviceName, int(interval.Minutes()), metricCount)
}

func formatMetricBuilderValue(value any) string {
	switch v := value.(type) {
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	default:
		return fmt.Sprintf("%v", value)
	}
}

func buildDimensionSeriesKey(dimensions map[string]string) string {
	parts := buildDimensionParts(dimensions)
	if len(parts) == 0 {
		return "__default__"
	}
	return strings.Join(parts, "__")
}

func buildDimensionMetricName(baseName string, dimensions map[string]string) string {
	baseName = strings.TrimSpace(baseName)
	if baseName == "" {
		return baseName
	}

	parts := buildDimensionParts(dimensions)
	if len(parts) == 0 {
		return baseName
	}

	return connectors.SanitizeString(baseName + "__" + strings.Join(parts, "__"))
}

func buildDimensionParts(dimensions map[string]string) []string {
	if len(dimensions) == 0 {
		return nil
	}

	keys := make([]string, 0, len(dimensions))
	for key, value := range dimensions {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		if _, skip := hostDimensionKeys[strings.ToLower(key)]; skip {
			continue
		}
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return nil
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		value := strings.TrimSpace(dimensions[key])
		if value == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s_%s", key, value))
	}
	return parts
}
