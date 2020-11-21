package sfu

import (
	"github.com/streadway/amqp"
)

type RabbitMQQueue struct {
	Name      string
	RealQueue amqp.Queue
	Channel   amqp.Channel
}

type RabbitMQQueueProvider struct {
	Connection amqp.Connection
}

func CreateRabbitMQProvider(url string) (RabbitMQQueueProvider, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	return RabbitMQQueueProvider{conn}, nil
}

func (q RabbitMQQueue) OnMessage(fn message) {
	go func() {
		msgs, err := q.Channel.Consume(
			q.Name, // queue
			"",     // consumer
			true,   // auto-ack
			false,  // exclusive
			false,  // no-local
			false,  // no-wait
			nil,    // args
		)

		if err != nil {
			// TODO log error
			return
		}

		for d := range msgs {
			go fn(d.Body)
		}
	}()
}

func (q RabbitMQQueue) Write(msg string) error {
	err := q.Channel.Publish(
		"", // exchange name
		q.Name, // queue name
		true, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(msg),
		}
	)
	return err
}

func (qp RabbitMQQueueProvider) CreateQueue(name string) (RabbitMQQueue, error) {
	ch, err := qp.Connection.Channel()
	if err != nil {
		return nil, err
	}

	q, err := ch.QueueDeclare(
		name,  // name
		false, // durable
		true,  // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return nil, err
	}

	return RabbitMQQueue{name, q, ch}, nil
}

func (qp RabbitMQQueueProvider) Close() {
	qp.Connection.Close()
}
