package nats

import (
	"github.com/gwos/tng/transit"
	stan "github.com/nats-io/go-nats-streaming"
	stand "github.com/nats-io/nats-streaming-server/server"
	"github.com/nats-io/nats-streaming-server/stores"
	"log"
	"time"
)

var (
	ClusterID          = "tng-cluster"
	DispatcherClientID = "tng-dispatcher"
	DurableID          = "tng-store-durable"
	FilestoreDir       = "src/main/resources/datastore"
	PublisherID        = "tng-publisher"
	QueueGroup         = "tng-query-store-group"
)

/* DispatcherFn */
type DispatcherFn func([]byte) error

/* DispatcherMap */
type DispatcherMap map[string]DispatcherFn

func StartServer() error {
	opts := stand.GetDefaultOptions()
	opts.ID = ClusterID
	opts.StoreType = stores.TypeFile
	opts.FilestoreDir = FilestoreDir

	var err error
	Server, err = stand.RunServerWithOpts(opts, nil)
	if err != nil {
		return err
	}

	return nil
}

func StopServer() {
	Server.Shutdown()
}

func StartDispatcher(dispatcherMap *DispatcherMap) error {
	var err error
	Connection, err = stan.Connect(
		ClusterID,
		DispatcherClientID,
		stan.NatsURL(stan.DefaultNatsURL),
	)
	if err != nil {
		return err
	}

	for subject, dispatcherFn := range *dispatcherMap {
		_, err = Connection.QueueSubscribe(
			subject,
			QueueGroup,
			func(msg *stan.Msg) {
				err = dispatcherFn(msg.Data)
				if err == nil {
					_ = msg.Ack()
					transit.AgentStatistics.BytesSent += len(msg.Data)
					transit.AgentStatistics.MessagesSent++
					log.Println("Delivered\nMessage:", msg)
				} else {
					transit.AgentStatistics.LastError = err.Error()
					log.Println("Not delivered\nError: ", err.Error(), "\nMessage: ", msg)
				}
			},
			stan.SetManualAckMode(),
			stan.AckWait(15*time.Second),
			stan.DurableName(DurableID),
			stan.StartWithLastReceived(),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func StopDispatcher() error {
	return Connection.Close()
}

func Publish(subject string, msg []byte) error {
	connection, err := stan.Connect(
		ClusterID,
		PublisherID,
		stan.NatsURL(stan.DefaultNatsURL),
	)
	if err != nil {
		return err
	}

	err = connection.Publish(subject, msg)
	if err != nil {
		return err
	}

	err = connection.Close()
	if err != nil {
		return err
	}

	return nil
}
