package consumer

import (
	"context"
	"errors"
	"log/slog"

	"github.com/IBM/sarama"

	"github.com/vixart/rocket-factory/platform/pkg/kafka"
)

// Option is a functional option for configuring a Consumer.
type Option func(*Consumer)

// WithMiddlewares attaches a middleware chain to the Consumer.
// Middlewares apply in the order given: the first one wraps the whole chain.
func WithMiddlewares(mws ...kafka.Middleware) Option {
	return func(c *Consumer) {
		c.middlewares = append(c.middlewares, mws...)
	}
}

// Consumer wraps sarama.ConsumerGroup and adds middleware support.
//
// It runs an endless consume loop over the given topics. On a consumer group
// rebalance sarama returns from Consume and it has to be called again —
// the loop inside Consume handles that automatically.
type Consumer struct {
	group       sarama.ConsumerGroup
	topics      []string
	middlewares []kafka.Middleware
}

// NewConsumer creates a Consumer for the given topics.
//
// group must be created via sarama.NewConsumerGroup with a valid configuration:
//   - Consumer.Offsets.Initial — initial offset strategy (OffsetOldest / OffsetNewest)
//   - Consumer.Group.Rebalance.GroupStrategies — rebalance strategy (RoundRobin, Range, ...)
func NewConsumer(group sarama.ConsumerGroup, topics []string, opts ...Option) *Consumer {
	c := &Consumer{
		group:  group,
		topics: topics,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Consume runs the endless message consumption loop.
//
// handler is called for every message. On success (nil) the message is marked as
// processed (at-least-once semantics). On error the message is logged and skipped:
// the offset is NOT committed, but consumption continues.
//
// The method blocks until ctx is cancelled or a fatal error occurs.
func (c *Consumer) Consume(ctx context.Context, handler kafka.MessageHandler) error {
	newGroupHandler := NewGroupHandler(handler, c.middlewares...)

	for {
		if err := c.group.Consume(ctx, c.topics, newGroupHandler); err != nil {
			if errors.Is(err, sarama.ErrClosedConsumerGroup) {
				return nil
			}

			slog.ErrorContext(ctx, "Kafka consumption failed", "error", err)
			return err
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		// After a rebalance sarama returns from the current Consume, so it has to be
		// called again to pick up the newly assigned partitions.
		slog.InfoContext(ctx, "Kafka consumer group is rebalancing...")
	}
}
