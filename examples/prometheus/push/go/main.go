package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
	"math/rand"
	"time"
)

const (
	prometheusConnectorUrl = "http://localhost:8099/api/v1"
)

var (
	headers = map[string]string{
		"GWOS-APP-NAME":  "GW8",
		"GWOS-API-TOKEN": "6aeece7b-e8c3-4aa2-8e12-a7c22382c9b5",
	}
	completionTime = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "db_backup_last_completion_timestamp_seconds",
		Help: "The timestamp of the last completion of a DB backup, successful or not.",
		ConstLabels: prometheus.Labels{
			"critical": fmt.Sprintf("%f", rand.Float64()*10),
			"warning":  fmt.Sprintf("%f", rand.Float64()+0.5),
			"resource": "Prometheus-Go-Push",
			"group":    "Prometheus-Go",
			"unitType": "MB",
		},
	})
	successTime = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "db_backup_last_success_timestamp_seconds",
		Help: "The timestamp of the last successful completion of a DB backup.",
		ConstLabels: prometheus.Labels{
			"critical": fmt.Sprintf("%f", rand.Float64()*10),
			"warning":  fmt.Sprintf("%f", rand.Float64()+0.5),
			"resource": "Prometheus-Go-Push",
			"group":    "Prometheus-Go",
			"unitType": "MB",
		},
	})
	duration = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "db_backup_duration_seconds",
		Help: "The duration of the last DB backup in seconds.",
		ConstLabels: prometheus.Labels{
			"critical": fmt.Sprintf("%f", rand.Float64()*10),
			"warning":  fmt.Sprintf("%f", rand.Float64()+0.5),
			"resource": "Prometheus-Go-Push",
			"group":    "Prometheus-Go",
			"unitType": "MB",
		},
	})
	records = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "db_backup_records_processed",
		Help: "The number of records processed in the last DB backup.",
		ConstLabels: prometheus.Labels{
			"critical": fmt.Sprintf("%f", rand.Float64()*10),
			"warning":  fmt.Sprintf("%f", rand.Float64()+0.5),
			"resource": "Prometheus-Go-Push",
			"group":    "Prometheus-Go",
			"unitType": "MB",
		},
	})
)

func performBackup() (int, error) {
	// Perform the backup and return the number of backed up records and any
	// applicable error.
	// ...
	return rand.Int(), nil
}

func main() {
	registry := prometheus.NewRegistry()
	registry.MustRegister(completionTime, duration, records)
	// Note that successTime is not registered.

	pusher := push.New(prometheusConnectorUrl, "db_backup").Gatherer(registry)
	pusher.Format(expfmt.FmtText)
	pusher.Client(CustomHttpDoer{})

	start := time.Now()
	n, err := performBackup()
	records.Set(float64(n))
	// Note that time.Since only uses a monotonic clock in Go1.9+.
	duration.Set(time.Since(start).Seconds())
	completionTime.SetToCurrentTime()
	if err != nil {
		fmt.Println("DB backup failed:", err)
	} else {
		// Add successTime to pusher only in case of success.
		// We could as well register it with the registry.
		// This example, however, demonstrates that you can
		// mix Gatherers and Collectors when handling a Pusher.
		pusher.Collector(successTime)
		successTime.SetToCurrentTime()
	}
	// Add is used here rather than Push to not delete a previously pushed
	// success timestamp in case of a failure of this backup.
	if err := pusher.Add(); err != nil {
		fmt.Println("Could not push to PushGateway:", err)
	}
}
