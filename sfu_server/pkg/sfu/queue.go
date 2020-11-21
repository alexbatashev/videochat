package sfu 

type message func(string)

// Queue is something to read messages from or write to
type Queue interface {
	OnMessage(fn message)
	Write(msg string) error
}

// QueueProvider is an interface for a queue provider (RabbitMQ, Kafka, Mock)
type QueueProvider interface {
	CreateQueue(name string) Queue
	Close()
}
