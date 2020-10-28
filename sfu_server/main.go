package main

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/pion/webrtc/v3"
	"github.com/streadway/amqp"
)

type Peer struct {
	id             string
	connection     *webrtc.PeerConnection
	videoTrackLock sync.RWMutex
	audioTrackLock sync.RWMutex
	videoTracks    [4]*webrtc.Track
	audioTrack     *webrtc.Track
	peerChannel    *amqp.Channel
	peerQueue      *amqp.Queue
	peerNo         int
}
type Room struct {
	peers          map[string]*Peer
	peersLock      sync.RWMutex
	peersCountLock sync.RWMutex
	peersCount     int
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
	Command   string
	PeerId    string
	Data      string
	OfferKind string
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
				addPeer(command.RoomId, command.PeerId, command.Data, conn)
			}

			if command.Command == "exchange_ice" {
				log.Println("Exchanging ICE candidates")

				go addICECandidate(command.RoomId, command.PeerId, command.Data)
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
