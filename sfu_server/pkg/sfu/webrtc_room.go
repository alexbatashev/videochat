package sfu

import (
	"sync"

	"github.com/pion/webrtc/v3"
)

type Peer struct {
	id              string
	connection      *webrtc.PeerConnection
	videoTrackLocks [4]sync.RWMutex
	videoTracks     [4]*webrtc.TrackLocalStaticRTP
	peerQueue       Queue
	peerNo          int
	connected       bool
}

type Room struct {
	peers          map[string]*Peer
	peersLock      sync.RWMutex
	peersCountLock sync.RWMutex
	peersCount     int
}

var rooms map[string]*Room
var roomsLock = sync.RWMutex{}

type PeerCallback func(*Peer)
type TrackCallback func(*webrtc.TrackLocalStaticRTP)

// func initRooms() {
// 	rooms = make(map[string]*Room)
// }

// func AddRoom(RoomID string) {
// 	roomsLock.Lock()
// 	rooms[RoomID] = &Room{}
// 	rooms[RoomID].peers = make(map[string]*Peer)
// 	roomsLock.Unlock()
// }

func (r *Room) AddPeer(PeerID string, q Queue) *Peer {
	peer := &Peer{}
	peer.id = PeerID
	peer.connected = false
	r.peersCountLock.Lock()
	peer.peerNo = r.peersCount
	r.peersCount++
	r.peersCountLock.Unlock()

	peer.peerQueue = q

	r.peersLock.Lock()
	r.peers[PeerID] = peer
	r.peersLock.Unlock()

	return peer
}

func (peer *Peer) AddTrack(id int, trackId, label string) (*webrtc.TrackLocalStaticRTP, error) {
	peer.videoTrackLocks[id].Lock()
	track, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: "video/vp8"}, trackId, label)
	peer.videoTracks[id] = track
	peer.videoTrackLocks[id].Unlock()
	return track, err
}

func (peer *Peer) IsConnected() bool {
	return peer.connected
}

func WithEachPeer(roomId string, fn PeerCallback) {
	roomsLock.RLock()
	room := rooms[roomId]
	roomsLock.RUnlock()
	// Is it thread safe???
	room.peersLock.RLock()
	for _, v := range rooms[roomId].peers {
		// room.peersLock.RUnlock()
		fn(v)
		// room.peersLock.RLock()
	}
	room.peersLock.RUnlock()
}

func (peer *Peer) WithTrack(id int, fn TrackCallback) {
	peer.videoTrackLocks[id].RLock()
	if peer.videoTracks[id] != nil {
		fn(peer.videoTracks[id])
	}
	peer.videoTrackLocks[id].RUnlock()
}
