package main

import "github.com/gorilla/websocket"

type Proxy struct {
	clientConn  *websocket.Conn
	backendConn *websocket.Conn
	filter      *Filter
	queue       *MessageQueue
}
