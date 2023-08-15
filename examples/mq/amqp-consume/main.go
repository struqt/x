package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/struqt/x/mq/amqp"
)

var (
	queue = flag.String("queue", "demo-000", "Name of a queue")
	url   = flag.String("url", "amqp://user:12345@127.0.0.1:5672", "URL of an AMQP server")
)

func init() {
	flag.Parse()
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	consumer := amqp.NewConsumer(*queue, *url)
	consumer.RunWith(ctx, func(message []byte) bool {
		log.Printf("Received a message: %s", string(message))
		return true
	})
}
