package nats

import (
	"github.com/nats-io/go-nats"
	stan "github.com/nats-io/go-nats-streaming"
	stand "github.com/nats-io/nats-streaming-server/server"
	"github.com/nats-io/nats-streaming-server/stores"
	"log"
	"time"
)

// Define NATS IDs
const (
	ClusterID          = "tng-cluster"
	DispatcherClientID = "tng-dispatcher"
	DurableID          = "tng-store-durable"
	PublisherID        = "tng-publisher"
	QueueGroup         = "tng-query-store-group"
)

// DispatcherFn defines message processor
type DispatcherFn func([]byte) error

// DispatcherMap maps subject-processor
type DispatcherMap map[string]DispatcherFn

// StartServer runs NATS
func StartServer(storeType, filestoreDir string) error {
	opts := stand.GetDefaultOptions()
	opts.ID = ClusterID
	opts.FilestoreDir = filestoreDir
	switch storeType {
	case "MEMORY":
		opts.StoreType = stores.TypeMemory
	case "FILE":
		opts.StoreType = stores.TypeFile
	default:
		opts.StoreType = stores.TypeFile
	}

	if Server == nil || Server.State() == stand.Shutdown {
		var err error
		Server, err = stand.RunServerWithOpts(opts, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

// StopServer shutdowns NATS
func StopServer() {
	Server.Shutdown()
}

// StartDispatcher subscribes processors by subject
func StartDispatcher(dispatcherMap *DispatcherMap) error {
	if Connection == nil || Connection.NatsConn().Status() == nats.CLOSED {
		var err error
		Connection, err = stan.Connect(
			ClusterID,
			DispatcherClientID,
			stan.NatsURL(stan.DefaultNatsURL),
		)
		if err != nil {
			return err
		}

		for subject, fn := range *dispatcherMap {
			dispatcherFn := fn /* prevent loop override */
			_, err = Connection.QueueSubscribe(
				subject,
				QueueGroup,
				func(msg *stan.Msg) {
					err = dispatcherFn(msg.Data)
					if err == nil {
						_ = msg.Ack()
						log.Println("Delivered\nMessage:", msg)
					} else {
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
	}

	return nil
}

// StopDispatcher ends dispatching
func StopDispatcher() error {
	return Connection.Close()
}

// Publish adds message in queue
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
