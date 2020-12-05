package sfu

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	kafka "github.com/segmentio/kafka-go"
)

type KafkaQueue struct {
	Name   string
	Reader *kafka.Reader
	Writer *kafka.Writer
}

type KafkaQueueProvider struct {
	Connection *kafka.Conn
	URL        string
}

func connectToKafka(uri string) *kafka.Conn {
	for {
		conn, err := kafka.Dial("tcp", uri)
		if err == nil {
			return conn
		}

		log.Println(err)
		log.Printf("Trying to reconnect to Kafka at %s\n", uri)
		time.Sleep(500 * time.Millisecond)
	}
}

func CreateKafkaProvider(url string) (*KafkaQueueProvider, error) {
	conn := connectToKafka(url)

	return &KafkaQueueProvider{conn, url}, nil
}

func (q *KafkaQueue) OnMessage(fn message) {
	go func() {
		for {
			msg, err := q.Reader.ReadMessage(context.Background())
			if err != nil {
				log.Println(err)
			} else {
				fn(msg.Value)
			}
		}
	}()
}

func (q *KafkaQueue) Write(msg []byte) error {
	keyUUID, err := uuid.NewRandom()
	if err != nil {
		log.Println(err)
	}

	keyStr := keyUUID.String()
	kafkaMsg := kafka.Message{
		Key:   []byte(keyStr),
		Value: msg,
	}
	err = q.Writer.WriteMessages(context.Background(), kafkaMsg)
	return err
}

func (qp *KafkaQueueProvider) CreateQueue(name string) (Queue, error) {
	topic := kafka.TopicConfig{
		Topic:             name,
		NumPartitions:     -1,
		ReplicationFactor: -1,
	}
	err := qp.Connection.CreateTopics(topic)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{qp.URL},
		GroupID:  name,
		Topic:    name,
		MinBytes: 10,   // 10KB
		MaxBytes: 10e6, // 10MB
	})
	writer := &kafka.Writer{
		Addr:     kafka.TCP(qp.URL),
		Topic:    name,
		Balancer: &kafka.LeastBytes{},
	}
	log.Println("Created queue " + name)
	return &KafkaQueue{name, reader, writer}, nil
}

func (qp KafkaQueueProvider) Close() {
	qp.Connection.Close()
}
