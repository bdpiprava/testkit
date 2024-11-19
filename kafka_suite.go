package testkit

import (
	"strings"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/sirupsen/logrus"

	"github.com/bdpiprava/testkit/context"
)

const pollTimeout = 10 * time.Second
const deliveryTimeout = 10 * time.Second

// OnMessage is a callback function that is called when a message is received
type OnMessage func(*kafka.Message) bool

// KafkaSuite is a suite that provides tooling for postgres integration tests
type KafkaSuite struct {
	context.CtxSuite
	servers     map[string]*kafka.MockCluster
	consumers   []*kafka.Consumer
	initializer sync.Once
}

// RequiresKafka is a helper function to get the test database based on configuration
// returns the server address
func (s *KafkaSuite) RequiresKafka(topics ...string) string {
	err := s.Initialize(s.T().Name())
	s.Require().NoError(err)

	s.initializer.Do(func() {
		s.servers = make(map[string]*kafka.MockCluster)
		s.consumers = make([]*kafka.Consumer, 0)
	})

	ctx := s.GetContext(s.T().Name())
	log := context.GetLogger(*ctx).WithFields(logrus.Fields{
		"test": s.T().Name(),
		"func": "RequiresKafka",
	})
	if cluster, ok := s.servers[s.T().Name()]; ok {
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
	s.servers[s.T().Name()] = cluster
	return cluster.BootstrapServers()
}

// Produce a message to the kafka topic
func (s *KafkaSuite) Produce(topic string, key, value []byte, headers ...kafka.Header) {
	ctx := s.GetContext(s.T().Name())
	servers := s.getCluster().BootstrapServers()
	log := context.GetLogger(*ctx).WithFields(logrus.Fields{
		"test":   s.T().Name(),
		"func":   "Produce",
		"topic":  topic,
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
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Headers: headers,
		Key:     key,
		Value:   value,
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
func (s *KafkaSuite) Consume(topics []string, callback OnMessage) {
	ctx := s.GetContext(s.T().Name())
	servers := s.getCluster().BootstrapServers()
	log := context.GetLogger(*ctx).WithFields(logrus.Fields{
		"test":   s.T().Name(),
		"func":   "Consume",
		"topics": strings.Join(topics, ","),
		"server": servers,
	})

	s.NotNil(callback, "callback is required")
	log.Info("Creating consumer")
	consumer, err := kafka.NewConsumer(s.getKafkaConfig())
	s.Require().NoError(err)
	s.consumers = append(s.consumers, consumer)
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

func (s *KafkaSuite) getKafkaConfig() *kafka.ConfigMap {
	return &kafka.ConfigMap{
		"bootstrap.servers": s.getCluster().BootstrapServers(),
		"group.id":          s.T().Name(),
		"auto.offset.reset": "earliest",
	}
}

// getCluster returns the kafka cluster
func (s *KafkaSuite) getCluster() *kafka.MockCluster {
	cluster, ok := s.servers[s.T().Name()]
	if !ok {
		s.Require().Fail("Kafka cluster not found. call RequiresKafka before calling Produce")
	}
	return cluster
}

// TearDownSuite perform the cleanup of the database
func (s *KafkaSuite) TearDownSuite() {
	for _, c := range s.consumers {
		closeSilently(c)
	}

	for _, s := range s.servers {
		s.Close()
	}
}
