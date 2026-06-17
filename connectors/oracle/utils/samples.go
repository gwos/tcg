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
	ResourceID  string
	ServiceName string
	Dimensions  map[string]string
	Value       float64
	StartTime   time.Time
	EndTime     time.Time
	NoData      bool
}

func ListSamples(
	ctx context.Context, client ociMon.MonitoringClient, compartment compartment, definition definition,
	interval time.Duration, aggregation string,
) ([]sample, error) {
	aggregation = strings.TrimSpace(aggregation)
	if aggregation == "" {
		aggregation = "sum"
	}
	endTime := time.Now().UTC().Truncate(time.Minute)
	startTime := endTime.Add(-interval).Add(-1 * time.Minute)
	end := ociCom.SDKTime{Time: endTime}
	start := ociCom.SDKTime{Time: startTime}
	query := fmt.Sprintf("%s[%dm].%s()", definition.Name, int(interval.Minutes()), aggregation)
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
		hostName, hasHost := getHostName(definition.Dimensions)
		resourceID, hasID := getResourceID(definition.Dimensions)
		if !hasHost && !hasID {
			return nil, nil
		}
		return []sample{
			{
				HostName:    hostName,
				ResourceID:  resourceID,
				ServiceName: definition.Name,
				Dimensions:  cloneDimensions(definition.Dimensions),
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
		hostName, hasHost := getHostName(tags)
		resourceID, hasID := getResourceID(tags)
		if !hasHost && !hasID {
			continue
		}

		value, pointTime, ok := getValue(item.AggregatedDatapoints)
		if !ok {
			value = 0
			pointTime = endTime
		}

		result = append(result, sample{
			HostName:    hostName,
			ResourceID:  resourceID,
			ServiceName: serviceName,
			Dimensions:  cloneDimensions(tags),
			Value:       value,
			StartTime:   pointTime,
			EndTime:     pointTime,
			NoData:      !ok,
		})
	}

	return result, nil
}

func cloneDimensions(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
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
