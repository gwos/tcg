package oracle

import (
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

func collectMetrics() {
	if extConfig.OracleTenancyOCID == "" || extConfig.OracleUserOCID == "" ||
		extConfig.OraclePrivateKey == "" || extConfig.OracleFingerprint == "" || extConfig.OracleRegion == "" {
		log.Error().Msg("failed to create oracle identity client: missing required config parameters")
		return
	}

	provider := ociCom.NewRawConfigurationProvider(
		extConfig.OracleTenancyOCID,
		extConfig.OracleUserOCID,
		extConfig.OracleRegion,
		extConfig.OracleFingerprint,
		extConfig.OraclePrivateKey,
		nil,
	)

	ideClient, err := ociIde.NewIdentityClientWithConfigurationProvider(provider)
	if err != nil {
		log.Error().Err(err).Msg("failed to create oracle identity client")
		return
	}
	ideClient.SetRegion(extConfig.OracleRegion)

	monClient, err := ociMon.NewMonitoringClientWithConfigurationProvider(provider)
	if err != nil {
		log.Error().Err(err).Msg("failed to create oracle monitoring client")
		return
	}
	monClient.SetRegion(extConfig.OracleRegion)

	compartments, err := utils.ListCompartments(ctxCancel, ideClient, extConfig.OracleTenancyOCID)
	if err != nil {
		log.Error().Err(err).Msg("failed to list oracle compartments")
		return
	}

	servicesByResource := make(map[string][]transit.MonitoredService)
	resourceGroupByHost := make(map[string]string)
	for _, compartment := range compartments {
		definitions, err := utils.ListDefinitions(ctxCancel, monClient, compartment.ID)
		if err != nil {
			log.Error().Err(err).
				Str("compartment_id", compartment.ID).
				Str("compartment_name", compartment.Name).
				Msg("failed to list oracle metric definitions")
			continue
		}

		for _, definition := range definitions {
			if !extConfig.GWMapping.Service.MatchString(definition.Name) {
				continue
			}

			samples, err := utils.ListSamples(ctxCancel, monClient, compartment, definition, extConfig.CheckInterval)
			if err != nil {
				log.Error().Err(err).
					Str("compartment_id", compartment.ID).
					Str("namespace", definition.Namespace).
					Str("metric_name", definition.Name).
					Msg("failed to list oracle metric samples")
				continue
			}

			for _, sample := range samples {
				if !extConfig.GWMapping.Host.MatchString(sample.HostName) {
					continue
				}

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
				service, err := connectors.BuildServiceForMetric(sample.HostName, metricBuilder)
				if err != nil {
					log.Error().Err(err).
						Str("host", sample.HostName).
						Str("service", sample.ServiceName).
						Msg("failed to build service for metric")
					continue
				}
				service.LastPluginOutput = buildServiceLastPluginOutput(sample.ServiceName, extConfig.CheckInterval, sample.Value, sample.NoData)
				servicesByResource[sample.HostName] = append(servicesByResource[sample.HostName], *service)
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

	resourceNames := make([]string, 0, len(servicesByResource))
	for resourceName := range servicesByResource {
		resourceNames = append(resourceNames, resourceName)
	}
	sort.Strings(resourceNames)

	mResources := make([]transit.MonitoredResource, 0, len(resourceNames))
	resourceRefsByGroup := make(map[string][]transit.ResourceRef)
	for _, resourceName := range resourceNames {
		services := servicesByResource[resourceName]
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

	if err = connectors.SendMetrics(ctxCancel, mResources, &resourceGroups); err != nil {
		log.Error().Err(err).Msg("failed to send oracle metrics")
	}
}

func buildServiceLastPluginOutput(serviceName string, interval time.Duration, value float64, noData bool) string {
	if noData {
		return fmt.Sprintf(
			"%s sum(%dm)=0 (no metrics found for the selected period; defaulting to 0)",
			serviceName, int(interval.Minutes()),
		)
	}
	valueText := strconv.FormatFloat(value, 'f', -1, 64)
	return fmt.Sprintf("%s sum(%dm)=%s", serviceName, int(interval.Minutes()), valueText)
}
