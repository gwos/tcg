package nats

import (
	stan "github.com/nats-io/go-nats-streaming"
	stand "github.com/nats-io/nats-streaming-server/server"
	"github.com/nats-io/nats-streaming-server/stores"
	"log"
	"time"
)

// Define NATS IDs
const (
	ClusterID            = "tng-cluster"
	DispatcherClientID   = "tng-dispatcher"
	DispatcherDurableID  = "tng-store-dispatcher"
	DispatcherQueueGroup = "tng-queue-dispatcher"
	PublisherID          = "tng-publisher"
)

var (
	cfg            Config
	dispatcherConn stan.Conn
	stanServer     *stand.StanServer
)

// Config defines NATS configurable options
type Config struct {
	DispatcherAckWait time.Duration
	FilestoreDir      string
	StoreType         string
	NatsURL           string
}

// DispatcherFn defines message processor
type DispatcherFn func([]byte) error

// DispatcherMap maps subject-processor
type DispatcherMap map[string]DispatcherFn

// Connect returns connection
func Connect(clientID string) (stan.Conn, error) {
	return stan.Connect(
		ClusterID,
		clientID,
		stan.NatsURL(cfg.NatsURL),
	)
}

// StartServer runs NATS
func StartServer(config Config) error {
	cfg = config
	opts := stand.GetDefaultOptions()
	opts.ID = ClusterID
	opts.NATSServerURL = cfg.NatsURL
	opts.FilestoreDir = cfg.FilestoreDir
	switch cfg.StoreType {
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
	if dispatcherConn == nil {
		dispatcherConn, err = stan.Connect(
			ClusterID,
			DispatcherClientID,
			stan.NatsURL(cfg.NatsURL),
		)
	}
	if err != nil {
		return err
	}

	for subject, fn := range *dispatcherMap {
		dispatcherFn := fn /* prevent loop override */
		_, err = dispatcherConn.QueueSubscribe(
			subject,
			DispatcherQueueGroup,
			func(msg *stan.Msg) {
				if err := dispatcherFn(msg.Data); err != nil {
					log.Println("Not delivered\nError: ", err.Error(), "\nMessage: ", msg)
				} else {
					_ = msg.Ack()
					log.Println("Delivered\nMessage:", msg)
				}
			},
			stan.SetManualAckMode(),
			stan.AckWait(cfg.DispatcherAckWait),
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
	err := dispatcherConn.Close()
	dispatcherConn = nil
	return err
}

// Publish adds message in queue
func Publish(subject string, msg []byte) error {
	connection, err := stan.Connect(
		ClusterID,
		PublisherID,
		stan.NatsURL(cfg.NatsURL),
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
