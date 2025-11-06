package main

import "time"

type Message struct {
	Type      string    `json:"type"` // join, leave, message
	Username  string    `json:"username"`
	Room      string    `json:"room"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}
