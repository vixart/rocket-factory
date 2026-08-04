package kafka

import (
	"context"
	"log/slog"

	"github.com/vixart/rocket-factory/platform/pkg/kafka"
)

// ConsumerLogging is a middleware that logs incoming messages.
func ConsumerLogging() kafka.Middleware {
	return func(next kafka.MessageHandler) kafka.MessageHandler {
		return func(ctx context.Context, msg kafka.Message) error {
			slog.InfoContext(ctx, "Kafka message received", "topic", msg.Topic)
			return next(ctx, msg)
		}
	}
}
