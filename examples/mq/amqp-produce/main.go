package main

import (
	"context"
	"flag"
	"sync"
	"time"

	"github.com/struqt/x/logging"
	"github.com/struqt/x/mq"
	"github.com/struqt/x/mq/amqp"
)

var (
	queue = flag.String("queue", "demo-000", "Name of a queue")
	url   = flag.String("url", "amqp://user:12345@127.0.0.1:5672", "URL of an AMQP server")
)

var log logging.Logger

func init() {
	flag.Parse()
	logging.LogVerbosity = 127
	logging.LogConsoleThreshold = -128
	log = logging.NewLogger("").WithName("amqp-produce")
}

func main() {
	timeout := 18*time.Second + 100*time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var wg sync.WaitGroup
	wg.Add(2)
	amqp.SetLogger(log)
	producer := amqp.NewProducer(*queue, *url, 5)
	go sendEvery2Sec(ctx, &wg, producer)
	go func() {
		defer wg.Done()
		producer.RunWith(ctx)
	}()
	wg.Wait()
	log.Info("Demo is ending ...")
}

func sendEvery2Sec(ctx context.Context, wg *sync.WaitGroup, producer mq.Producer) {
	defer wg.Done()
	ticker := time.NewTicker(1000 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			go func() {
				message := "Hello World --- " + time.Now().String()
				producer.SendString(message)
				log.Info(message)
			}()
		case <-ctx.Done():
			log.Info("Demo Ticker is stopping --")
			return
		}
	}
}
