package main

import (
	"log"
	"encoding/json"

	"github.com/pion/webrtc/v2"
	"github.com/streadway/amqp"
)

func init() {
	// Create a MediaEngine object to configure the supported codec
	m = webrtc.MediaEngine{}

	// Setup the codecs you want to use.
	m.RegisterCodec(webrtc.NewRTPVP8Codec(webrtc.DefaultPayloadTypeVP8, 90000))
	m.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))

	// Create the API object with the MediaEngine
	api = webrtc.NewAPI(webrtc.WithMediaEngine(m))
}

func failOnError(err error, msg string) {
  if err != nil {
    log.Fatalf("%s: %s", msg, err)
  }
}

type Message struct {
	command string
	roomId string
	peerId string
	data string
}

func main() {
	// init()

	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	failOnError(err, "Failed to connect to rabbitmq")
	defer conn.Close()

	ch, err := conn.Channel()
  failOnError(err, "Failed to open a channel")
  defer ch.Close()

	q, err := ch.QueueDeclare(
		"sfu", // name
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	failOnError(err, "Failed to declare a queue")

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

go func() {
  for d := range msgs {
		log.Printf("Received a message: %s", d.Body)
		command := &Message{}
		err := json.Unmarshal(d.Body, command)

		if err != nil {
			log.Printf("Error decoding JSON: %s", err)
		}

		// TODO acknowledge message
		if err := d.Ack(false); err != nil {
			log.Printf("Error acknowledging message : %s", err)
		} else {
			log.Printf("Acknowledged message")
		}


		if command.command == "create_room" {
			createRoom(command.roomId)
		}

		if command.command == "add_peer" {
			_ = addPeer(command.roomId, command.peerId, command.data)
		}
  }
}()

log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
<-forever
}
