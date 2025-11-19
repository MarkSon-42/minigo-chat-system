// proxy.go : 1. 양방향 중계 client <---> proxy <---> backend server |  2. filterling : client -> backend 방향만 필터링  |  3. queueing & error processing

package main

import (
	"encoding/json"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

type Proxy struct {
	clientConn  *websocket.Conn
	backendConn *websocket.Conn
	filter      *Filter       // filter.go의 Filter 공유
	queue       *MessageQueue // queue.go
	storage     *Storage
}

// NewProxy generator - 그럼 이건 생성자 함수? 파라미터로 클라이언트연결, 필터, 큐, 사용자이름, 채팅방이름.. 다시 () 에는 반환값.

func NewProxy(clientConn *websocket.Conn, filter *Filter, queue *MessageQueue, storage *Storage, username, room string) (*Proxy, error) {
	backendURL, err := url.Parse(*backendAddr) // Parse() : 문자열을 URL 구조체로 변환
	if err != nil {
		return nil, err
	}

	query := backendURL.Query()
	query.Set("username", username)
	query.Set("room", room)
	backendURL.RawQuery = query.Encode()

	backendConn, _, err := websocket.DefaultDialer.Dial(backendURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return &Proxy{
		clientConn:  clientConn,
		backendConn: backendConn,
		filter:      filter,
		queue:       queue,
		storage:     storage,
	}, nil
}

func (p *Proxy) Start() {
	done := make(chan struct{})

	go p.clientToBackend(done)
	go p.backendToClient(done)

	<-done // blocking .. 채널에서 값을 받을 때까지 대기
	p.Close()
}

func (p *Proxy) clientToBackend(done chan struct{}) {
	defer close(done)

	for {
		_, data, err := p.clientConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[Proxy] Client read error: %v", err)
			}
			return
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("[Proxy] Invalid message format: %v", err)
			continue
		}

		allowed, filteredMsg := p.filter.CheckMessage(&msg)
		if !allowed {
			log.Printf("[Proxy] Message blocked from %s: %s", msg.Username, msg.Content)
			continue
		}
		if filteredMsg != nil {
			msg = *filteredMsg
		}

		if !p.queue.Enqueue(&msg) {
			log.Printf("[Proxy] Queue full, message dropped from %s", msg.Username)
			continue
		}

		if p.storage != nil {
			if err := p.storage.LogMessage(&msg); err != nil {
				log.Printf("[Proxy] Failed to log message: %v", err)
			}
		}

		filteredData, err := json.Marshal(msg)
		if err != nil {
			log.Printf("[Proxy] Failed to marshal message: %v", err)
			continue
		}

		if err := p.backendConn.WriteMessage(websocket.TextMessage, filteredData); err != nil {
			log.Printf("[Proxy] Failed to send to backend: %v", err)
			return
		}

	}
}

func (p *Proxy) backendToClient(done chan struct{}) {
	defer close(done)

	p.backendConn.SetReadDeadline(time.Now().Add(60 * time.Second))
	p.clientConn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	p.backendConn.SetPongHandler(func(string) error {
		p.backendConn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, data, err := p.backendConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[Proxy] Backend read error: %v", err)
			}
			return
		}
		if err := p.clientConn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("[Proxy] Failed to send to client %v", err)
			return
		}
	}
}

func (p *Proxy) Close() {
	if p.clientConn != nil {
		p.clientConn.Close()
	}
	if p.backendConn != nil {
		p.backendConn.Close()
	}
}
