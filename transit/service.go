package transit

import (
	"encoding/json"
	"errors"
	"github.com/gwos/tng/nats"
	"gopkg.in/yaml.v2"
	"log"
	"net/http"
	"os"
)

const (
	SendResourceWithMetricsSubject = "send-resource-with-metrics"
	SynchronizeInventorySubject    = "synchronize-inventory"
)

func init() {
	configFile, err := os.Open("/home/vladislavsenkevich/Projects/groundwork/_rep/tng/libtransit/config.yml")
	if err != nil {
		log.Fatal(err)
	}

	decoder := yaml.NewDecoder(configFile)
	err = decoder.Decode(&config)
	if err != nil {
		log.Fatal(err)
	}

	err = config.connect()
	if err != nil {
		log.Fatal(err)
	}

	dispatcherMap := nats.DispatcherMap{
		SendResourceWithMetricsSubject: func(b []byte) error {
			_, err := config.sendResourcesWithMetrics(b)
			return err
		},
		SynchronizeInventorySubject: func(b []byte) error {
			_, err := config.synchronizeInventory(b)
			return err
		},
	}

	_, err = nats.StartServer()
	if err != nil {
		log.Fatal(err)
	}

	err = nats.StartDispatcher(&dispatcherMap)
	if err != nil {
		log.Fatal(err)
	}
}

type Service struct {
}

func (transitService Service) SendResourceWithMetrics(resourcesWithMetricsJson []byte) error {
	return nats.Publish(SendResourceWithMetricsSubject, resourcesWithMetricsJson)
}

func (transitService Service) SynchronizeInventory(inventoryJson []byte) error {
	return nats.Publish(SynchronizeInventorySubject, inventoryJson)
}

func (transitService Service) ListMetrics() (*[]MetricDescriptor, error) {
	return config.listMetrics()
}

var config Transit

func (transit Transit) synchronizeInventory(inventory []byte) (*OperationResults, error) {
	headers := map[string]string{
		"Accept":         "application/json",
		"Content-Type":   "application/json",
		"GWOS-API-TOKEN": transit.Config.Token,
		"GWOS-APP-NAME":  "gw8",
	}

	statusCode, byteResponse, err := sendRequest(http.MethodPost, "http://localhost/api/synchronizer", headers, nil, inventory)
	if err != nil {
		return nil, err
	}
	if statusCode == 401 {
		err = transit.connect()
		if err != nil {
			return nil, err
		}
	}
	if statusCode != 200 {
		return nil, errors.New(string(byteResponse))
	}

	var operationResults OperationResults

	err = json.Unmarshal(byteResponse, &operationResults)
	if err != nil {
		return nil, err
	}

	return &operationResults, nil
}

func (transit Transit) sendResourcesWithMetrics(resources []byte) (*OperationResults, error) {
	headers := map[string]string{
		"Accept":         "application/json",
		"Content-Type":   "application/json",
		"GWOS-API-TOKEN": transit.Config.Token,
		"GWOS-APP-NAME":  "gw8",
	}

	statusCode, byteResponse, err := sendRequest(http.MethodPost, "http://localhost/api/monitoring", headers, nil, resources)
	if err != nil {
		return nil, err
	}
	if statusCode == 401 {
		err = transit.connect()
		if err != nil {
			return nil, err
		}
	}
	if statusCode != 200 {
		return nil, errors.New(string(byteResponse))
	}

	var operationResults OperationResults

	err = json.Unmarshal(byteResponse, &operationResults)
	if err != nil {
		return nil, err
	}

	return &operationResults, nil
}

// TODO: implement
func (transit Transit) listMetrics() (*[]MetricDescriptor, error) {
	// setup label descriptor samples
	cores := LabelDescriptor{
		Description: "Number of Cores",
		Key:         "cores",
		ValueType:   StringType,
	}
	sampleTime := LabelDescriptor{
		Description: "Sample Time",
		Key:         "sampleTime",
		ValueType:   IntegerType,
	}
	load1 := MetricDescriptor{
		Type:        "local_load_1",
		Description: "Local Load for 1 minute",
		DisplayName: "LocalLoad1",
		Labels:      []*LabelDescriptor{&cores, &sampleTime},
		MetricKind:  GAUGE,
		ComputeType: Query,
		CustomName:  "load-one-minute",
		Unit:        UnitCounter,
		ValueType:   DoubleType,
		Thresholds: []*ThresholdDescriptor{
			&ThresholdDescriptor{Key: "critical", Value: 200},
			&ThresholdDescriptor{Key: "warning", Value: 100},
		},
	}
	load5 := MetricDescriptor{
		Type:        "local_load_5",
		Description: "Local Load for 5 minute",
		DisplayName: "LocalLoad5",
		Labels:      []*LabelDescriptor{&cores, &sampleTime},
		MetricKind:  GAUGE,
		ComputeType: Query,
		CustomName:  "load-five-minutes",
		Unit:        UnitCounter,
		ValueType:   DoubleType,
		Thresholds: []*ThresholdDescriptor{
			&ThresholdDescriptor{Key: "critical", Value: 205},
			&ThresholdDescriptor{Key: "warning", Value: 105},
		},
	}
	load15 := MetricDescriptor{
		Type:        "local_load_15",
		Description: "Local Load for 15 minute",
		DisplayName: "LocalLoad15",
		Labels:      []*LabelDescriptor{&cores, &sampleTime},
		MetricKind:  GAUGE,
		ComputeType: Query,
		CustomName:  "load-fifteen-minutes",
		Unit:        UnitCounter,
		ValueType:   DoubleType,
		Thresholds: []*ThresholdDescriptor{
			&ThresholdDescriptor{Key: "critical", Value: 215},
			&ThresholdDescriptor{Key: "warning", Value: 115},
		},
	}
	arr := []MetricDescriptor{load1, load5, load15}
	return &arr, nil
}

// create and connect to a Transit instance from a Groundwork connection configuration
func (transit *Transit) connect() error {
	formValues := map[string]string{
		"gwos-app-name": "gw8",
		"user":          transit.Config.Account,
		"password":      transit.Config.Password,
	}

	headers := map[string]string{
		"Accept":       "text/plain",
		"Content-Type": "application/x-www-form-urlencoded",
	}

	statusCode, byteResponse, err := sendRequest(http.MethodPost, "http://localhost/api/auth/login", headers, formValues, nil)
	if err != nil {
		return err
	}
	if statusCode == 200 {
		transit.Config.Token = string(byteResponse)
		return nil
	}

	return errors.New(string(byteResponse))
}

func (transit Transit) disconnect() error {
	formValues := map[string]string{
		"gwos-app-name":  "gw8",
		"gwos-api-token": transit.Config.Token,
	}

	headers := map[string]string{
		"Accept":       "text/plain",
		"Content-Type": "application/x-www-form-urlencoded",
	}

	statusCode, byteResponse, err := sendRequest(http.MethodPost, "http://localhost/api/auth/logout", headers, formValues, nil)
	if err != nil {
		return err
	}

	if statusCode == 200 {
		return nil
	}
	return errors.New(string(byteResponse))
}
