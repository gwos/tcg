package main

// #define ERROR_LEN 250 /* for strncpy error message */
// #include <string.h> /* for strncpy error message */
import "C"
import (
	"encoding/json"
	"github.com/gwos/tng/transit"
	stan "github.com/nats-io/go-nats-streaming"
	stand "github.com/nats-io/nats-streaming-server/server"
	"github.com/nats-io/nats-streaming-server/stores"
	"log"
	"time"
)

var transitConfig transit.Transit

const (
	clusterID  = "test-cluster"
	clientID   = "order-query-store1"
	channel    = "order-notification"
	durableID  = "store-durable"
	queueGroup = "order-query-store-group"
)

var count = 0

func main() {
}

func init() {
	initNats()

	initSubscriber()
}

func initNats() {
	opts := stand.GetDefaultOptions()
	opts.ID = "test-cluster"
	opts.StoreType = stores.TypeFile
	opts.FilestoreDir = "datastore"

	_, err := stand.RunServerWithOpts(opts, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func initSubscriber() {
	sc, err := stan.Connect(
		clusterID,
		clientID,
		stan.NatsURL(stan.DefaultNatsURL),
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = sc.QueueSubscribe("foo", queueGroup, func(msg *stan.Msg) {
		log.Println("From subscriber", msg)

		count++

		if count < 3 {
			_ = msg.Ack()
		}
	}, stan.SetManualAckMode(),
		stan.AckWait(15*time.Second),
		stan.DurableName(durableID),
		stan.StartWithLastReceived(),
	)
}

//export TestNats
func TestNats(msg *C.char) {
	sc, err := stan.Connect(
		clusterID,
		"publisherId",
		stan.NatsURL(stan.DefaultNatsURL),
	)
	if err != nil {
		log.Fatal("TestNats: ", err)
	}

	err = sc.Publish("foo", []byte(C.GoString(msg)))
	if err != nil {
		log.Fatal(err)
	}

	err = sc.Close()
}

//export TestMonitoredResource
func TestMonitoredResource(str *C.char, errorMsg *C.char) *C.char {
	resource := transit.MonitoredResource{}
	if err := json.Unmarshal([]byte(C.GoString(str)), &resource); err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.ERROR_LEN)
		return nil
	}

	// resource.Labels = map[string]string{"key1": "value1", "key02": "value02"}
	resource.Status = transit.SERVICE_PENDING
	buf, _ := json.Marshal(&resource)

	log.Printf("#TestMonitoredResource: %v, %s", resource, buf)

	/* https://github.com/golang/go/wiki/cgo#go-strings-and-c-strings */
	return C.CString(string(buf))
}

//export SendResourcesWithMetrics
func SendResourcesWithMetrics(resourcesWithMetricsJson, errorMsg *C.char) *C.char {
	var resourceWithMetrics transit.TransitSendMetricsRequest

	err := json.Unmarshal([]byte(C.GoString(resourcesWithMetricsJson)), &resourceWithMetrics)
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.ERROR_LEN)
		return nil
	}

	operationResults, err := transitConfig.SendResourcesWithMetrics(&resourceWithMetrics)
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.ERROR_LEN)
		return nil
	}

	operationResultsJson, err := json.Marshal(operationResults)
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.ERROR_LEN)
		return nil
	}
	return C.CString(string(operationResultsJson))
}

//export ListMetrics
func ListMetrics(errorMsg *C.char) *C.char {
	monitorDescriptor, err := transitConfig.ListMetrics()
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.ERROR_LEN)
		return nil
	}

	bytes, err := json.Marshal(monitorDescriptor)
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.ERROR_LEN)
		return nil
	}

	return C.CString(string(bytes))
}

//export SynchronizeInventory
func SynchronizeInventory(inventoryJson, errorMsg *C.char) *C.char {
	var inventory transit.TransitSendInventoryRequest

	err := json.Unmarshal([]byte(C.GoString(inventoryJson)), &inventory)
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.ERROR_LEN)
		return nil
	}

	operationResults, err := transitConfig.SynchronizeInventory(&inventory)

	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.ERROR_LEN)
		return nil
	}

	operationResultsJson, err := json.Marshal(operationResults)
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.ERROR_LEN)
		return nil
	}

	return C.CString(string(operationResultsJson))
}

//TODO:
func ListInventory() {
}

//export Connect
func Connect(credentialsJson, errorMsg *C.char) bool {
	transitConfig = transit.Transit{}

	var credentials transit.Credentials

	err := json.Unmarshal([]byte(C.GoString(credentialsJson)), &credentials)
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.ERROR_LEN)
		return false
	}

	err = transitConfig.Connect(credentials)
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.ERROR_LEN)
		return false
	}

	//start nats

	return true
}

//export Disconnect
func Disconnect(errorMsg *C.char) bool {
	err := transitConfig.Disconnect()
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.ERROR_LEN)
		return false
	}
	return true
}
