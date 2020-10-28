package main

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/pion/webrtc/v2"
	"github.com/streadway/amqp"
)

type Peer struct {
	id             string
	connection     *webrtc.PeerConnection
	videoTrackLock sync.RWMutex
	audioTrackLock sync.RWMutex
	videoTrack     *webrtc.Track
	audioTrack     *webrtc.Track
}
type Room struct {
	peers     map[string]*Peer
	peersLock sync.RWMutex
}

func initAll() {
	// Create a MediaEngine object to configure the supported codec
	m = webrtc.MediaEngine{}

	// Setup the codecs you want to use.
	m.RegisterCodec(webrtc.NewRTPVP8Codec(webrtc.DefaultPayloadTypeVP8, 90000))
	m.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))

	// Create the API object with the MediaEngine
	api = webrtc.NewAPI(webrtc.WithMediaEngine(m))

	rooms = make(map[string]*Room)
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

type Message struct {
	Command string
	RoomId  string
	PeerId  string
	Data    string
}

type PeerMsg struct {
	Command string
	PeerId  string
	Data    string
}

func main() {
	initAll()

	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq/")
	failOnError(err, "Failed to connect to rabbitmq")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"sfu", // name
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	failOnError(err, "Failed to declare a queue")

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
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

			log.Printf("Command is %s", command.Command)

			if command.Command == "create_room" {
				log.Println("Creating new room")
				createRoom(command.RoomId)
			}

			if command.Command == "add_peer" {
				log.Println("Adding peer")
				peerChannel, err := conn.Channel()
				if err != nil {
					panic(err)
				}
				peerQueue, err := peerChannel.QueueDeclare(
					command.PeerId, // name
					false,          // durable
					false,          // delete when unused
					false,          // exclusive
					false,          // no-wait
					nil,            // arguments
				)
				offer := addPeer(command.RoomId, command.PeerId, command.Data)
				log.Printf("New offer is %s", offer)
				err = peerChannel.Publish(
					"", // exchange
					peerQueue.Name,
					true, // mandatory
					false, // immediate
					amqp.Publishing{
						ContentType: "text/plain",
						Body:        []byte(offer),
					},
				)
				log.Println("Published message")
				if err != nil {
					panic(err)
				}
			}

			// TODO acknowledge message
			if err := d.Ack(false); err != nil {
				log.Printf("Error acknowledging message : %s", err)
			} else {
				log.Printf("Acknowledged message")
			}
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
