package main

import (
	"context"
	"flag"
	"time"

	"github.com/struqt/logging"
	"github.com/struqt/x/mq/amqp"
)

var (
	queue = flag.String("queue", "demo-000", "Name of a queue")
	url   = flag.String("url", "amqp://user:12345@127.0.0.1:5672", "URL of an AMQP server")
)

func init() {
	flag.Parse()
	logging.LogVerbosity = 127
	logging.LogConsoleThreshold = -128
}

func main() {
	log := logging.NewLogger("").WithName("amqp-produce")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	amqp.SetLogger(log)
	consumer := amqp.NewConsumer(*queue, *url)
	consumer.RunWith(ctx, func(message []byte) bool {
		log.Info(string(message))
		return true
	})
}
