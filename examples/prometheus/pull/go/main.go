package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"time"
)

const (
	HostName      = "FinanceServicesGo"
	HostGroupName = "PrometheusDemo"
)

var (
	services = []string{"analytics", "distribution", "sales"}
	// nodes    = []string{"node1", "node2"}

	//dynamicLabels = []string{"node", "service", "code"}
	dynamicLabels  = []string{"service"}
	requestsLabels = prometheus.Labels{
		"resource": HostName,
		"group":    HostGroupName,
		"warning":  "70",
		"critical": "90",
	}
	bytesLabels = prometheus.Labels{
		"resource": HostName,
		"group":    HostGroupName,
		"warning":  "40000",
		"critical": "45000",
	}
	responseLabels = prometheus.Labels{
		"resource": HostName,
		"group":    HostGroupName,
		"warning":  "2.0",
		"critical": "2.5",
	}

	requestsPerMinute = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:        "requests_per_minute",
			Help:        "Finance Services http requests per minute.",
			ConstLabels: requestsLabels,
		},
		dynamicLabels,
	)

	bytesPerMinute = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:        "bytes_per_minute",
			Help:        "Finance Services bytes transferred over http per minute.",
			ConstLabels: bytesLabels,
		},
		dynamicLabels,
	)

	responseTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:        "response_time",
			Help:        "Finance Services http response time average over 1 minute.",
			ConstLabels: responseLabels,
		},
		dynamicLabels,
	)

	analyticsTimer = time.Date(2020, 1,1, 0, 0, 0, 0, time.UTC)
	distributionTimer = time.Date(2020, 1,1, 0, 0, 0, 0, time.UTC)
	salesTimer = time.Date(2020, 1,1, 0, 0, 0, 0, time.UTC)

	randomizer = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func main() {
	registry := prometheus.NewRegistry()
	_ = registry.Register(requestsPerMinute)
	_ = registry.Register(bytesPerMinute)
	_ = registry.Register(responseTime)

	// go metricsGenerator()
	go requestsGenerator()

	gwHandler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	http.Handle("/metrics", gwHandler)
	http.HandleFunc("/", handler)
	http.HandleFunc("/analytics", analyticsHandler)
	http.HandleFunc("/distribution", distributionHandler)
	http.HandleFunc("/sales", salesHandler)
	log.Fatal(http.ListenAndServe(":2222", nil))
}

func analyticsHandler(w http.ResponseWriter, r *http.Request) {
	genericHandler(w, r, "analytics", &analyticsTimer)
}

func distributionHandler(w http.ResponseWriter, r *http.Request) {
	genericHandler(w, r, "distribution", &distributionTimer)
}

func salesHandler(w http.ResponseWriter, r *http.Request) {
	genericHandler(w, r, "sales", &salesTimer)
}

func genericHandler(w http.ResponseWriter, r *http.Request, serviceName string, timer *time.Time) {
	// instrument your code
	start := time.Now()
	labels := prometheus.Labels{"service": serviceName}

	// call your application logic here...
	bytesCount := processRequest()

	//  per minute metrics
	if timer.Add(time.Minute).Before(start) {
		*timer = start
		requestsPerMinute.With(labels).Set(1)
		bytesPerMinute.With(labels).Set(0)
	} else {
		requestsPerMinute.With(labels).Inc()
		bytesPerMinute.With(labels).Add(float64(bytesCount))
	}

	// normally calculate response time metrics
	elapsed := float64(time.Since(start).Milliseconds())
	// fake the response time for demo, normally use elapsed variable
	elapsed = float64(randomizer.Intn(30)) / 10

	responseTime.With(labels).Set(elapsed)
	_, _ = w.Write([]byte("Groundwork Prometheus Metrics example response for " + serviceName + "\n"))
}

// this is where your application would process the request and return a response
func processRequest() int {
	return randomizer.Intn(12500)
}

func requestsGenerator() {
	for ;; {
		resp, _ := http.Get("http://localhost:2222/analytics")
		ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		resp, _ = http.Get("http://localhost:2222/distribution")
		ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		resp, _ = http.Get("http://localhost:2222/sales")
		ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		time.Sleep(time.Second * 15) // generate 4 requests per minute
	}
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
	_, _ = w.Write([]byte("Groundwork Prometheus Metrics example. Hit the /metrics end point to see Prometheus Exposition metrics..."))
	w.WriteHeader(200)
}
