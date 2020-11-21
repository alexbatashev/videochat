package sfu

import (
	"net/url"
	"sync"

	"github.com/pion/sdp/v3"
	"github.com/pion/webrtc/v3"
)

type WebRTCRoomController struct {
	MediaEngine webrtc.MediaEngine
	API *webrtc.API
	Rooms map[string]*Room
	RoomsLock sync.RWMutex
}

func CreateWebRTCRoomController() WebRTCRoomController {
	controller := WebRTCRoomController{}
	controller.MediaEngine = webrtc.MediaEngine{}
	controller.RoomsLock = sync.RWMutex{}
	controller.Rooms = make(map[string]*Room)

	controller.MediaEngine.RegisterCodec(webrtc.NewRTPVP8Codec(webrtc.DefaultPayloadTypeVP8, 90000))
	controller.MediaEngine.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))

	sdes, _ := url.Parse(sdp.SDESRTPStreamIDURI)
	sdedMid, _ := url.Parse(sdp.SDESMidURI)
	exts := []sdp.ExtMap{
		{
			URI: sdes,
		},
		{
			URI: sdedMid,
		},
	}

	se := webrtc.SettingEngine{}
	se.AddSDPExtensions(webrtc.SDPSectionVideo, exts)

	controller.API = webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithSettingEngine((se)))

	return controller
}
