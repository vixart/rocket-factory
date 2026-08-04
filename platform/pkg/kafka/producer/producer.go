package producer

import (
	"context"
	"log/slog"

	"github.com/IBM/sarama"

	"github.com/vixart/rocket-factory/platform/pkg/kafka"
)

// Producer wraps sarama.SyncProducer and sends messages to a single topic.
//
// IMPORTANT: sarama.SyncProducer requires a sarama.Config with
// Producer.Return.Successes = true, otherwise SendMessage blocks forever. Make sure
// it is set when the sarama.Config is built (usually in the service config layer).
type Producer struct {
	syncProducer sarama.SyncProducer
	topic        string
}

// NewProducer creates a Producer bound to a single topic.
// syncProducer must be created with Producer.Return.Successes = true.
func NewProducer(syncProducer sarama.SyncProducer, topic string) *Producer {
	return &Producer{
		syncProducer: syncProducer,
		topic:        topic,
	}
}

// Send publishes a message synchronously and blocks until the broker acknowledges it.
// It returns an error when the write is not acknowledged.
func (p *Producer) Send(ctx context.Context, msg *kafka.Message) error {
	saramaMsg := &sarama.ProducerMessage{
		Topic:   p.topic,
		Key:     sarama.ByteEncoder(msg.Key),
		Value:   sarama.ByteEncoder(msg.Value),
		Headers: convertHeaders(msg.Headers),
	}

	partition, offset, err := p.syncProducer.SendMessage(saramaMsg)
	if err != nil {
		slog.ErrorContext(ctx, "failed to send message", "error", err)
		return err
	}

	slog.InfoContext(
		ctx, "message sent",
		"topic", p.topic,
		"partition", partition,
		"offset", offset,
		"key", string(msg.Key),
		"value_size", len(msg.Value),
	)

	return nil
}

func convertHeaders(headers []kafka.Header) []sarama.RecordHeader {
	if len(headers) == 0 {
		return nil
	}

	result := make([]sarama.RecordHeader, 0, len(headers))
	for _, h := range headers {
		result = append(result, sarama.RecordHeader{
			Key:   []byte(h.Key),
			Value: h.Value,
		})
	}

	return result
}
