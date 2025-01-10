package testkit

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/sirupsen/logrus"

	"github.com/bdpiprava/testkit/kitkafka"
)

const pollTimeout = 10 * time.Second
const deliveryTimeout = 10 * time.Second

var mu sync.RWMutex

// OnMessage is a callback function that is called when a message is received
type OnMessage func(*kafka.Message) bool

// RequiresKafka is a helper function to get the test database based on configuration
// returns the server address
func (s *Suite) RequiresKafka(topics ...string) string {
	log := s.Logger().WithFields(logrus.Fields{
		"test": s.T().Name(),
		"func": "RequiresKafka",
	})

	mu.RLock()
	if cluster, ok := s.kafkaServers[s.T().Name()]; ok {
		log.Tracef("Kafka cluster already exists, returning bootstrap servers %s", cluster.BootstrapServers())
		mu.RUnlock()
		return cluster.BootstrapServers()
	}
	mu.RUnlock()

	log.Trace("Creating new Kafka cluster")
	cluster, err := kafka.NewMockCluster(1)
	s.Require().NoError(err)

	log.Infof("Creating topics: %v", topics)
	for _, topic := range topics {
		s.Require().NoError(cluster.CreateTopic(topic, 1, 1))
	}

	log.Infof("Topics created: %v", topics)
	mu.Lock()
	defer mu.Unlock()
	s.kafkaServers[s.T().Name()] = cluster
	return cluster.BootstrapServers()
}

// Produce a message to the kafka topic
func (s *Suite) Produce(topic string, key, value []byte, headers ...kafka.Header) {
	s.ProduceMessage(kitkafka.Message{
		Topic:   topic,
		Key:     key,
		Value:   value,
		Headers: headers,
	})
}

// ProduceMessage produce a message to kafka cluster
func (s *Suite) ProduceMessage(message kitkafka.Message) {
	servers := s.getCluster().BootstrapServers()
	log := s.Logger().WithFields(logrus.Fields{
		"test":   s.T().Name(),
		"func":   "Produce",
		"topic":  message.Topic,
		"server": servers,
	})

	log.Info("Creating producer")
	producer, err := kafka.NewProducer(s.getKafkaConfig())
	s.Require().NoError(err)
	defer producer.Close()

	log.Info("Producing message")
	deliveryChan := make(chan kafka.Event)
	err = producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &message.Topic,
			Partition: kafka.PartitionAny,
		},
		Headers:       message.Headers,
		Key:           message.Key,
		Value:         message.Value,
		Timestamp:     message.Timestamp,
		TimestampType: message.TimestampType,
	}, deliveryChan)
	s.Require().NoError(err)

	log.Info("Waiting for delivery confirmation")
	select {
	case <-time.After(deliveryTimeout):
		s.FailNow("Delivery timeout")
	case <-deliveryChan:
		log.Info("Delivered")
		break
	}
}

// Consume a message from the kafka topic
func (s *Suite) Consume(topics []string, callback OnMessage) {
	servers := s.getCluster().BootstrapServers()
	log := s.Logger().WithFields(logrus.Fields{
		"test":   s.T().Name(),
		"func":   "Consume",
		"topics": strings.Join(topics, ","),
		"server": servers,
	})

	s.NotNil(callback, "callback is required")
	log.Info("Creating consumer")
	consumer, err := kafka.NewConsumer(s.getKafkaConfig())
	s.Require().NoError(err)
	mu.Lock()
	defer mu.Unlock()
	s.kafkaConsumers = append(s.kafkaConsumers, consumer)
	s.Require().NoError(consumer.SubscribeTopics(topics, nil))

	go func(consumer *kafka.Consumer) {
		var wg sync.WaitGroup
		for {
			wg.Add(1)
			if s.doConsume(consumer, log, callback, &wg) {
				break
			}
			wg.Wait()
		}
	}(consumer)
}

func (s *Suite) doConsume(consumer *kafka.Consumer, log *logrus.Entry, callback OnMessage, wg *sync.WaitGroup) bool {
	defer wg.Done()
	if consumer.IsClosed() {
		return true
	}

	ev := consumer.Poll(int(pollTimeout.Milliseconds()))
	switch e := ev.(type) {
	case *kafka.Message:
		log.Trace("Received message")
		return callback(e)
	case kafka.PartitionEOF:
		log.Info("Partition EOF")
		break
	case kafka.Error:
		log.Warn(fmt.Sprintf("Received error from kafka: %#v", e))
	case kafka.AssignedPartitions:
		s.Require().NoError(consumer.Assign(e.Partitions))
	}

	return false
}

// WaitForMessage waits for a message to be consumed from the kafka topics
func (s *Suite) WaitForMessage(topic string, timout time.Duration) (*kafka.Message, error) {
	timeoutTimer := time.NewTimer(timout)
	defer timeoutTimer.Stop()

	done := make(chan struct{})
	received := make(chan *kafka.Message, 1)
	s.Consume([]string{topic}, func(msg *kafka.Message) bool {
		received <- msg
		close(done)
		return true
	})

	// Then - wait for the message to be consumed
	select {
	case <-done:
		return <-received, nil
	case <-timeoutTimer.C:
		return nil, fmt.Errorf("timeout reached while waiting for the message in topic %s", topic)
	}
}

func (s *Suite) getKafkaConfig() *kafka.ConfigMap {
	return &kafka.ConfigMap{
		"bootstrap.servers": s.getCluster().BootstrapServers(),
		"group.id":          s.T().Name(),
		"auto.offset.reset": "earliest",
	}
}

// getCluster returns the kafka cluster for current test or from parent tests or suite
func (s *Suite) getCluster() *kafka.MockCluster {
	mu.RLock()
	defer mu.RUnlock()
	name := s.T().Name()
	for {
		cluster, ok := s.kafkaServers[name]
		if ok {
			return cluster
		}

		if idx := strings.LastIndex(name, "/"); idx <= 0 {
			break
		}

		name = name[:strings.LastIndex(name, "/")]
	}

	s.Require().Fail("Kafka cluster not found. call RequiresKafka before calling Produce")
	return nil
}

// cleanKafkaResources closes the kafka consumers and servers
func (s *Suite) cleanKafkaResources() {
	mu.RLock()
	defer mu.RUnlock()
	for _, c := range s.kafkaConsumers {
		closeSilently(c)
	}

	for _, server := range s.kafkaServers {
		if server != nil {
			server.Close()
		}
	}
}
