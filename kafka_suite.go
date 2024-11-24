package testkit

import (
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/sirupsen/logrus"

	"github.com/bdpiprava/testkit/kitkafka"
)

const pollTimeout = 10 * time.Second
const deliveryTimeout = 10 * time.Second

// OnMessage is a callback function that is called when a message is received
type OnMessage func(*kafka.Message) bool

// RequiresKafka is a helper function to get the test database based on configuration
// returns the server address
func (s *Suite) RequiresKafka(topics ...string) string {
	log := s.Logger().WithFields(logrus.Fields{
		"test": s.T().Name(),
		"func": "RequiresKafka",
	})
	if cluster, ok := s.kafkaServers[s.T().Name()]; ok {
		log.Tracef("Kafka cluster already exists, returning bootstrap servers %s", cluster.BootstrapServers())
		return cluster.BootstrapServers()
	}

	log.Trace("Creating new Kafka cluster")
	cluster, err := kafka.NewMockCluster(1)
	s.Require().NoError(err)

	log.Infof("Creating topics: %v", topics)
	for _, topic := range topics {
		s.Require().NoError(cluster.CreateTopic(topic, 1, 1))
	}

	log.Infof("Topics created: %v", topics)
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
	s.kafkaConsumers = append(s.kafkaConsumers, consumer)
	s.Require().NoError(consumer.SubscribeTopics(topics, nil))

	go func(consumer *kafka.Consumer) {
		done := false
		for {
			if done {
				closeSilently(consumer)
				break
			}

			ev := consumer.Poll(int(pollTimeout.Milliseconds()))
			switch e := ev.(type) {
			case *kafka.Message:
				log.Trace("Received message")
				if callback(e) {
					done = true
				}
			case kafka.PartitionEOF:
				log.Info("Partition EOF")
				break
			case kafka.Error:
				s.Require().NoError(e)
			case kafka.AssignedPartitions:
				s.Require().NoError(consumer.Assign(e.Partitions))
			}
		}
	}(consumer)
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
	for _, c := range s.kafkaConsumers {
		closeSilently(c)
	}

	for _, server := range s.kafkaServers {
		if server != nil {
			server.Close()
		}
	}
}
