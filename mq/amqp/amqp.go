package amqp

import (
	"context"
	"fmt"
	"sync"
	"time"

	rabbit "github.com/rabbitmq/amqp091-go"
	"github.com/struqt/logging"
	"github.com/struqt/x/mq"
)

var log logging.Logger
var retryDelays = []int{1, 4, 9, 16}

var setupOnce sync.Once

func setup() {
	setupOnce.Do(func() {
		if log.IsZero() {
			log = logging.NewLogger("").WithName("AMQP")
		}
	})
}

func SetLogger(logger logging.Logger) {
	log = logger
}

func Reconnect(name, url string) (*rabbit.Channel, *rabbit.Connection) {
	retryIndex := 0
	for {
		uri, _ := rabbit.ParseURI(url)
		conn, err := rabbit.Dial(url)
		if err == nil {
			ch, err := conn.Channel()
			if err == nil {
				_, err := ch.QueueDeclare(
					name,  // name
					true,  // durable
					false, // delete when unused
					false, // exclusive
					false, // no-wait
					nil,   // arguments
				)
				if err == nil {
					err = ch.Confirm(false)
					if err == nil {
						log.Info("Connected.", "host", uri.Host, "port", uri.Port, "queue", name)
						return ch, conn
					} else {
						log.Error(err, "Error on ch.Confirm(..)")
					}
				} else {
					log.Error(err, "Error on ch.QueueDeclare(..)")
				}
				if err = ch.Close(); err != nil {
					log.Error(err, "Error on ch.Close()")
				}
			} else {
				log.Error(err, "Error on conn.Channel()")
			}
			if err = conn.Close(); err != nil {
				log.Error(err, "Error on conn.Close()")
			}
		} else {
			log.Error(err, "Error on rabbit.Dial(..)")
		}
		log.Info(
			fmt.Sprintf("Failed to connect, retrying in %d seconds ...", retryDelays[retryIndex]),
			"host", uri.Host, "port", uri.Port, "queue", name,
		)
		time.Sleep(time.Duration(retryDelays[retryIndex]) * time.Second)
		retryIndex = (retryIndex + 1) % len(retryDelays)
	}
}

func NewConsumer(queue, url string) mq.Consumer {
	setup()
	return &consumer{
		queue:      queue,
		url:        url,
		connection: nil,
		channel:    nil,
	}
}

func NewProducer(queue, url string, backlog int) mq.Producer {
	setup()
	var channel chan []byte
	if backlog <= 0 {
		channel = make(chan []byte)
	} else {
		channel = make(chan []byte, backlog)
	}
	return &producer{
		queue:      queue,
		url:        url,
		backlog:    channel,
		connection: nil,
		channel:    nil,
	}
}

type consumer struct {
	queue      string
	url        string
	connection *rabbit.Connection
	channel    *rabbit.Channel
}

func (c *consumer) RunWith(ctx context.Context, consume mq.Consume) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if nil != c.channel && !c.channel.IsClosed() {
			if err := c.channel.Close(); err != nil {
				log.Error(err, "Error on c.channel.Close()")
			}
			c.channel = nil
		}
		if nil != c.connection && !c.connection.IsClosed() {
			if err := c.connection.Close(); err != nil {
				log.Error(err, "Error on c.connection.Close()")
			}
			c.connection = nil
		}
		log.Info("Start connecting ...")
		ch, conn := Reconnect(c.queue, c.url)
		c.channel = ch
		c.connection = conn
		messages, err := c.channel.Consume(
			c.queue, // Queue name
			"",      // Consumer name (if empty, a unique name will be generated)
			false,   // Auto-acknowledgement
			false,   // Exclusive
			false,   // Wait for server to be ready
			false,   // No extra arguments
			nil,     // Consumer tag
		)
		if err != nil {
			log.Error(err, "Error on c.channel.Consume(..)")
			log.Info("Failed to register a consumer, retrying ...")
			continue
		}
		log.Info("Start message receiving ...")
	loopMessage:
		for {
			select {
			case <-ctx.Done():
				break loopMessage
			case message := <-messages:
				if nil != consume {
					if consume(message.Body) {
						_ = message.Ack(false)
					}
				}
			}
		}
		log.Info("Finish message receiving.")
	}
}

type producer struct {
	backlog    chan []byte
	queue      string
	url        string
	connection *rabbit.Connection
	channel    *rabbit.Channel
}

func (p *producer) SendBytes(message []byte) {
	p.backlog <- message
}

func (p *producer) SendString(message string) {
	p.SendBytes([]byte(message))
}

func (p *producer) RunWith(ctx context.Context) {
	var message []byte
	retryIndex := 0
	for {
		if message != nil {
			if p.publish(message, 5) {
				message = nil
			} else {
				time.Sleep(time.Duration(retryDelays[retryIndex]) * time.Second)
				retryIndex = (retryIndex + 1) % len(retryDelays)
				continue
			}
		}
		select {
		case message = <-p.backlog:
			continue
		case <-ctx.Done():
			log.Info("Producer is stopping ...", "queue", p.queue)
			return
		}
	}
}

func (p *producer) publish(message []byte, maxRetry int) bool {
	for i := 0; i <= maxRetry; i++ {
		if p.channel == nil {
			ch, conn := Reconnect(p.queue, p.url)
			p.channel = ch
			p.connection = conn
		}
		confirm, err := p.channel.PublishWithDeferredConfirmWithContext(
			context.Background(),
			"",      // exchange
			p.queue, // routing key
			false,   // mandatory
			false,   // immediate
			rabbit.Publishing{
				Body:         message,
				ContentType:  "text/plain",
				DeliveryMode: rabbit.Persistent,
			},
		)
		if err != nil {
			p.channel = nil
			p.connection = nil
			log.Error(err, "Error on p.channel.PublishWithDeferredConfirmWithContext(..)")
			log.Info(fmt.Sprintf("Failed to publish a message, retrying ... (%d/%d)", i+1, maxRetry))
			continue
		}
		if confirm != nil {
			if confirm.Wait() {
				log.V(1).Info(fmt.Sprintf("Message acked by server - %d", confirm.DeliveryTag))
				return true
			} else {
				log.V(1).Info(fmt.Sprintf("Message nacked by server, retrying ... (%d/%d)", i+1, maxRetry))
				time.Sleep(1 * time.Second) // wait for a while before retrying
			}
		}
	}
	return false
}
