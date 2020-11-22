package sfu

type RoomController interface {
	AddRoom(roomId string)
	RemoveRoom(roomId string)
	AddPeer(roomId string, peerId string, q Queue) string
	AddICECandidate(roomId string, peerId string, ice string)
	RemovePeer(roomId string, peerId string)
	SetRemoteDescription(roomId string, peerId string, offer string)
}
