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
					go fn(res)
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
	log.Println("Sending a message!")
	retryCounter := 0
	for {
		err = q.Writer.WriteMessages(context.Background(), kafkaMsg)
		if err == nil || retryCounter == 50 {
			break
		}
		if err != nil {
			log.Println(err)
		}
		retryCounter++
		time.Sleep(20 * time.Millisecond)
	}
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
	var brokers []string
	goBrokers, err := qp.Connection.Brokers()
	for _, b := range goBrokers {
		bStr := b.Host + ":" + strconv.Itoa(b.Port)
		brokers = append(brokers, bStr)
	}
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  name,
		Topic:    name,
		MinBytes: 10,   // 10KB
		MaxBytes: 10e6, // 10MB
	})
	writer := &kafka.Writer{
		Addr:      kafka.TCP(brokers[0]),
		Topic:     name,
		BatchSize: 1,
		RequiredAcks: kafka.RequireAll,
		MaxAttempts: 500,
	}
	log.Println("Created queue " + name)
	return &KafkaQueue{name, reader, writer}, nil
}

func (qp KafkaQueueProvider) Close() {
	qp.ControllerConnection.Close()
	qp.Connection.Close()
}
