package testkit_test

import (
	"sync"
	"testing"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"

	"github.com/bdpiprava/testkit"
)

type KafkaTestSuiteTest struct {
	testkit.Suite
}

func TestKafkaTestSuiteTest(t *testing.T) {
	testkit.Run(t, new(KafkaTestSuiteTest))
}

func (s *KafkaTestSuiteTest) SetupSuite() {
	s.RequiresKafka("suite_topic")
}

func (s *KafkaTestSuiteTest) Test_ShouldGetKafkaServerFromTheParentTestWhenNotDeclaredAtTestLevel() {
	var wg sync.WaitGroup
	wg.Add(1)
	s.Run("level2", func() {
		s.RequiresKafka("level2_topic")
		s.Run("level3", func() {
			s.Run("level4", func() {
				s.Produce("level2_topic", []byte("key"), []byte("value"))
				s.Consume([]string{"level2_topic"}, func(_ *kafka.Message) bool {
					wg.Done()
					return true
				})
			})
		})
	})
	wg.Wait()
}

func (s *KafkaTestSuiteTest) Test_ShouldGetKafkaServerFromTheSuiteWhenNotDeclaredAtTestLevel() {
	var wg sync.WaitGroup
	wg.Add(1)
	s.Run("level2", func() {
		s.Run("level3", func() {
			s.Run("level4", func() {
				s.Produce("suite_topic", []byte("key"), []byte("value"))
				s.Consume([]string{"suite_topic"}, func(_ *kafka.Message) bool {
					wg.Done()
					return true
				})
			})
		})
	})
	wg.Wait()
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
