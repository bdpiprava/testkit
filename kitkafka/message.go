// Package kitkafka ...
package kitkafka

import (
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// Message represents a Kafka message
type Message struct {
	Topic         string
	Value         []byte
	Key           []byte
	Headers       []kafka.Header
	Timestamp     time.Time
	TimestampType kafka.TimestampType
}
