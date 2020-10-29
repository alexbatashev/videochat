package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/pion/randutil"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
	"github.com/streadway/amqp"

	"sfu_server/signal"
)

var peerConnectionConfig = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{
		{
			URLs:           []string{"stun:turn:3478"},
			Username:       "guest",
			Credential:     "guest",
			CredentialType: webrtc.ICECredentialTypePassword,
		},
		{
			URLs:           []string{"turn:turn:3478"},
			Username:       "guest",
			Credential:     "guest",
			CredentialType: webrtc.ICECredentialTypePassword,
		},
	},
	SDPSemantics: webrtc.SDPSemanticsUnifiedPlanWithFallback,
	BundlePolicy: webrtc.BundlePolicyMaxCompat,
	// ICECandidatePoolSize: 12,
}

var rooms map[string]*Room
var roomsLock = sync.RWMutex{}

// Media engine
var m webrtc.MediaEngine

// API object
var api *webrtc.API

const (
	rtcpPLIInterval = time.Second * 3
)

func createRoom(id string) {
	roomsLock.Lock()
	rooms[id] = &Room{}
	rooms[id].peers = make(map[string]*Peer)
	roomsLock.Unlock()
}

func addPeer(roomId string, peerId string, offerStr string, conn *amqp.Connection) {

	for {
		roomsLock.RLock()
		if rooms[roomId] == nil {
			roomsLock.RUnlock()
		} else {
			roomsLock.RUnlock()
			break
		}
	}

	peer := &Peer{}
	peer.id = peerId
	peer.connected = false

	// Set up peer queue
	// TODO make it die when unused
	var err error
	peer.peerChannel, err = conn.Channel()
	if err != nil {
		panic(err)
	}
	peerQueue, err := peer.peerChannel.QueueDeclare(
		peerId, // name
		false,  // durable
		false,  // delete when unused
		false,  // exclusive
		false,  // no-wait
		nil,    // arguments
	)
	peer.peerQueue = &peerQueue

	// Parse client offer
	offer := webrtc.SessionDescription{}
	signal.Decode(offerStr, &offer)

	log.Println("Look for codecs")
	videoCodecs := m.GetCodecsByKind(webrtc.RTPCodecTypeVideo)
	if len(videoCodecs) == 0 {
		panic("Offer contained no video codecs")
	}

	// Create new peer connection
	log.Println("Create new peer connection")
	peer.connection, err = api.NewPeerConnection(peerConnectionConfig)
	if err != nil {
		panic(err)
	}

	log.Println("Set OnICECandidate callback")
	peer.connection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			log.Println("Finished gathering candidates")
		} else {
			log.Println("New candidate!")
			candInit := candidate.ToJSON()
			candStr := signal.Encode(candInit)
			log.Println(candStr)

			// TODO refactor queue handling. Queues must be stored with the peer info.
			conn, err := amqp.Dial("amqp://guest:guest@rabbitmq/")
			peerChannel, err := conn.Channel()
			peerQueue, err := peerChannel.QueueDeclare(
				peerId, // name
				false,  // durable
				false,  // delete when unused
				false,  // exclusive
				false,  // no-wait
				nil,    // arguments
			)

			peerMsg := PeerMsg{
				"exchange_ice",
				peerId,
				candStr,
				"",
			}

			jsonMsg, err := json.Marshal(peerMsg)
			if err != nil {
				panic(err)
			}

			err = peerChannel.Publish(
				"", // exchange
				peerQueue.Name,
				true,  // mandatory
				false, // immediate
				amqp.Publishing{
					ContentType: "text/plain",
					Body:        []byte(jsonMsg),
				},
			)
		}
	})

	for i := 0; i < 4; i++ {
		s := strconv.Itoa(i)
		peer.videoTracks[i], err = peer.connection.NewTrack(videoCodecs[0].PayloadType, randutil.NewMathRandomGenerator().Uint32(), "video_"+s, "pion_"+s)
		log.Printf("Creating track #%d", i)
		if err != nil {
			panic(err)
		}
		_, err = peer.connection.AddTrack(peer.videoTracks[i])
		if err != nil {
			panic(err)
		}
	}

	rooms[roomId].peersCountLock.Lock()
	peer.peerNo = rooms[roomId].peersCount
	rooms[roomId].peersCount++
	rooms[roomId].peersCountLock.Unlock()

	log.Println("Set OnTrack callback")
	peer.connection.OnTrack(func(remoteTrack *webrtc.Track, receiver *webrtc.RTPReceiver) {
		if remoteTrack.PayloadType() == webrtc.DefaultPayloadTypeVP8 || remoteTrack.PayloadType() == webrtc.DefaultPayloadTypeVP9 || remoteTrack.PayloadType() == webrtc.DefaultPayloadTypeH264 {
			log.Println("New video track")
			// Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
			go func() {
				ticker := time.NewTicker(rtcpPLIInterval)
				for range ticker.C {
					for i := 0; i < 4; i++ {
						if i != peer.peerNo {
							err := peer.connection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: peer.videoTracks[i].SSRC()}})
							if err != nil {
								panic(err)
							}
						}
					}
				}
			}()
			for peer.connected {
				packet, readErr := remoteTrack.ReadRTP()
				if readErr != nil {
					panic(err)
				}
				roomsLock.RLock()
				rooms[roomId].peersLock.RLock()
				for _, v := range rooms[roomId].peers {
					if v.peerNo != peer.peerNo && v.connected {
						peer.videoTrackLock.RLock()
						v.videoTrackLock.RLock()
						packet.SSRC = v.videoTracks[peer.peerNo].SSRC()
						if writeErr := v.videoTracks[peer.peerNo].WriteRTP(packet); writeErr != nil && !errors.Is(writeErr, io.ErrClosedPipe) {
							panic(writeErr)
						}
						v.videoTrackLock.RUnlock()
						peer.videoTrackLock.RUnlock()
					}
				}
				rooms[roomId].peersLock.RUnlock()
				roomsLock.RUnlock()
			}
		}
	})

	peer.connection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Printf("Connection State has changed %s \n", connectionState.String())
		if connectionState == webrtc.ICEConnectionStateConnected {
			peer.connected = true
		} else {
			peer.connected = false
		}
	})

	log.Println("Set remote description")
	peer.connection.SetRemoteDescription(offer)

	log.Println("Create answer")
	answer, err := peer.connection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}

	// Sets the LocalDescription, and starts our UDP listeners
	log.Println("Set local description")
	err = peer.connection.SetLocalDescription(answer)

	roomsLock.RLock()
	rooms[roomId].peersLock.Lock()
	rooms[roomId].peers[peerId] = peer
	rooms[roomId].peersLock.Unlock()
	roomsLock.RUnlock()

	strAnswer := signal.Encode(answer)
	peerMsg := PeerMsg{
		"exchange_offer",
		peerId,
		strAnswer,
		"answer",
	}

	jsonMsg, err := json.Marshal(peerMsg)
	if err != nil {
		panic(err)
	}

	err = peer.peerChannel.Publish(
		"", // exchange
		peer.peerQueue.Name,
		true,  // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(jsonMsg),
		},
	)
	if err != nil {
		panic(err)
	}
	log.Println("Done")
}

