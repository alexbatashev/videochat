package sfu

import (
	"context"
	b64 "encoding/base64"
	"log"
	"net"
	"strconv"
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
	Connection           *kafka.Conn
	URL                  string
	ControllerConnection *kafka.Conn
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

	controller, err := conn.Controller()
	if err != nil {
		return nil, err
	}
	var controllerConn *kafka.Conn
	controllerConn, err = kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		return nil, err
	}

	return &KafkaQueueProvider{conn, url, controllerConn}, nil
}

func (q *KafkaQueue) OnMessage(fn message) {
	go func() {
		for {
			msg, err := q.Reader.ReadMessage(context.Background())
			if err != nil {
				log.Println(err)
			} else {
				value := msg.Value
				res, err := b64.StdEncoding.DecodeString(string(value))
				if err == nil {
					fn(res)
				}
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
	topicConfigs := []kafka.TopicConfig{
		kafka.TopicConfig{
			Topic:             name,
			NumPartitions:     1,
			ReplicationFactor: 3,
		},
	}
	err := qp.ControllerConnection.CreateTopics(topicConfigs...)
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
	qp.ControllerConnection.Close()
	qp.Connection.Close()
}
