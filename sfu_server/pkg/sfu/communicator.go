package sfu

import (
	"encoding/json"
	"log"
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
		log.Print(err)
	}

	q.OnMessage(func(msg []byte) {
		command := &Message{}
		err := json.Unmarshal(msg, command)

		if err != nil {
			log.Print(err)
			return
		}

		log.Printf("New command: %s", command.Command)
		if command.Command == "create_room" {
			ctrl.AddRoom(command.RoomId)
		} else if command.Command == "add_peer" {
			peerQueue, err := qp.CreateQueue(command.PeerId)
			if err != nil {
				return
			}

			offer := ctrl.AddPeer(command.RoomId, command.PeerId, peerQueue)
			peerMsg := PeerMsg{
				"exchange_offer",
				command.PeerId,
				offer,
				"answer",
			}

			jsonMsg, err := json.Marshal(peerMsg)
			err = peerQueue.Write(jsonMsg)
			if err != nil {
				log.Print(err)
				return
			}
		} else if command.Command == "exchange_ice" {
			ctrl.AddICECandidate(command.RoomId, command.PeerId, command.Data)
		} else if command.Command == "remove_peer" {
			ctrl.RemovePeer(command.RoomId, command.PeerId)
		} else if command.Command == "remoove_room" {
			ctrl.RemoveRoom(command.RoomId)
		} else if command.Command == "remote_answer" {
			log.Printf("Accepting remote answer: %s", command.PeerId)
			ctrl.SetRemoteDescription(command.RoomId, command.PeerId, command.Data)
		}
	})
}
