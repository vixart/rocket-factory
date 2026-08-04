package kafka

import (
	"context"
)

// MessageHandler processes a single incoming Kafka message.
//
// It is the terminal handler of the consumer pipeline:
//
//	handler := func(ctx context.Context, msg kafka.Message) error {
//	    // decode msg.Value, process it, return an error on failure
//	}
//
// When it returns an error the message offset is NOT committed (at-least-once semantics).
type MessageHandler func(ctx context.Context, msg Message) error

// Middleware wraps a MessageHandler to add cross-cutting behaviour
// (logging, metrics, tracing) without touching the business handler.
//
// Middlewares are applied outside-in: the first one wraps the whole chain,
// the last one sits closest to the business handler.
type Middleware func(next MessageHandler) MessageHandler
