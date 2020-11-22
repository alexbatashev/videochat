package sfu

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/alexbatashev/videochat/pkg/signal"
	// "github.com/pion/randutil"
	"github.com/pion/rtcp"
	// "github.com/pion/sdp/v3"
	"github.com/pion/webrtc/v3"
)

type WebRTCRoomController struct {
	MediaEngine webrtc.MediaEngine
	API         *webrtc.API
	Rooms       map[string]*Room
	RoomsLock   sync.RWMutex
}

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
}

const (
	rtcpPLIInterval = time.Second * 3
)

func CreateWebRTCRoomController() WebRTCRoomController {
	controller := WebRTCRoomController{}
	controller.MediaEngine = webrtc.MediaEngine{}
	controller.RoomsLock = sync.RWMutex{}
	controller.Rooms = make(map[string]*Room)

	// controller.MediaEngine.RegisterCodec(webrtc.NewRTPVP8Codec(webrtc.DefaultPayloadTypeVP8, 90000))
	controller.MediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "video/VP8", ClockRate: 90000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
		PayloadType:        96,
	}, webrtc.RTPCodecTypeVideo)
	// controller.MediaEngine.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))

	// sdes, _ := url.Parse(sdp.SDESRTPStreamIDURI)
	// sdedMid, _ := url.Parse(sdp.SDESMidURI)
	// exts := []sdp.ExtMap{
	// 	{
	// 		URI: sdes,
	// 	},
	// 	{
	// 		URI: sdedMid,
	// 	},
	// }

	// se := webrtc.SettingEngine{}
	// se.AddSDPExtensions(webrtc.SDPSectionVideo, exts)

	controller.API = webrtc.NewAPI(webrtc.WithMediaEngine(&controller.MediaEngine))

	return controller
}

func (rc *WebRTCRoomController) AddRoom(roomId string) {
	rc.RoomsLock.Lock()
	if rc.Rooms[roomId] != nil {
		rc.RoomsLock.Unlock()
		return
	}
	rc.Rooms[roomId] = &Room{}
	rc.Rooms[roomId].peers = make(map[string]*Peer)
	rc.RoomsLock.Unlock()
}

func (rc *WebRTCRoomController) RemoveRoom(roomId string) {
	rc.RoomsLock.Lock()
	delete(rc.Rooms, roomId)
	rc.RoomsLock.Unlock()
}

func (rc *WebRTCRoomController) AddPeer(roomId string, peerId string, q Queue) string {
	// Wait for room to be created
	for {
		rc.RoomsLock.RLock()
		if rc.Rooms[roomId] == nil {
			rc.RoomsLock.RUnlock()
		} else {
			rc.RoomsLock.RUnlock()
			break
		}
	}

	rc.RoomsLock.RLock()
	peer := rc.Rooms[roomId].AddPeer(peerId, q)
	rc.RoomsLock.RUnlock()

	var err error

	peer.connection, err = rc.API.NewPeerConnection(peerConnectionConfig)
	if err != nil {
		log.Print(err)
		return ""
	}

	peer.connection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			log.Println("Finished gathering candidates")
		} else {
			log.Println("New candidate!")
			candInit := candidate.ToJSON()
			candStr := signal.Encode(candInit)

			peerMsg := PeerMsg{
				"exchange_ice",
				peerId,
				candStr,
				"",
			}

			jsonMsg, err := json.Marshal(peerMsg)
			if err != nil {
				return
			}

			err = peer.peerQueue.Write(jsonMsg)
			if err != nil {
				log.Print(err)
			}
		}
	})

	for i := 0; i < 4; i++ {
		s := strconv.Itoa(i)
		track, err := peer.AddTrack(i, "video_"+s, "pion_"+s)
		log.Printf("Creating track #%d", i)
		if err != nil {
			panic(err)
		}
		_, err = peer.connection.AddTrack(track)
		if err != nil {
			log.Print(err)
			return ""
		}
	}

	peer.connection.OnTrack(func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Println("New video track")
		// Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
		go func() {
			ticker := time.NewTicker(rtcpPLIInterval)
			for range ticker.C {
				errSend := peer.connection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(remoteTrack.SSRC())}})
				if errSend != nil {
					log.Println(errSend)
				}
			}
		}()
		go func() {
			for peer.IsConnected() {
				// packet, readErr := remoteTrack.ReadRTP()
				rtpBuf := make([]byte, 1400)
				i, readErr := remoteTrack.Read(rtpBuf)
				if readErr != nil {
					// Do not die on errors, just stop the stream
					log.Println(err)
					break
				}
				WithEachPeer(roomId, func(v *Peer) {
					if v.peerNo != peer.peerNo && v.IsConnected() {
						v.WithTrack(peer.peerNo, func(track *webrtc.TrackLocalStaticRTP) {
							// log.Printf("From %d to %d", peer.peerNo, v.peerNo)
							// packet.SSRC = track.SSRC()
							if _, writeErr := track.Write(rtpBuf[:i]); writeErr != nil && !errors.Is(writeErr, io.ErrClosedPipe) {
								return
							}
						})
					}
				})
			}
		}()
	})

	peer.connection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Printf("Connection State has changed %s \n", connectionState.String())
		if connectionState == webrtc.ICEConnectionStateConnected {
			peer.connected = true
		} else {
			peer.connected = false
		}
	})

	offer, err := peer.connection.CreateOffer(nil)
	if err != nil {
		log.Print(err)
		return ""
	}

	if err = peer.connection.SetLocalDescription(offer); err != nil {
		log.Print(err)
		return ""
	}

	offerStr := signal.Encode(offer)

	return offerStr
}

