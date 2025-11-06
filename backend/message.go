package main

import "time"

type Message struct {
	Type      string    `json:"type"` // join, leave, message
	Username  string    `json:"username"`
	Room      string    `json:"room"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// helper function for new message creation
func NewMessage(msgType, username, room, content string) *Message {
	return &Message{
		Type:      msgType,
		Username:  username,
		Room:      room,
		Content:   content,
		Timestamp: time.Now(),
	}
}
