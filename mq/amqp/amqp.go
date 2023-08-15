package amqp

import (
	"context"
	"log"
	"time"

	rabbit "github.com/rabbitmq/amqp091-go"
	"github.com/struqt/x/mq"
)

var retryDelays = []int{1, 4, 9, 16}

func Reconnect(name, url string) (*rabbit.Channel, *rabbit.Connection) {
	retryIndex := 0
	for {
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
						log.Printf("Successfully connected to <%s %s>.\n", name, url)
						return ch, conn
					} else {
						log.Println(err)
					}
				} else {
					log.Println(err)
				}
				if err = ch.Close(); err != nil {
					log.Println(err)
				}
			} else {
				log.Println(err)
			}
			if err = conn.Close(); err != nil {
				log.Println(err)
			}
		} else {
			log.Println(err)
		}
		log.Printf("Failed to connect to <%s %s>, retrying in %d seconds...\n", name, url, retryDelays[retryIndex])
		time.Sleep(time.Duration(retryDelays[retryIndex]) * time.Second)
		retryIndex = (retryIndex + 1) % len(retryDelays)
	}
}

func NewConsumer(queue, url string) mq.Consumer {
	return &consumer{
		queue:      queue,
		url:        url,
		connection: nil,
		channel:    nil,
	}
}

func NewProducer(queue, url string, backlog int) mq.Producer {
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
				log.Println(err)
			}
			c.channel = nil
		}
		if nil != c.connection && !c.connection.IsClosed() {
			if err := c.connection.Close(); err != nil {
				log.Println(err)
			}
			c.connection = nil
		}
		log.Printf("Start connecting ...")
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
			log.Println(err)
			log.Printf("Failed to register a consumer, retrying ...")
			continue
		}
		log.Printf("Start message receiving ...")
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
		log.Printf("Finish message receiving")
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
			log.Printf("Producer <%s %s> is stopping...\n", p.queue, p.url)
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
			log.Println(err)
			log.Printf("Failed to publish a message, retrying...(%d/%d)\n", i+1, maxRetry)
			continue
		}
		if confirm != nil {
			if confirm.Wait() {
				log.Printf("Message acked by server - %d\n", confirm.DeliveryTag)
				return true
			} else {
				log.Printf("Message nacked by server, retrying...(%d/%d)\n", i+1, maxRetry)
				time.Sleep(1 * time.Second) // wait for a while before retrying
			}
		}
	}
	return false
}
