package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"math/rand"
	"net/http"
	"time"
)

const (
	HostName = "FinanceServicesGo"
	HostGroupName = "PrometheusDemo"
)

var (
	services = []string{"analytics", "distribution", "sales"}
	nodes = []string{"node1", "node2"}

	//dynamicLabels = []string{"node", "service", "code"}
	dynamicLabels = []string{"service"}
	requestsLabels = prometheus.Labels{
		"resource": HostName,
		"group": HostGroupName,
		"warning": "70",
		"critical": "90",
	}
	bytesLabels = prometheus.Labels{
		"resource": HostName,
		"group": HostGroupName,
		"warning": "40000",
		"critical": "45000",
	}
	responseLabels = prometheus.Labels{
		"resource": HostName,
		"group": HostGroupName,
		"warning": "2.0",
		"critical": "2.5",
	}

	requestsPerMinute = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "requests_per_minute",
			Help: "Finance Services http requests per minute.",
			ConstLabels: requestsLabels,
		},
		dynamicLabels,
	)

	bytesPerMinute = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "bytes_per_minute",
			Help: "Finance Services bytes transferred over http per minute",
			ConstLabels: bytesLabels,
		},
		dynamicLabels,
	)

	responseTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "response_time",
			Help: "Finance Services http response time average over 1 minutew",
			ConstLabels: responseLabels,
		},
		dynamicLabels,
	)

	randomizer = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func main() {
	registry := prometheus.NewRegistry()
	registry.Register(requestsPerMinute)
	registry.Register(bytesPerMinute)
	registry.Register(responseTime)

	go metricsGenerator()

	gwHandler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	http.Handle("/metrics", gwHandler)
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":2222", nil))
}

func metricsGenerator() {
	// simulate request traffic
	for {
		for _, service := range services {
			//for _, node := range nodes {
			//	code := "200"
			//	if rand.Intn(20) > 18 {
			//		code = "500"
			//	}
			requestsPerMinute.With(
				prometheus.Labels{
					"service": service,
					//"node":    node,
					//"code":   code,
				}).Set(float64(randomizer.Intn(100)))
			bytesPerMinute.With(
				prometheus.Labels{
					"service": service,
					//"node":    node,
					//"code":    code,
				}).Set(float64(randomizer.Intn(50000)))
			responseTime.With(
				prometheus.Labels{
					"service": service,
				}).Set(float64(randomizer.Intn(30)) / 10)
			//			}
		}
		time.Sleep(time.Second * 30)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Groundwork Prometheus Metrics example. Hit the /metrics end point to see Prometheus Exposition metrics... "))
	w.WriteHeader(200)
}