func (rc *WebRTCRoomController) AddICECandidate(roomId string, peerId string, ice string) {
	iceCandidate := webrtc.ICECandidateInit{}
	signal.Decode(ice, &iceCandidate)

	// Wait til room is created
	for {
		rc.RoomsLock.RLock()
		if rc.Rooms[roomId] == nil {
			rc.RoomsLock.RUnlock()
			time.Sleep(100 * time.Millisecond)
		} else {
			rc.RoomsLock.RUnlock()
			break
		}
	}

	// Wait until peer is created
	for {
		rc.RoomsLock.RLock()
		rc.Rooms[roomId].peersLock.RLock()
		if rc.Rooms[roomId].peers[peerId] == nil {
			rc.Rooms[roomId].peersLock.RUnlock()
			rc.RoomsLock.RUnlock()
			time.Sleep(100 * time.Millisecond)
		} else {
			rc.Rooms[roomId].peersLock.RUnlock()
			rc.RoomsLock.RUnlock()
			break
		}
	}
	rc.RoomsLock.RLock()
	rc.Rooms[roomId].peers[peerId].connection.AddICECandidate(iceCandidate)
	rc.RoomsLock.RUnlock()
}

func (rc *WebRTCRoomController) RemovePeer(roomId string, peerId string) {
	// Wait for peer to disconnect. Otherwise, someone may write there.
	for {
		rc.RoomsLock.RLock()
		rc.Rooms[roomId].peersLock.RLock()
		if rc.Rooms[roomId].peers[peerId].connected {
			rc.Rooms[roomId].peersLock.RUnlock()
			rc.RoomsLock.RUnlock()
			time.Sleep(100 * time.Millisecond)
		} else {
			rc.Rooms[roomId].peersLock.RUnlock()
			rc.RoomsLock.RUnlock()
			break
		}
	}

	rc.RoomsLock.RLock()
	rc.Rooms[roomId].peersLock.Lock()
	delete(rc.Rooms[roomId].peers, peerId)
	rc.Rooms[roomId].peersLock.Unlock()
	rc.RoomsLock.RUnlock()
}

func (rc *WebRTCRoomController) SetRemoteDescription(roomId string, peerId string, offer string) {
	// Wait til room is created
	for {
		rc.RoomsLock.RLock()
		if rc.Rooms[roomId] == nil {
			rc.RoomsLock.RUnlock()
			time.Sleep(100 * time.Millisecond)
		} else {
			rc.RoomsLock.RUnlock()
			break
		}
	}

	// Wait until peer is created
	for {
		rc.RoomsLock.RLock()
		rc.Rooms[roomId].peersLock.RLock()
		if rc.Rooms[roomId].peers[peerId] == nil {
			rc.Rooms[roomId].peersLock.RUnlock()
			rc.RoomsLock.RUnlock()
			time.Sleep(100 * time.Millisecond)
		} else {
			rc.Rooms[roomId].peersLock.RUnlock()
			rc.RoomsLock.RUnlock()
			break
		}
	}

	rc.RoomsLock.RLock()
	rc.Rooms[roomId].peersLock.RLock()
	sdp := webrtc.SessionDescription{}
	signal.Decode(offer, &sdp)
	rc.Rooms[roomId].peers[peerId].connection.SetRemoteDescription(sdp)
	rc.Rooms[roomId].peersLock.RUnlock()
	rc.RoomsLock.RUnlock()
}