func addICECandidate(roomId string, peerId string, iceStr string) {
	iceCandidate := webrtc.ICECandidateInit{}
	signal.Decode(iceStr, &iceCandidate)
	log.Println("Adding ICE candidate")
	// Wait until room is created
	for {
		roomsLock.RLock()
		if rooms[roomId] == nil {
			roomsLock.RUnlock()
			time.Sleep(100 * time.Millisecond)
		} else {
			roomsLock.RUnlock()
			break
		}
	}
	// Wait until peer is created
	for {
		roomsLock.RLock()
		rooms[roomId].peersLock.RLock()
		if rooms[roomId].peers[peerId] == nil {
			rooms[roomId].peersLock.RUnlock()
			roomsLock.RUnlock()
			time.Sleep(100 * time.Millisecond)
		} else {
			rooms[roomId].peersLock.RUnlock()
			roomsLock.RUnlock()
			break
		}
	}
	roomsLock.RLock()
	rooms[roomId].peers[peerId].connection.AddICECandidate(iceCandidate)
	roomsLock.RUnlock()
	log.Println("ICe candidate was added")
}

func removePeer(roomId string, peerId string) {	
	// Wait for peer to disconnect. Otherwise, someone may write there.
	for {
		roomsLock.RLock()
		rooms[roomId].peersLock.RLock()
		if rooms[roomId].peers[peerId].connected {
			rooms[roomId].peersLock.RUnlock()
			roomsLock.RUnlock()
			time.Sleep(100 * time.Millisecond)
		} else {
			rooms[roomId].peersLock.RUnlock()
			roomsLock.RUnlock()
			break
		}
	}

	roomsLock.RLock()
	rooms[roomId].peersLock.Lock()
	delete(rooms[roomId].peers, peerId)
	rooms[roomId].peersLock.Unlock()
	roomsLock.RUnlock()
}
