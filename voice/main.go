package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // CORS 허용..했는데 프로덕션에선 제한 해야함.
		},
	}

	peersMutex sync.RWMutex
	peers      = make(map[string]*webrtc.PeerConnection) // username -< PeerConnection
)

func main() {
	log.Println("Voice SFU Server starting...")

	http.HandleFunc("/ws", handleWebSocket)

	if err := http.ListenAndServe(":9000", nil); err != nil {
		log.Fatal("Server failed:", err)
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade failed:", err)
		return
	}
	
}
