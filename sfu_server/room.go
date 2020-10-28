package main

import (
	"encoding/json"
	"io"
	"log"
	"sync"
	"time"

	"github.com/pion/randutil"
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
	SDPSemantics:         webrtc.SDPSemanticsUnifiedPlanWithFallback,
	ICECandidatePoolSize: 10,
	BundlePolicy:         webrtc.BundlePolicyMaxCompat,
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
	peer := &Peer{}
	peer.id = peerId

	// Parse client offer
	offer := webrtc.SessionDescription{}
	signal.Decode(offerStr, &offer)

	// TODO is it thread safe/necessary?
	log.Println("Populate from SDP")
	// err := m.PopulateFromSDP(offer)
	//if err != nil {
	//	panic(err)
	// }

	log.Println("Look for codecs")
	var err error
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

	log.Println("Set remote description")
	peer.connection.SetRemoteDescription(offer)

	// log.Println("Add transcievers")
	// _, err = peer.connection.AddTransceiver(webrtc.RTPCodecTypeAudio)
	// if err != nil {
	// 	panic(err)
	// }

	// _, err = peer.connection.AddTransceiver(webrtc.RTPCodecTypeVideo)
	// if err != nil {
	// 	panic(err)
	// }

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
		peer.videoTracks[i], err = peer.connection.NewTrack(webrtc.DefaultPayloadTypeVP8, randutil.NewMathRandomGenerator().Uint32(), "video", "pion")
		if err != nil {
			panic(err)
		}
		peer.connection.AddTrack(peer.videoTracks[i])
	}

	rooms[roomId].peersCountLock.Lock()
	peer.peerNo = rooms[roomId].peersCount
	rooms[roomId].peersCount++
	rooms[roomId].peersCountLock.Unlock()

	log.Println("Set OnTrack callback")
	peer.connection.OnTrack(func(remoteTrack *webrtc.Track, receiver *webrtc.RTPReceiver) {
		if remoteTrack.PayloadType() == webrtc.DefaultPayloadTypeVP8 || remoteTrack.PayloadType() == webrtc.DefaultPayloadTypeVP9 || remoteTrack.PayloadType() == webrtc.DefaultPayloadTypeH264 {
			// Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
			go func() {
				ticker := time.NewTicker(rtcpPLIInterval)
				for range ticker.C {
					err := peer.connection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: peer.videoTrack.SSRC()}})
					if err != nil {
						panic(err)
					}
				}
			}()
			rtpBuf := make([]byte, 1400)
			for {
				log.Printf("Re-sending stream from %s\n", peerId)
				i, err := remoteTrack.Read(rtpBuf)
				if err != nil {
					panic(err)
				}
				roomsLock.RLock()
				rooms[roomId].peersLock.RLock()
				for _, v := range rooms[roomId].peers {
						peer.videoTrackLock.RLock()
						v.videoTrackLock.RLock()
						_, err = v.videoTracks[peer.peerNo].Write(rtpBuf[:i])
						if err != nil {
							panic(err)
						}
						v.videoTrackLock.RUnlock()
						peer.videoTrackLock.RUnlock()
					}
				}
				// peer.videoTrackLock.RLock()
				// _, err = peer.videoTrack.Write(rtpBuf[:i])
				// peer.videoTrackLock.RUnlock()

				if err != io.ErrClosedPipe {
					panic(err)
				}
			// log.Println("new video track")
			// var err error
			// peer.videoTrackLock.Lock()
			// peer.videoTrack, err = peer.connection.NewTrack(remoteTrack.PayloadType(), remoteTrack.SSRC(), "video", "pion")
			// peer.videoTrackLock.Unlock()
			// if err != nil {
			// 	panic(err)
			// }

			// roomsLock.RLock()
			// rooms[roomId].peersLock.RLock()
			// for _, v := range rooms[roomId].peers {
			// 	if v.id != peer.id {
			// 		peer.videoTrackLock.RLock()
			// 		v.connection.AddTrack(peer.videoTrack)
			// 		peer.videoTrackLock.RUnlock()

			// 		// Re-negotiate terms with participant
			// 		newOffer, err := v.connection.CreateOffer(nil)
			// 		if err != nil {
			// 			panic(err)
			// 		}
			// 		err = v.connection.SetLocalDescription(newOffer)
			// 		newOfferStr := signal.Encode(newOffer)
			// 		peerMsg := PeerMsg{
			// 			"exchange_offer",
			// 			v.id,
			// 			newOfferStr,
			// 			"offer",
			// 		}

			// 		jsonMsg, err := json.Marshal(peerMsg)
			// 		if err != nil {
			// 			panic(err)
			// 		}

			// 		err = v.peerChannel.Publish(
			// 			"", // exchange
			// 			v.peerQueue.Name,
			// 			true,  // mandatory
			// 			false, // immediate
			// 			amqp.Publishing{
			// 				ContentType: "text/plain",
			// 				Body:        []byte(jsonMsg),
			// 			},
			// 		)
			// 		if err != nil {
			// 			panic(err)
			// 		}
			// 	}
			// }
			// rooms[roomId].peersLock.RUnlock()
			// roomsLock.RUnlock()

			// // Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
			// go func() {
			// 	ticker := time.NewTicker(rtcpPLIInterval)
			// 	for range ticker.C {
			// 		err := peer.connection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: peer.videoTrack.SSRC()}})
			// 		if err != nil {
			// 			panic(err)
			// 		}
			// 	}
			// }()

			// rtpBuf := make([]byte, 1400)
			// for {
			// 	log.Printf("Re-sending stream from %s\n", peerId)
			// 	i, err := remoteTrack.Read(rtpBuf)
			// 	if err != nil {
			// 		panic(err)
			// 	}
			// 	peer.videoTrackLock.RLock()
			// 	_, err = peer.videoTrack.Write(rtpBuf[:i])
			// 	peer.videoTrackLock.RUnlock()

			// 	if err != io.ErrClosedPipe {
			// 		panic(err)
			// 	}
			// }
		} else {
			var err error
			peer.audioTrackLock.Lock()
			peer.audioTrack, err = peer.connection.NewTrack(remoteTrack.PayloadType(), remoteTrack.SSRC(), "audio", "pion")
			peer.audioTrackLock.Unlock()
			if err != nil {
				panic(err)
			}

			roomsLock.RLock()
			for _, v := range rooms[roomId].peers {
				v.audioTrackLock.RLock()
				v.connection.AddTrack(peer.audioTrack)
				v.audioTrackLock.RUnlock()
			}
			roomsLock.RUnlock()

			rtpBuf := make([]byte, 1400)
			for {
				log.Printf("Reading from %s", peerId)
				i, err := remoteTrack.Read(rtpBuf)
				if err != nil {
					panic(err)
				}
				peer.audioTrackLock.RLock()
				_, err = peer.audioTrack.Write(rtpBuf[:i])
				peer.audioTrackLock.RUnlock()
				if err != io.ErrClosedPipe {
					if err != nil {
						panic(err)
					}
				}
			}
		}
	})

	peer.connection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Printf("Connection State has changed %s \n", connectionState.String())
	})

	log.Println("Create answer")
	answer, err := peer.connection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}

	// Sets the LocalDescription, and starts our UDP listeners
	log.Println("Set local description")
	err = peer.connection.SetLocalDescription(answer)

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
