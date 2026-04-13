package main

import (
	"context"
	"sort"
	"time"

	"github.com/gwos/tcg/sdk/mapping"
	ociCom "github.com/oracle/oci-go-sdk/v65/common"
	ociIde "github.com/oracle/oci-go-sdk/v65/identity"
	ociMon "github.com/oracle/oci-go-sdk/v65/monitoring"
	"github.com/rs/zerolog/log"

	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/oracle"
	"github.com/gwos/tcg/connectors/oracle/utils"
	"github.com/gwos/tcg/sdk/transit"
)

var extConfig = &oracle.ExtConfig{
	OracleTenancyOCID: "ocid1.tenancy.oc1..aaaaaaaagtixojupjfgpjbykxnqaq3pnkq23b4fiwqaaskapudqhdhlqvjlq",
	OracleUserOCID:    "ocid1.user.oc1..aaaaaaaam7hlqsedfhw2rr5ndzuffmgrhusez5gatlji46oc7lpp23i34muq",
	OraclePrivateKey:  "-----BEGIN PRIVATE KEY-----\nMIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQDQhxdK2Cfb90Bs\nPQ5VhKc0H0UOHOQ77hU6FA0ZUH3fK2o5/DEliPDtzhKahqSxQaz2j9dq3o9fLPOj\no8FPAlORHBqX3iOb7Gq+HKL7mxAhzYses1mFoIcu7hnP4CnZmYBpXPKth9QsSJnV\ndzeendfUf2bRbISF8470m3Uhu1BZkSG3DFWrdEQ9I8JFvkJ8aASO1Pyky98ed11p\n+iHZaXip0A756UWVD6EIO9mQNZRdXHfASF3kRcVPAf1qvr5z71JzawMIVd2HwcdF\nhE7O9xjpIqMfQsAsku42KFDKu0As1QRxw6rnUO2jbpUwVUtOUk8ePLrgQtC2KOY5\nfiFDNZOlAgMBAAECggEASt6+J7K9ePZo7JPtchoLWKUDi8Im1jew6mXpoBWs4/R4\nEvKwCKyG6LMtLcs7FaOhgPN7YlUfgAopPi9dCEurCnZfO/jXqzOyzaiFgjYWEDT3\nBuJZOae98cUuglsXW5gIKYxkW5bhwLaeeSzxTOOaayMFHRtL57ZftQaeAyr4efeB\n0mpcWqO5/BC/V4+7GKbHI3tG4a2mUMOnhMzB0Sua2HnrxVVd5Pe8n+JgcMTg7j2d\nJ8S313fnpLPtsSZKJ+dDbLxcoak5sB1nEr/oU9yxWHr7izlpWBLH9+4ThdNvIhTQ\n1VmwFS60CVUE0RgKVfMl4gnVAp/gctFJUaEudzTNLwKBgQD9rT66AYFZaErbjdH2\nhEJFn4R5obk1XBayXKHrgzLmLeekUhg/KprAMJlLrG/zBPhQk+EWtt13yg3i2AfI\nXM8auRf4+Q1AY+MM8WOei6dD+OWCiWdlQuNo2pgSnZdlUCv4zbNfGfQzr2EvB/Zt\nNVyKHted2Q5QwdhAiFrBzZBz1wKBgQDSb/4GOoD/Kt/WgpR/XCMHO1BruDBhasEk\n/A3Wt7i1Bol2j3SwM/ynRauR4B+3O8h5V5XlPll9OYqg00NQeZ/PESwaX43bp8s4\nB1clH0MtOqalsv+Bx64uquQMU9mLADcP2XRdzyYWUH6svLmWRs/VpCg8lEOxNvBC\nQ+AZNjyE4wKBgBpn/E2Udoh+CLzOCHsmzVc+AaY/pW3ehiZO6jP/1j6LrL84JHn+\nz1kONgbgjk63x7lri1S3+FnN2KAyaKz8rDpV5h7uneiD/VCNmBca5nB26j0qXG74\nBYAWsRnO+cF8FPezQea2ZusyaGVi6M35bgaaq0stGwZhB0fAaeIeqdjFAoGAddbm\n3eAG+lys6bdHpqYWq2cImrmSxgp8y9Qlf7ZzxVM6yNx+UGlcMuMtt0tVF0tv8Jse\nQjgO7rO5MGP9TuQ8nDyWgNI/YuBsSRy7LPt7p6kvRpycvbTukg16FHkj2dWt/78a\njamBP3+l397y4fcXTSBWs82mtmb4VXMi25tmelcCgYB7EQqQUNAwXOTPSIgY7Al3\nUDY79l1w979Svml6FKQVdJgYMelPyAuYP+9gN5+tKQVG/wnXYD8ajuJX1zh1Tcg/\nZsK/W8RxdmoQt7aVa5fmSvaeYdMc40a6Y/dj0scRIYrFxG7o+1Wg6sekFqlMxfPI\nu/mHddUc7bx3esqk61leHw==\n-----END PRIVATE KEY-----\nOCI_API_KEY",
	OracleFingerprint: "a7:68:ab:22:61:36:b4:86:23:6c:52:66:aa:91:b0:17",
	OracleRegion:      "ap-mumbai-1",
	CheckInterval:     1 * time.Minute,
}

func main() {
	extConfig.GWMapping = oracle.GWMapping{
		Host: []mapping.Mapping{
			{
				Matcher: "gwos-block.*",
			},
		},
		Service: []mapping.Mapping{
			{
				Matcher: ".*",
			},
		},
	}

	extConfig.GWMapping.Prepare()

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

	compartments, err := utils.ListCompartments(context.Background(), ideClient, extConfig.OracleTenancyOCID)
	if err != nil {
		log.Error().Err(err).Msg("failed to list oracle compartments")
		return
	}

	servicesByResource := make(map[string][]transit.MonitoredService)
	for _, compartment := range compartments {
		definitions, err := utils.ListDefinitions(context.Background(), monClient, compartment.ID)
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

			samples, err := utils.ListSamples(context.Background(), monClient, compartment, definition, extConfig.CheckInterval)
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
		extConfig.HostGroup = "TEST"
	}
	resourceGroups := []transit.ResourceGroup{
		connectors.CreateResourceGroup(extConfig.HostGroup, "TEST", transit.HostGroup, mResourcesRef),
	}

	log.Info().
		Interface("resources", mResources).
		Interface("resource_groups", resourceGroups).
		Msg("sending oracle metrics")
}
