package nats

import (
	"bufio"
	"fmt"
	stan "github.com/nats-io/go-nats-streaming"
	stand "github.com/nats-io/nats-streaming-server/server"
	"github.com/nats-io/nats-streaming-server/stores"
	"math/rand"
	"os"
	"strconv"
	"time"
)

const (
	StandServer    = "tng-transit"
	PrimaryChannel = "primary-metrics"
	PrimaryDurable = "primary-metrics"
	Listener       = "tng-listener"
	Publisher      = "tng-publisher"
)

func Start() (*stand.StanServer, error) {
	//opts := stand.GetDefaultOptions()
	//opts.StoreType = stores.TypeFile
	//opts.FilestoreDir = "nats-datastore"
	//opts.ID = StandServer
	//server, err := stand.RunServerWithOpts(opts, nil)
	//if err != nil {
	//	return nil, err
	//}
	//var sub stan.Subscription
	//
	//listener, _ := stan.Connect(StandServer, Listener)
	//
	//sub, _ = listener.Subscribe(PrimaryChannel,
	//	func(msg *stan.Msg) {
	//		t := time.Unix(0, msg.Timestamp).Format(time.RFC3339Nano)
	//		debug := fmt.Sprintf("Received: %s, Timestamp: %s, Subject: %s", string(msg.Data), t, msg.Subject)
	//		fmt.Println("received: ", debug)
	//	},
	//	stan.DurableName(PrimaryDurable))
	//
	//publisher, err := stan.Connect(StandServer, Publisher)

	return nil, nil
}

func Stop(server *stand.StanServer) {
	server.Shutdown()
}

func nats() {
	runPublisher := false
	runSubscriber := true

	// (1) start

	opts := stand.GetDefaultOptions()
	opts.StoreType = stores.TypeFile
	opts.FilestoreDir = "nats-datastore"
	opts.ID = StandServer
	server, _ := stand.RunServerWithOpts(opts, nil)
	fmt.Println("NATS Streaming Server started ..." + server.ClusterID())
	var sub stan.Subscription
	if runSubscriber {

		// start (2)

		listener, _ := stan.Connect(StandServer, Listener)
		sub, _ = listener.Subscribe(PrimaryChannel,
			func(msg *stan.Msg) {
				t := time.Unix(0, msg.Timestamp).Format(time.RFC3339Nano)
				debug := fmt.Sprintf("Received: %s, Timestamp: %s, Subject: %s", string(msg.Data), t, msg.Subject)
				fmt.Println("received: ", debug)
			},
			stan.DurableName(PrimaryDurable))

		// stop(2)
		// (3) sub.Close()
		// server.Shutdown()
	}

	if runPublisher {

		// (4) start

		publisher, err := stan.Connect(StandServer, Publisher)
		if err == nil {
			_ = publisher.Publish(PrimaryChannel, []byte("Hello Message 1 : "+strconv.Itoa(rand.Int())))
			_ = publisher.Publish(PrimaryChannel, []byte("Hello Message 2 : "+strconv.Itoa(rand.Int())))
			_ = publisher.Publish(PrimaryChannel, []byte("Hello Message 3 : "+strconv.Itoa(rand.Int())))
			fmt.Printf("published messages...\n")
		}

		// (4) stop
	}
	wait()
	if sub != nil {
		_ = sub.Close()
	}
}

func wait() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Press any key to continue: ")
	_, _ = reader.ReadString('\n')
}
