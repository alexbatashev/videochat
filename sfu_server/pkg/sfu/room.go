package sfu

type RoomController interface {
	AddRoom(roomId string)
	RemoveRoom(roomId string)
	AddPeer(roomId string, peerId string, q Queue) string
	AddICECandidate(roomId string, peerId string, ice string) error
	RemovePeer(roomId string, peerId string)
	GetPeerQueue(roomId string, peerId string) Queue
}
