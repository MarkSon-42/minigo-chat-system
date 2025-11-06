package main

import (
	"encoding/json"
	"log"
)

type Hub struct {
	clients map[*Client]bool

	broadcast chan *Message

	register chan *Client

	unregister chan *Client
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan *Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}

		case message := <-h.broadcast:
			jsonMessage, err := json.Marshal(message)
			if err != nil {
				log.Printf("json marshal error: %v", err)
				continue
			}

			for client := range h.clients {
				if client.room != message.Room {
					continue
				}

				select {
				case client.send <- jsonMessage:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}
