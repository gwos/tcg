package nats

import "C"
import (
	"github.com/gwos/tng/transit"
	stan "github.com/nats-io/go-nats-streaming"
	stand "github.com/nats-io/nats-streaming-server/server"
	"github.com/nats-io/nats-streaming-server/stores"
	"log"
	"time"
)

const (
	ClusterID   = "gw8-cluster"
	ClientID    = "gw8-client"
	QueueGroup  = "gw8-query-store-group"
	PublisherID = "tng-publisher"
	DurableID   = "store-durable"
)

func StartServer() (*stand.StanServer, error) {
	opts := stand.GetDefaultOptions()
	opts.ID = ClusterID
	opts.StoreType = stores.TypeFile
	opts.FilestoreDir = "src/main/resources/datastore"

	server, err := stand.RunServerWithOpts(opts, nil)
	if err != nil {
		log.Fatal(err)
	}

	return server, nil
}

func StartSubscriber(transitConfig *transit.Transit) (*stan.Conn, *stan.Subscription, error) {
	connection, err := stan.Connect(
		ClusterID,
		ClientID,
		stan.NatsURL(stan.DefaultNatsURL),
	)
	if err != nil {
		return nil, nil, err
	}

	subscription, err := connection.QueueSubscribe("resource-with-metrics", QueueGroup, func(msg *stan.Msg) {
		_, err := transitConfig.SendResourcesWithMetrics(msg.Data)
		if err == nil {
			_ = msg.Ack()
			log.Println("Delivered")
		} else {
			log.Println("Not delivered")
		}
	}, stan.SetManualAckMode(),
		stan.AckWait(15*time.Second),
		stan.DurableName(DurableID),
		stan.StartWithLastReceived(),
	)
	if err != nil {
		return nil, nil, err
	}

	return &connection, &subscription, nil
}

func Publish(msg string) error {
	connection, err := stan.Connect(
		ClusterID,
		PublisherID,
		stan.NatsURL(stan.DefaultNatsURL),
	)
	if err != nil {
		return err
	}

	err = connection.Publish("resource-with-metrics", []byte(msg))
	if err != nil {
		return err
	}

	err = connection.Close()
	if err != nil {
		return err
	}

	return nil
}

func Stop(server *stand.StanServer) {
	server.Shutdown()
}
