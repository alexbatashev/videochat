package sfu 

type message func([]byte)

// Queue is something to read messages from or write to
type Queue interface {
	OnMessage(fn message)
	Write(msg []byte) error
}

// QueueProvider is an interface for a queue provider (RabbitMQ, Kafka, Mock)
type QueueProvider interface {
	CreateQueue(name string) (Queue, error)
	Close()
}
