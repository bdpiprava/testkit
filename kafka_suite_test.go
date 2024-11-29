package testkit_test

import (
	"sync"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"

	"github.com/bdpiprava/testkit"
)

type KafkaTestSuiteTest struct {
	testkit.Suite
	suiteLevelTopic string
}

func TestKafkaTestSuiteTest(t *testing.T) {
	testkit.Run(t, new(KafkaTestSuiteTest))
}

func (s *KafkaTestSuiteTest) SetupSuite() {
	s.suiteLevelTopic = uuid.New().String()
	s.RequiresKafka(s.suiteLevelTopic)
}

func (s *KafkaTestSuiteTest) Test_ShouldGetKafkaServerFromTheParentTestWhenNotDeclaredAtTestLevel() {
	topic := uuid.New().String()
	var wg sync.WaitGroup
	wg.Add(1)
	s.Run("level2", func() {
		s.RequiresKafka(topic)
		s.Run("level3", func() {
			s.Run("level4", func() {
				s.Produce(topic, []byte("key"), []byte("value"))
				s.Consume([]string{topic}, func(_ *kafka.Message) bool {
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
				s.Produce(s.suiteLevelTopic, []byte("key"), []byte("value"))
				s.Consume([]string{s.suiteLevelTopic}, func(_ *kafka.Message) bool {
					wg.Done()
					return true
				})
			})
		})
	})
	wg.Wait()
}

func (s *KafkaTestSuiteTest) Test_RequiresKafka() {
	topics := []string{uuid.New().String(), uuid.New().String()}
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

func (s *KafkaTestSuiteTest) Test_WaitForMessage_Success() {
	topic := uuid.New().String()
	key := "this is message key"
	value := "this is message value"
	s.RequiresKafka(topic)

	// When
	s.Produce(topic, []byte(key), []byte(value))

	// Then
	got, gotErr := s.WaitForMessage(topic, time.Second*5)
	s.NotNil(got)
	s.NoError(gotErr)
	s.Equal(key, string(got.Key))
	s.Equal(value, string(got.Value))
}

func (s *KafkaTestSuiteTest) Test_WaitForMessage_Timeout() {
	topic := uuid.New().String()
	s.RequiresKafka(topic)

	// When no message is produced

	// Then
	got, gotErr := s.WaitForMessage(topic, time.Second)
	s.Nil(got)
	s.EqualError(gotErr, "timeout reached while waiting for the message in topic "+topic)
}
