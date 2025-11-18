package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var (
	listenAddr  = flag.String("listen", ":8081", "Proxy listen address")
	backendAddr = flag.String("backend", "ws://localhost:8080/ws", "Backend WebSocket address")

	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

type ProxyServer struct {
	filter *Filter
	queue  *MessageQueue
}

func NewProxyServer() *ProxyServer {
	return &ProxyServer{
		filter: NewFilter(),
		queue:  NewMessageQueue(1000),
	}
}

func (ps *ProxyServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	room := r.URL.Query().Get("room")

	if username == "" || room == "" {
		http.Error(w, "username and room are required", http.StatusBadRequest)
		return
	}

	clientConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[Proxy] Failed to upgrade client connection: %v", err)
		return
	}

	log.Printf("[Proxy] New client connected: %s (room: %s)", username, room)

	proxy, err := NewProxy(clientConn, ps.filter, ps.queue, username, room)
	if err != nil {
		log.Printf("[Proxy] Failed to create proxy %v", err)
		clientConn.Close()
		return
	}
	proxy.Start()
}

func (ps *ProxyServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (ps *ProxyServer) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"running","queue_size":0}`))
}

func main() {
	flag.Parse()

	log.Println("=== Chat System Proxy Server ===")
	log.Printf("Listen address: %s", *listenAddr)
	log.Printf("Backend address: %s", *backendAddr)

	proxy := NewProxyServer()

	http.HandleFunc("/ws", proxy.handleWebSocket)
	http.HandleFunc("/health", proxy.handleHealth)
	http.HandleFunc("/stats", proxy.handleStats)
	http.HandleFunc("/filter/rules", proxy.handleFilterRules)

	log.Printf("[Proxy] Starting server on %s", *listenAddr)
	log.Printf("[Proxy] Endpoints:")
	log.Println("  - WebSocket: /ws?username=<name>&room=<room>")
	log.Println("  - Health: /health")
	log.Println("  - Stats: /stats")

	if err := http.ListenAndServe(*listenAddr, nil); err != nil {
		log.Fatalf("[Proxy] Server failed: %v", err)
	}
}
