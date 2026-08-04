package kafka

import (
	"context"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/platform/pkg/auth"
	"github.com/vixart/rocket-factory/platform/pkg/kafka"
)

// SessionHeaderKey is the Kafka header the services use to pass the user session
// UUID between each other. It matches the gRPC metadata key (see
// order/internal/interceptor/auth.go) so that the same session-uuid travels the whole
// HTTP → Kafka → gRPC chain under one name.
const SessionHeaderKey = "session-uuid"

// ProducerSessionHeaders builds the Kafka headers of an outgoing message from the
// session-uuid in the context. When there is none it returns nil: the header is
// omitted, the consumer on the other side cannot restore the session-uuid, and any
// protected gRPC call made from the handler fails with Unauthenticated.
//
// Usage in a producer service:
//
//	return s.producer.Send(ctx, &kafka.Message{
//	    Key:     key,
//	    Value:   payload,
//	    Headers: kafkamw.ProducerSessionHeaders(ctx),
//	})
func ProducerSessionHeaders(ctx context.Context) []kafka.Header {
	sessionUUID, ok := auth.SessionUUIDFromContext(ctx)
	if !ok || sessionUUID == uuid.Nil {
		return nil
	}
	return []kafka.Header{
		{Key: SessionHeaderKey, Value: []byte(sessionUUID.String())},
	}
}

// ConsumerSession is a Kafka middleware that reads the session-uuid from the message
// headers and puts it into the context before calling the main handler.
//
// Why: the gRPC client SessionForwarder (see order/internal/interceptor/auth.go) takes
// the session-uuid from the context and forwards it in the outgoing metadata. Without
// this middleware the Kafka handler gets a bare context from
// sarama.ConsumerGroupSession with no session-uuid, and any protected gRPC call
// (InventoryService.CommitParts, for example) returns codes.Unauthenticated.
//
// When the header is missing or invalid the context is left untouched. That is
// deliberate: the middleware must not decide on behalf of the business handler what
// to do about a missing session (drop it, route it to a DLQ, run under a service
// identity) — that is the responsibility of the individual service.
func ConsumerSession() kafka.Middleware {
	return func(next kafka.MessageHandler) kafka.MessageHandler {
		return func(ctx context.Context, msg kafka.Message) error {
			for _, h := range msg.Headers {
				if h.Key == SessionHeaderKey && len(h.Value) > 0 {
					if sessionUUID, err := uuid.Parse(string(h.Value)); err == nil {
						ctx = auth.WithSessionUUID(ctx, sessionUUID)
					}
					break
				}
			}
			return next(ctx, msg)
		}
	}
}
