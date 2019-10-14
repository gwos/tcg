package nats

import (
	"github.com/gwos/tng/controller"
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

func StartServer() (*stand.StanServer, error) {
	opts := stand.GetDefaultOptions()
	opts.ID = ClusterID
	opts.StoreType = stores.TypeFile
	opts.FilestoreDir = FilestoreDir

	server, err := stand.RunServerWithOpts(opts, nil)
	if err != nil {
		log.Fatal(err)
	}

	return server, nil
}

func StartDispatcher(dispatcherMap *DispatcherMap) error {
	connection, err := stan.Connect(
		ClusterID,
		DispatcherClientID,
		stan.NatsURL(stan.DefaultNatsURL),
	)
	if err != nil {
		return err
	}

	for subject, dispatcherFn := range *dispatcherMap {
		_, err = connection.QueueSubscribe(
			subject,
			QueueGroup,
			func(msg *stan.Msg) {
				err = dispatcherFn(msg.Data)
				if err == nil {
					_ = msg.Ack()
					controller.AgentStatistics.BytesSent += len(msg.Data)
					controller.AgentStatistics.MessagesSent++
					log.Println("Delivered\nMessage:", msg)
				} else {
					controller.AgentStatistics.LastError = err.Error()
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

func Stop(server *stand.StanServer) {
	server.Shutdown()
}
