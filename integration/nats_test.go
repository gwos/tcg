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
	GroundWorkMonitoringValidPath      = "/api/monitoring"
	GroundWorkMonitoringInvalidPath    = "/api/invalidPath"
)

var deliveredCount = 0
var droppedCount = 0

func TestNATS(t *testing.T) {
	err := configNATS()
	if err != nil {
		t.Error(err)
		return
	}

	con, err := nats.Connect(TestNatsClientID)
	if err != nil {
		t.Error(err)
		return
	}

	log.Println("Invalid path:")
	services.GetTransitService().Config.GroundworkActions.SendResourceWithMetrics.Entrypoint = GroundWorkMonitoringInvalidPath

	subscription, err := con.QueueSubscribe(
		TestSendResourceWithMetricsSubject,
		QueueGroup,
		func(msg *stan.Msg) {
			_, err = services.GetTransitService().Transit.SendResourcesWithMetrics(msg.Data)
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
		t.Error(err)
		return
	}
	defer subscription.Close()
	defer func() {
		cmd := exec.Command("rm", "-rf", "src")
		_, err = cmd.Output()
		if err != nil {
			log.Println(err)
		}
	}()

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

	log.Println("Valid path:")
	services.GetTransitService().Config.GroundworkActions.SendResourceWithMetrics.Entrypoint = GroundWorkMonitoringValidPath

	time.Sleep(TestMessagesCount * 3 * time.Second)

	if deliveredCount == 0 {
		t.Errorf("Messages should be delivered, because Groundwork entrypoint is valid. deliveredCount = %d, want = %s",
			deliveredCount, "'>0'")
	}
}

func configNATS() error {
	err := os.Setenv(ConfigEnv, path.Join("..", ConfigName))
	if err != nil {
		return err
	}

	service := services.GetTransitService()

	err = service.StartNATS()
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

var testMessage = `{"context": {"appType":"VEMA","agentId":"3939333393342","traceToken":"token-99e93",
	"timeStamp":"2019-10-21T21:00:00.000+0000"},"resources":[{"resource":{"properties":{},
	"name":"GW8_TNG_TEST_HOST_1","type":"HOST","status":"HOST_UP",
	"lastCheckTime":"2019-10-21T21:00:00.000+0000"}},{"resource":{"properties":{},
	"name":"GW8_TNG_TEST_SERVICE_0","type":"SERVICE","owner":"GW8_TNG_TEST_HOST_1",
	"status":"SERVICE_OK","lastCheckTime":"2019-10-21T21:00:00.000+0000"},
	"metrics":[{"tags":{},"metricName":"GW8_TNG_TEST_SERVICE","sampleType":"Warning",
	"interval":{"startTime":"2019-10-20T21:00:00.000+0000","endTime":"2019-10-22T21:00:00.000+0000"},
	"value":{"valueType":"IntegerType","integerValue":1}}]}]}`
