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
	ClusterID            = "tng-cluster"
	DispatcherAckWait    = 15 * time.Second
	DispatcherClientID   = "tng-dispatcher"
	DispatcherDurableID  = "tng-store-dispatcher"
	DispatcherQueueGroup = "tng-queue-dispatcher"
	PublisherID          = "tng-publisher"
)

var (
	dispatcherConn stan.Conn
	stanServer     *stand.StanServer
)

// DispatcherFn defines message processor
type DispatcherFn func([]byte) error

// DispatcherMap maps subject-processor
type DispatcherMap map[string]DispatcherFn

// Connect returns connection
func Connect(clientID string) (stan.Conn, error) {
	return stan.Connect(
		ClusterID,
		clientID,
		stan.NatsURL(stan.DefaultNatsURL),
	)
}

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

	var err error
	if stanServer == nil || stanServer.State() == stand.Shutdown {
		stanServer, err = stand.RunServerWithOpts(opts, nil)
	}
	return err
}

// StopServer shutdowns NATS
func StopServer() {
	stanServer.Shutdown()
}

// StartDispatcher subscribes processors by subject
func StartDispatcher(dispatcherMap *DispatcherMap) error {
	var err error
	if dispatcherConn == nil || dispatcherConn.NatsConn().Status() == nats.CLOSED {
		dispatcherConn, err = stan.Connect(
			ClusterID,
			DispatcherClientID,
			stan.NatsURL(stan.DefaultNatsURL),
		)
	}

	for subject, fn := range *dispatcherMap {
		dispatcherFn := fn /* prevent loop override */
		_, err = dispatcherConn.QueueSubscribe(
			subject,
			DispatcherQueueGroup,
			func(msg *stan.Msg) {
				if err := dispatcherFn(msg.Data); err != nil {
					_ = msg.Ack()
					log.Println("Delivered\nMessage:", msg)
				} else {
					log.Println("Not delivered\nError: ", err.Error(), "\nMessage: ", msg)
				}
			},
			stan.SetManualAckMode(),
			stan.AckWait(DispatcherAckWait),
			stan.DurableName(DispatcherDurableID),
			stan.StartWithLastReceived(),
		)
		if err != nil {
			break
		}
	}
	return err
}

// StopDispatcher ends dispatching
func StopDispatcher() error {
	return dispatcherConn.Close()
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
	return err
}
