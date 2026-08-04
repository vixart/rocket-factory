package kafka

import "time"

// Header is a Kafka message header (key-value pair)
// Kafka allows duplicate keys, so a slice is used instead of a map.
type Header struct {
	Key   string
	Value []byte
}

// Message is a transport-agnostic wrapper around a Kafka message.
type Message struct {
	Headers   []Header
	Timestamp time.Time

	Key       []byte
	Value     []byte
	Topic     string
	Partition int32
	Offset    int64
}
