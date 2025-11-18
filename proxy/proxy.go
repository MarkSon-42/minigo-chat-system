// proxy.go : 1. 양방향 중계 client <---> proxy <---> backend server |  2. filterling : client -> backend 방향만 필터링  |  3. queueing & error processing

package main

import (
	"net/url"

	"github.com/gorilla/websocket"
)

type Proxy struct {
	clientConn  *websocket.Conn
	backendConn *websocket.Conn
	filter      *Filter
	queue       *MessageQueue
}

// NewProxy generator

func NewProxy(clientConn *websocket.Conn, filter *Filter, queue *MessageQueue, username, room string) {
	backendURL, err := url.Parse(*backendAddr)
	if err != nil {
		return nil, err
	}
}

func (p *Proxy) Start() {

}

func (p *Proxy) clientToBackend(done chan struct{}) {

}

func (p *Proxy) backendToClient(done chan struct{}) {

}

func (p *Proxy) Close() {

}
