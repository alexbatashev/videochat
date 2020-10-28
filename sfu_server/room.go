package main

import (
	"io"
	"log"
	"sync"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v2"

	"sfu_server/signal"
)

var peerConnectionConfig = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{
		{
			URLs: []string{"stun:stun.l.google.com:19302"},
		},
		{
			URLs:           []string{"turn:turn:3478"},
			Username:       "guest",
			Credential:     "guest",
			CredentialType: webrtc.ICECredentialTypePassword,
		},
	},
	SDPSemantics: webrtc.SDPSemanticsUnifiedPlanWithFallback,
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

func addPeer(roomId string, peerId string, offerStr string) string {
	peer := &Peer{}

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

	log.Println("Add transcievers")
	_, err = peer.connection.AddTransceiver(webrtc.RTPCodecTypeAudio)
	if err != nil {
		panic(err)
	}

	_, err = peer.connection.AddTransceiver(webrtc.RTPCodecTypeVideo)
	if err != nil {
		panic(err)
	}

	log.Println("Set OnTrack callback")
	peer.connection.OnTrack(func(remoteTrack *webrtc.Track, receiver *webrtc.RTPReceiver) {
		if remoteTrack.PayloadType() == webrtc.DefaultPayloadTypeVP8 || remoteTrack.PayloadType() == webrtc.DefaultPayloadTypeVP9 || remoteTrack.PayloadType() == webrtc.DefaultPayloadTypeH264 {
			log.Println("new video track")
			var err error
			peer.videoTrackLock.Lock()
			peer.videoTrack, err = peer.connection.NewTrack(remoteTrack.PayloadType(), remoteTrack.SSRC(), "video", "pion")
			peer.videoTrackLock.Unlock()
			if err != nil {
				panic(err)
			}

			roomsLock.RLock()
			for _, v := range rooms[roomId].peers {
				peer.videoTrackLock.RLock()
				v.connection.AddTrack(peer.videoTrack)
				peer.videoTrackLock.RUnlock()
				v.videoTrackLock.RLock();
				peer.connection.AddTrack(v.videoTrack)
				v.videoTrackLock.RUnlock();
			}
			roomsLock.RUnlock()

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
				i, err := remoteTrack.Read(rtpBuf)
				if err != nil {
					panic(err)
				}
				peer.videoTrackLock.RLock()
				_, err = peer.videoTrack.Write(rtpBuf[:i])
				peer.videoTrackLock.RUnlock()
				if err != io.ErrClosedPipe {
					if err != nil {
						panic(err)
					}
				}
			}

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

	return signal.Encode(*peer.connection.LocalDescription())
}
