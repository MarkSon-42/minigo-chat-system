package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// message transfer wait time for client
	writeWait = 10 * time.Second

	pongWait = 60 * time.Second

	// ping 전송 주기 - pongWait보다 짧아야 함.
	pingPeriod = (pongWait * 9) / 10

	maxMessageSize = 512
)

type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	username string
	room     string
}

// readPump : 읽기 고루틴
//   - websocket에서 메세지 일기 -> json parsing -> hub로 전달
//   - Pong 핸들러: 클라이언트가 살아있는지 확인 .. 각 연결마다 하나의 고루틴에서 실행됨
func (c *Client) readPump() {
	defer func() {
		leaveMessage := NewMessage("leave", c.username, c.room, c.username+" left the room")
		c.hub.broadcast <- leaveMessage

		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		// message parsing
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("json unmarshal error: %v", err)
			continue
		}

		// user info setting
		msg.Username = c.username
		msg.Room = c.room
		msg.Timestamp = time.Now()

		c.hub.broadcast <- &msg
	}
}

// Hub에서 받은 메세지를 WebSocket으로 클라이언트에게 전송.. 웹소켓은 R/W를 동시에 안전하게 할 수 없음 -> 별도의 고루틴으로 분리 필요

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// batch processing : transfer queued messages together
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
