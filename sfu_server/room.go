package main

import (
	"sync"

	"github.com/pion/webrtc/v3"
	"github.com/streadway/amqp"
)

type Peer struct {
	id              string
	connection      *webrtc.PeerConnection
	videoTrackLocks [4]sync.RWMutex
	videoTracks     [4]*webrtc.Track
	peerChannel     *amqp.Channel
	peerQueue       *amqp.Queue
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
type TrackCallback func(*webrtc.Track)

func initRooms() {
	rooms = make(map[string]*Room)
}

func AddRoom(RoomID string) {
	roomsLock.Lock()
	rooms[RoomID] = &Room{}
	rooms[RoomID].peers = make(map[string]*Peer)
	roomsLock.Unlock()
}

func (r *Room) AddPeer(PeerID string, conn *amqp.Connection) *Peer {
	peer := &Peer{}
	peer.id = PeerID
	peer.connected = false
	r.peersCountLock.Lock()
	peer.peerNo = r.peersCount
	r.peersCount++
	r.peersCountLock.Unlock()

	// Set up peer queue
	// TODO make it die when unused
	var err error
	peer.peerChannel, err = conn.Channel()
	if err != nil {
		panic(err)
	}
	peerQueue, err := peer.peerChannel.QueueDeclare(
		PeerID, // name
		false,  // durable
		false,  // delete when unused
		false,  // exclusive
		false,  // no-wait
		nil,    // arguments
	)
	peer.peerQueue = &peerQueue

	if err != nil {
		panic(err)
	}

	return peer
}

func (peer *Peer) AddTrack(id int, payloadType uint8, ssrc uint32, trackId, label string) (*webrtc.Track, error) {
	peer.videoTrackLocks[id].Lock()
	track, err := peer.connection.NewTrack(payloadType, ssrc, trackId, label)
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
		room.peersLock.RUnlock()
		fn(v)
		room.peersLock.RLock()
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
