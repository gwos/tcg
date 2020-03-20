package nats

import (
	"fmt"
	"github.com/gwos/tng/cache"
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
	ClusterID    = "tng-cluster"
	DispatcherID = "tng-dispatcher"
	PublisherID  = "tng-publisher"
)

var (
	cfg            Config
	connDispatcher stan.Conn
	connPublisher  stan.Conn
	stanServer     *stand.StanServer
	natsURL        string
)

// Config defines NATS configurable options
type Config struct {
	DispatcherAckWait     time.Duration
	DispatcherMaxInflight int
	MaxPubAcksInflight    int
	FilestoreDir          string
	StoreType             string
	NatsHost              string
}

// DispatcherFn defines message processor
type DispatcherFn func([]byte) error

// DispatcherOption defines subscription
type DispatcherOption struct {
	DurableID string
	Subject   string
	Handler   DispatcherFn
}

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
	if connPublisher != nil {
		connPublisher.Close()
		connPublisher = nil
	}
	stanServer.Shutdown()
}

// StartDispatcher subscribes processors by subject
func StartDispatcher(options []DispatcherOption) error {
	err := StopDispatcher()
	if err != nil {
		return err
	}
	if connDispatcher == nil {
		connDispatcher, err = stan.Connect(
			ClusterID,
			DispatcherID,
			stan.NatsURL(natsURL),
		)
	}
	if err != nil {
		return err
	}

	for _, o := range options {
		dispatcherFn := o.Handler /* prevent loop override */
		durableID := o.DurableID
		_, err = connDispatcher.Subscribe(
			o.Subject,
			func(msg *stan.Msg) {
				if _, isDone := cache.DispatcherDoneCache.Get(fmt.Sprintf("%s-%d", msg.Subject, msg.Sequence)); isDone {
					return
				}

				if err := dispatcherFn(msg.Data); err != nil {
					log.With(log.Fields{
						"nats.durableID": durableID,
						"nats.subject":   msg.Subject,
					}).WithDebug(log.Fields{
						"error":   err,
						"message": msg,
					}).Log(log.InfoLevel, "NATS: Not delivered")
				} else {
					_ = msg.Ack()
					_ = cache.DispatcherDoneCache.Add(fmt.Sprintf("%s-%d", msg.Subject, msg.Sequence), 0, 10*time.Minute)
					log.With(log.Fields{
						"nats.durableID": durableID,
						"nats.subject":   msg.Subject,
					}).WithDebug(log.Fields{
						"message": msg,
					}).Log(log.InfoLevel, "NATS: Delivered")
				}
			},
			stan.SetManualAckMode(),
			stan.AckWait(cfg.DispatcherAckWait),
			stan.MaxInflight(cfg.DispatcherMaxInflight),
			stan.DurableName(fmt.Sprintf("%s-%s", DispatcherID, o.DurableID)),
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
	if connDispatcher != nil {
		err := connDispatcher.Close()
		connDispatcher = nil
		return err
	}
	return nil
}

// Publish adds message in queue
func Publish(subject string, msg []byte) error {
	var err error
	if connPublisher == nil {
		connPublisher, err = stan.Connect(
			ClusterID,
			PublisherID,
			stan.NatsURL(natsURL),
			stan.MaxPubAcksInflight(cfg.MaxPubAcksInflight),
		)
	}
	if err != nil {
		return err
	}

	err = connPublisher.Publish(subject, msg)
	return err
}
