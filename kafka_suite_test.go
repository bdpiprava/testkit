package testkit_test

import (
	"sync"
	"testing"

	"github.com/bdpiprava/testkit"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/suite"
)

type KafkaTestSuiteTest struct {
	testkit.KafkaSuite
}

func TestKafkaTestSuiteTest(t *testing.T) {
	suite.Run(t, new(KafkaTestSuiteTest))
}

func (s *KafkaTestSuiteTest) Test_RequiresKafka() {
	topics := []string{"topic1", "topic2"}
	key := "this is message key"
	value := "this is message value"
	s.RequiresKafka(topics...)

	var wg sync.WaitGroup
	wg.Add(1)
	count := 0
	s.Consume(topics, func(msg *kafka.Message) bool {
		if string(msg.Key) == key && string(msg.Value) == value {
			count++
		}

		if count == 2 {
			defer wg.Done()
			return true
		}
		return false
	})

	s.Produce(topics[0], []byte(key), []byte(value))
	s.Produce(topics[1], []byte(key), []byte(value))
	wg.Wait()
}
