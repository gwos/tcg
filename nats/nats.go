package nats

import (
	"fmt"
	"github.com/gwos/tng/log"
	stan "github.com/nats-io/go-nats-streaming"
	stand "github.com/nats-io/nats-streaming-server/server"
	"github.com/nats-io/nats-streaming-server/stores"
	"strconv"
	"strings"
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
	natsURL        string
)

// Config defines NATS configurable options
type Config struct {
	DispatcherAckWait time.Duration
	FilestoreDir      string
	StoreType         string
	NatsHost          string
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
		stan.NatsURL(natsURL),
	)
}

// StartServer runs NATS
func StartServer(config Config) error {
	var err error
	cfg = config
	natsOpts := stand.DefaultNatsServerOptions

	addrParts := strings.Split(cfg.NatsHost, ":")
	if len(addrParts) == 2 {
		if addrParts[0] != "" {
			natsOpts.Host = addrParts[0]
		}
		if port, err := strconv.Atoi(addrParts[1]); err == nil {
			natsOpts.Port = port
		}
	}
	natsURL = fmt.Sprintf("nats://%s:%d", natsOpts.Host, natsOpts.Port)

	stanOpts := stand.GetDefaultOptions()
	stanOpts.ID = ClusterID
	stanOpts.FilestoreDir = cfg.FilestoreDir
	switch cfg.StoreType {
	case "MEMORY":
		stanOpts.StoreType = stores.TypeMemory
	case "FILE":
		stanOpts.StoreType = stores.TypeFile
	default:
		stanOpts.StoreType = stores.TypeFile
	}

	if stanServer == nil || stanServer.State() == stand.Shutdown {
		stanServer, err = stand.RunServerWithOpts(stanOpts, &natsOpts)
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
			stan.NatsURL(natsURL),
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
						log.Info("Not delivered")
						log.Debug("Error: ", err.Error(), "\nMessage: ", msg)
				} else {
					_ = msg.Ack()
						log.Info("Delivered")
						log.Debug("Message:", msg)
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
	if dispatcherConn != nil {
		err := dispatcherConn.Close()
		dispatcherConn = nil
		return err
	}
	return nil
}

// Publish adds message in queue
func Publish(subject string, msg []byte) error {
	connection, err := stan.Connect(
		ClusterID,
		PublisherID,
		stan.NatsURL(natsURL),
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
