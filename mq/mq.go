package mq

import (
	"context"
)

type Producer interface {
	RunWith(ctx context.Context)
	SendBytes(message []byte)
	SendString(message string)
}

type Consume func(message []byte) bool

type Consumer interface {
	RunWith(ctx context.Context, consume Consume)
}
