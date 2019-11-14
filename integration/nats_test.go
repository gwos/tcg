package integration

import (
	"github.com/gwos/tng/nats"
	"github.com/gwos/tng/services"
	stan "github.com/nats-io/go-nats-streaming"
	"log"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"
)

const (
	TestNatsClientID                   = "test-nats-client"
	DurableID                          = "tng-store-durable-test"
	QueueGroup                         = "tng-query-store-group"
	TestSendResourceWithMetricsSubject = "send-resource-with-metrics-test"
	TestMessagesCount                  = 3
	GWValidHost                        = "localhost:80"
	GWInvalidHost                      = "localhost:23"
)

var deliveredCount = 0
var droppedCount = 0

// Test for ensuring that all data is stored in NATS and later resent
// if Groundwork Foundation is unavailable
func TestNatsQueue_1(t *testing.T) {
	err := configNats()
	if err != nil {
		t.Error(err)
		return
	}
	log.Println("Config have invalid path to Groundwork Foundation, messages will be stored in a datastore:")
	services.GetTransitService().Config.GWConfig.Host = GWInvalidHost

	connection, subscription, err := connectAndSubscribe()
	if err != nil {
		t.Error(err)
	}
	defer cleanNats(connection, subscription, t)

	for i := 0; i < TestMessagesCount; i++ {
		err := nats.Publish(TestSendResourceWithMetricsSubject, []byte(testMessage))
		if err != nil {
			t.Error(err)
			return
		}
		time.Sleep(1 * time.Second)
	}

	if deliveredCount != 0 {
		t.Errorf("Messages shouldn't be delivered, because Groundwork entrypoint is invalid. deliveredCount = %d, want = %d",
			deliveredCount, 0)
		return
	}

	services.GetTransitService().Config.GWConfig.Host = GWValidHost
	log.Println("Invalid path was changed to valid one")

	time.Sleep(TestMessagesCount * 2 * time.Second)

	if deliveredCount == 0 {
		t.Errorf("Messages should be delivered, because Groundwork entrypoint is valid. deliveredCount = %d, want = %s",
			deliveredCount, "'>0'")
	}
}

// Test for ensuring that all data is stored in NATS and later resent
// after NATS streaming server restarting
func TestNatsQueue_2(t *testing.T) {
	err := configNats()
	if err != nil {
		t.Error(err)
		return
	}

	log.Println("Config have invalid path to Groundwork Foundation, messages will be stored in a datastore:")
	services.GetTransitService().Config.GWConfig.Host = GWInvalidHost

	connection, subscription, err := connectAndSubscribe()
	if err != nil {
		t.Error(err)
	}
	defer cleanNats(connection, subscription, t)

	for i := 0; i < TestMessagesCount; i++ {
		err := nats.Publish(TestSendResourceWithMetricsSubject, []byte(testMessage))
		if err != nil {
			t.Error(err)
		}
		time.Sleep(1 * time.Second)
	}

	if deliveredCount != 0 {
		t.Errorf("Messages shouldn't be delivered, because Groundwork entrypoint is invalid. deliveredCount = %d, want = %d",
			deliveredCount, 0)
		return
	}

	log.Println("Stopping NATS server ...")
	err = services.GetTransitService().StopNats()
	if err != nil {
		t.Error(err)
		return
	}
	log.Println("NATS Server was stopped successfully")

	services.GetTransitService().Config.GWConfig.Host = GWValidHost
	log.Println("Invalid path was changed to valid one")

	log.Println("Starting NATS server ...")
	err = services.GetTransitService().StartNats()
	if err != nil {
		t.Error(err)
		return
	}

	log.Println("NATS Server was started successfully")
	time.Sleep(TestMessagesCount * 2 * time.Second)

	if deliveredCount == 0 {
		t.Errorf("Messages should be delivered, because Groundwork entrypoint is valid. deliveredCount = %d, want = %s",
			deliveredCount, "'>0'")
	}
}

func connectAndSubscribe() (stan.Conn, stan.Subscription, error) {
	connection, err := nats.Connect(TestNatsClientID)
	if err != nil {
		return nil, nil, err
	}
	subscription, err := connection.QueueSubscribe(
		TestSendResourceWithMetricsSubject,
		QueueGroup,
		func(msg *stan.Msg) {
			_, err := services.GetTransitService().SendResourcesWithMetrics(msg.Data)
			if err == nil {
				_ = msg.Ack()
				deliveredCount++
				log.Println("Delivered")
			} else {
				droppedCount++
				log.Println("Not delivered")
			}
		},
		stan.SetManualAckMode(),
		stan.AckWait(2*TestMessagesCount*time.Second),
		stan.DurableName(DurableID),
		stan.StartWithLastReceived(),
	)
	if err != nil {
		return nil, nil, err
	}

	return connection, subscription, nil
}

func configNats() error {
	err := os.Setenv(ConfigEnv, path.Join("..", ConfigName))
	if err != nil {
		return err
	}

	service := services.GetTransitService()

	err = service.StartNats()
	if err != nil {
		return err
	}

	err = service.StartTransport()
	if err != nil {
		return err
	}

	err = service.Connect()
	if err != nil {
		return err
	}

	return nil
}

func cleanNats(connection stan.Conn, subscription stan.Subscription, t *testing.T) {
	err := subscription.Unsubscribe()
	if err != nil {
		t.Error(err)
	}

	err = connection.Close()
	if err != nil {
		t.Error(err)
	}

	err = services.GetTransitService().StopNats()
	if err != nil {
		t.Error(err)
	}

	cmd := exec.Command("rm", "-rf", "src")
	_, err = cmd.Output()
	if err != nil {
		t.Error(err)
	}
	deliveredCount = 0
	droppedCount = 0
}

var testMessage = `{"context": {"appType":"VEMA","agentId":"3939333393342","traceToken":"token-99e93",
	"timeStamp":"2019-10-21T21:00:00.000+0000"},"resources":[{"resource":{"properties":{},
	"name":"GW8_TNG_TEST_HOST_1","type":"HOST","status":"HOST_UP",
	"lastCheckTime":"2019-10-21T21:00:00.000+0000"}},{"resource":{"properties":{},
	"name":"GW8_TNG_TEST_SERVICE_0","type":"SERVICE","owner":"GW8_TNG_TEST_HOST_1",
	"status":"SERVICE_OK","lastCheckTime":"2019-10-21T21:00:00.000+0000"},
	"metrics":[{"tags":{},"metricName":"GW8_TNG_TEST_SERVICE","sampleType":"Warning",
	"interval":{"startTime":"2019-10-20T21:00:00.000+0000","endTime":"2019-10-22T21:00:00.000+0000"},
	"value":{"valueType":"IntegerType","integerValue":1}}]}]}`
