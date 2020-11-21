package sfu

import (
	"encoding/json"
	"fmt"
)

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

func StartServer(qp QueueProvider, name string, ctrl RoomController) {
	q, err := qp.CreateQueue(name)
	if err != nil {
		fmt.Panic(err)
	}

	q.OnMessage(func (msg string) {
		command := &Message{}
		err := json.Unmarshall(msg, command)

		if err != nil {
			fmt.Errorf(err)
			return
		}

		if command.Command == "create_room" {
			ctrl.AddRoom(command.RoomId)
		} else if command.Command == "add_peer" {
			peerQueue, err := qp.CreateQueue(command.PeerId)
			if err != nil {
				return
			}

			offer := ctrl.AddPeer(command.RoomId, command.PeerId, peerQueue)
			err = peerQueue.Write(offer)
			if err != nil {
				fmt.Errorf(err)
				return
			}
		} else if command.Command == "exchange_ice" {
			err := ctrl.AddICECandidate(command.RoomId, command.PeerId, command.Data)
			if err != nil {
				return
			}
		} else if command.Command == "remove_peer" {
			err := ctrl.RemovePeer(command.RoomId, command.PeerId)
		} else if command.Command == "remoove_room" {
			err := ctrl.RemoveRoom(command.RoomId)
		}
	})
}
