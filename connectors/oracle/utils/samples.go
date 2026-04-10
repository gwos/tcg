package utils

import (
	"context"
	"fmt"
	"strings"
	"time"

	ociCom "github.com/oracle/oci-go-sdk/v65/common"
	ociMon "github.com/oracle/oci-go-sdk/v65/monitoring"
)

type sample struct {
	HostName    string
	ServiceName string
	Value       float64
	StartTime   time.Time
	EndTime     time.Time
	NoData      bool
}

func ListSamples(
	ctx context.Context, client ociMon.MonitoringClient, compartment compartment, definition definition, interval time.Duration,
) ([]sample, error) {
	endTime := time.Now().UTC().Truncate(time.Minute)
	startTime := endTime.Add(-interval).Add(-1 * time.Minute)
	end := ociCom.SDKTime{Time: endTime}
	start := ociCom.SDKTime{Time: startTime}
	query := fmt.Sprintf("%s[%dm].sum()", definition.Name, int(interval.Minutes()))
	resolution := fmt.Sprintf("%dm", int(interval.Minutes()))

	resp, err := client.SummarizeMetricsData(ctx, ociMon.SummarizeMetricsDataRequest{
		CompartmentId: ociCom.String(compartment.ID),
		SummarizeMetricsDataDetails: ociMon.SummarizeMetricsDataDetails{
			Namespace:  ociCom.String(definition.Namespace),
			Query:      ociCom.String(query),
			Resolution: ociCom.String(resolution),
			StartTime:  &start,
			EndTime:    &end,
		},
	})
	if err != nil {
		return nil, err
	}

	result := make([]sample, 0, len(resp.Items))
	if len(resp.Items) == 0 {
		hostName, ok := getHostName(definition.Dimensions)
		if !ok {
			return nil, nil
		}
		return []sample{
			{
				HostName:    hostName,
				ServiceName: definition.Name,
				Value:       0,
				StartTime:   endTime,
				EndTime:     endTime,
				NoData:      true,
			},
		}, nil
	}

	for _, item := range resp.Items {
		serviceName := definition.Name
		if item.Name != nil && strings.TrimSpace(*item.Name) != "" {
			serviceName = strings.TrimSpace(*item.Name)
		}

		tags := item.Dimensions
		if len(tags) == 0 {
			tags = definition.Dimensions
		}
		hostName, ok := getHostName(tags)
		if !ok {
			continue
		}

		value, pointTime, ok := getValue(item.AggregatedDatapoints)
		if !ok {
			value = 0
			pointTime = endTime
		}

		result = append(result, sample{
			HostName:    hostName,
			ServiceName: serviceName,
			Value:       value,
			StartTime:   pointTime,
			EndTime:     pointTime,
			NoData:      !ok,
		})
	}

	return result, nil
}

func getValue(datapoints []ociMon.AggregatedDatapoint) (float64, time.Time, bool) {
	var (
		hasValue bool
		maxTime  time.Time
		value    float64
	)
	for _, datapoint := range datapoints {
		if datapoint.Timestamp == nil || datapoint.Value == nil {
			continue
		}
		if !hasValue || datapoint.Timestamp.Time.After(maxTime) {
			hasValue = true
			maxTime = datapoint.Timestamp.Time
			value = *datapoint.Value
		}
	}
	return value, maxTime, hasValue
}
