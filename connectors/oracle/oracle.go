package oracle

import (
	"sort"
	"time"

	ociCom "github.com/oracle/oci-go-sdk/v65/common"
	ociIde "github.com/oracle/oci-go-sdk/v65/identity"
	ociMon "github.com/oracle/oci-go-sdk/v65/monitoring"
	"github.com/rs/zerolog/log"

	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/oracle/utils"
	"github.com/gwos/tcg/sdk/transit"
)

const (
	defaultHostGroupName        = "TCG-ORACLE"
	defaultHostGroupDescription = "Default host group created by the oracle connector"
)

func collectMetrics() {
	for len(config.GetConfig().Connector.AgentID) == 0 {
		time.Sleep(1 * time.Second)
	}

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
				}
				service, err := connectors.BuildServiceForMetric(sample.HostName, metricBuilder)
				if err != nil {
					log.Error().Err(err).
						Str("host", sample.HostName).
						Str("service", sample.ServiceName).
						Msg("failed to build service for metric")
					continue
				}
				servicesByResource[sample.HostName] = append(servicesByResource[sample.HostName], *service)
			}
		}
	}

	resourceNames := make([]string, 0, len(servicesByResource))
	for resourceName := range servicesByResource {
		resourceNames = append(resourceNames, resourceName)
	}
	sort.Strings(resourceNames)

	mResources := make([]transit.MonitoredResource, 0, len(resourceNames))
	mResourcesRef := make([]transit.ResourceRef, 0, len(resourceNames))
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
		mResourcesRef = append(
			mResourcesRef,
			connectors.CreateResourceRef(resourceName, "", transit.ResourceTypeHost),
		)
	}

	if len(mResources) == 0 {
		log.Debug().Msg("oracle connector collected no resources, skip sending empty payload")
		return
	}

	if extConfig.HostGroup == "" {
		extConfig.HostGroup = defaultHostGroupName
	}
	resourceGroups := []transit.ResourceGroup{
		connectors.CreateResourceGroup(extConfig.HostGroup, defaultHostGroupDescription, transit.HostGroup, mResourcesRef),
	}

	if err = connectors.SendMetrics(ctxCancel, mResources, &resourceGroups); err != nil {
		log.Error().Err(err).Msg("failed to send oracle metrics")
	}
}
