package main

import "github.com/pion/webrtc/v4"

type Peer struct {
	username string
	pc       *webrtc.PeerConnection
}
