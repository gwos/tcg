package nats

import (
	stan "github.com/nats-io/go-nats-streaming"
	"github.com/nats-io/nats-streaming-server/server"
)

var Connection stan.Conn
var Server *server.StanServer