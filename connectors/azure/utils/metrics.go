package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"

	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/sdk/transit"
)

func MetricsList(
	cred *azidentity.DefaultAzureCredential,
	subscriptionID string,
	resource *armresources.GenericResourceExpanded,
	definitions []string,
) ([]armmonitor.MetricsClientListResponse, error) {
	client, err := armmonitor.NewMetricsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to init client: %w", err)
	}

	result := make([]armmonitor.MetricsClientListResponse, 0)
	timespan := fmt.Sprintf(
		"%s/%s",
		time.Now().Add(-time.Minute).Format("2006-01-02T15:04:05"),
		time.Now().Format("2006-01-02T15:04:05"),
	)
	for _, definition := range definitions {
		metric, err := client.List(context.Background(), *resource.ID, &armmonitor.MetricsClientListOptions{
			Metricnames: &definition,
			Timespan:    &timespan,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list metrics: %w", err)
		}
		if len(metric.Value) == 0 {
			continue
		}
		result = append(result, metric)
	}

	return result, nil
}

func CreateMetricBuilder(metric armmonitor.MetricsClientListResponse) connectors.MetricBuilder {
	var value float64 = 0
	if len(metric.Value[0].Timeseries) == 0 || len(metric.Value[0].Timeseries[0].Data) == 0 {
		value = 0
	} else {
		if metric.Value[0].Timeseries[0].Data[len(metric.Value[0].Timeseries[0].Data)-1].Maximum != nil {
			value = *metric.Value[0].Timeseries[0].Data[len(metric.Value[0].Timeseries[0].Data)-1].Maximum
		}
		if metric.Value[0].Timeseries[0].Data[len(metric.Value[0].Timeseries[0].Data)-1].Total != nil {
			value = *metric.Value[0].Timeseries[0].Data[len(metric.Value[0].Timeseries[0].Data)-1].Total
		}
		if metric.Value[0].Timeseries[0].Data[len(metric.Value[0].Timeseries[0].Data)-1].Average != nil {
			value = *metric.Value[0].Timeseries[0].Data[len(metric.Value[0].Timeseries[0].Data)-1].Average
		}
		if metric.Value[0].Timeseries[0].Data[len(metric.Value[0].Timeseries[0].Data)-1].Count != nil {
			value = *metric.Value[0].Timeseries[0].Data[len(metric.Value[0].Timeseries[0].Data)-1].Count
		}
	}
	return connectors.MetricBuilder{
		Name:       *metric.Value[0].Name.Value,
		CustomName: *metric.Value[0].Name.Value,
		Value:      value,
		UnitType:   transit.UnitCounter,
	}
}
