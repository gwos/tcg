package main

import (
	"fmt"
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
	dynamicLabels  = []string{"resource", "service"}
	requestsLabels = prometheus.Labels{
		// "resource": HostName,
		"group":    HostGroupName,
		"warning":  "85",
		"critical": "95",
	}
	bytesLabels = prometheus.Labels{
		// "resource": HostName,
		"group":    HostGroupName,
		"warning":  "45000",
		"critical": "48000",
	}
	responseLabels = prometheus.Labels{
		// "resource": HostName,
		"group":    HostGroupName,
		"warning":  "2.5",
		"critical": "2.8",
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
	instrumentedHandler(w, r, "analytics")
}

func distributionHandler(w http.ResponseWriter, r *http.Request) {
	instrumentedHandler(w, r, "distribution")
}

func salesHandler(w http.ResponseWriter, r *http.Request) {
	instrumentedHandler(w, r, "sales")
}

func instrumentedHandler(w http.ResponseWriter, r *http.Request, serviceName string) {
	// instrument your http handler, start the timer ...
	start := time.Now()
	for ix := 1; ix <= 1000; ix++ {
		hostName := fmt.Sprintf("%s-%d", HostName, ix)
		for iy := 1; iy <= 10; iy++ {
			serviceFullName := fmt.Sprintf("service-%d", ix)
			labels := prometheus.Labels{"service": serviceFullName, "resource": hostName}

			// call your application logic here... this returns simulated random instrumentation numbers
			requestsNumber, bytesNumber, responseTimeNumber := processRequest()

			// instrument requestsPerMinute with random number, this could also be done with a histogram
			requestsPerMinute.With(labels).Set(requestsNumber)
			// instrument bytes per minute with random number, this could also be done with a histogram
			bytesPerMinute.With(labels).Set(bytesNumber)
			// instrument response time with random number, you would normally use the elapsed variable
			responseTime.With(labels).Set(responseTimeNumber)
		}
	}
	// calculate responseTime, you would normally set instrument this on responseTime
	elapsed := float64(time.Since(start).Nanoseconds())
	message := fmt.Sprintf("Groundwork Prometheus Metrics example response for %s in %f ns\n", serviceName, elapsed)
	_, _ = w.Write([]byte(message))
}

// Here is where your application would process the request & return a response. Instead we just generate random numbers
func processRequest() (float64, float64, float64) {
	requestsPerMinute := float64(randomizer.Intn(100))
	bytesPerMinute := float64(randomizer.Intn(50000))
	responseTime := float64(randomizer.Intn(30)) / 10
	return requestsPerMinute, bytesPerMinute, responseTime
}

// simulate the generation of requests
func requestsGenerator() {
	for ; ; {
		resp, _ := http.Get("http://localhost:2222/analytics")
		_, _ = ioutil.ReadAll(resp.Body)
		_ = resp.Body.Close()
		resp, _ = http.Get("http://localhost:2222/distribution")
		_, _ = ioutil.ReadAll(resp.Body)
		_ = resp.Body.Close()
		resp, _ = http.Get("http://localhost:2222/sales")
		_, _ = ioutil.ReadAll(resp.Body)
		_ = resp.Body.Close()
		time.Sleep(time.Second * 30)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("Groundwork Prometheus Metrics example. Hit the /metrics end point to see Prometheus Exposition metrics..."))
	w.WriteHeader(200)
}
