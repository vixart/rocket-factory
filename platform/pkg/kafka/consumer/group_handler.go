package consumer

import (
	"context"
	"log/slog"

	"github.com/IBM/sarama"
	"github.com/go-faster/errors"

	"github.com/vixart/rocket-factory/platform/pkg/kafka"
)

// groupHandler implements sarama.ConsumerGroupHandler.
//
// Lifecycle (driven by sarama on every rebalance):
//  1. Setup        — called once when partitions are assigned
//  2. ConsumeClaim — called per assigned partition, each in its own goroutine
//  3. Cleanup      — called after every ConsumeClaim has returned
type groupHandler struct {
	handler kafka.MessageHandler
}

// NewGroupHandler builds a groupHandler by wrapping handler into the middleware chain.
//
// Middlewares are applied in reverse order (the last one ends up closest to the
// handler) so that the call order matches the order they were passed in:
// WithMiddlewares(logging, metrics) → logging → metrics → handler.
func NewGroupHandler(handler kafka.MessageHandler, middlewares ...kafka.Middleware) *groupHandler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}

	return &groupHandler{
		handler: handler,
	}
}

func (g *groupHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (g *groupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim reads messages from a single partition until it closes or the session is cancelled.
//
// At-least-once semantics: MarkMessage is called only after successful processing.
// On handler error the message is logged and skipped — the consumer group will
// receive it again after a restart, since the offset was not committed.
func (g *groupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				slog.InfoContext(session.Context(), "Kafka message channel closed")
				return nil
			}

			msg := kafka.Message{
				Key:       message.Key,
				Value:     message.Value,
				Topic:     message.Topic,
				Partition: message.Partition,
				Offset:    message.Offset,
				Timestamp: message.Timestamp,
				Headers:   extractHeaders(message.Headers),
			}

			if err := g.handler(session.Context(), msg); err != nil {
				slog.ErrorContext(session.Context(), "Kafka handler failed", "error", err)

				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return err // do NOT commit: graceful shutdown
				}

				continue
			}

			session.MarkMessage(message, "")

		case <-session.Context().Done():
			slog.InfoContext(session.Context(), "Kafka session context is done")
			return nil
		}
	}
}

func extractHeaders(headers []*sarama.RecordHeader) []kafka.Header {
	result := make([]kafka.Header, 0, len(headers))
	for _, h := range headers {
		if h != nil && h.Key != nil {
			result = append(result, kafka.Header{
				Key:   string(h.Key),
				Value: h.Value,
			})
		}
	}

	return result
}
