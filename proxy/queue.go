package main

import (
	"log"
	"time"
)

type MessageQueue struct {
	messages chan *Message
	capacity int
}

func NewMessageQueue(capacity int) *MessageQueue {
	return &MessageQueue{
		messages: make(chan *Message, capacity),
		capacity: capacity,
	}
}

func (q *MessageQueue) Enqueue(msg *Message) bool {
	select {
	case q.messages <- msg:
		return true
	case <-time.After(100 * time.Millisecond):
		log.Printf("[Queue] Full, dropping message from %s", msg.Username)
		return false
	}
}

func (q *MessageQueue) Dequeue() (*Message, bool) {
	select {
	case msg := <-q.messages:
		return msg, true
	case <-time.After(1 * time.Second):
		return nil, false
	}
}

func (q *MessageQueue) Size() int {
	return len(q.messages)
}
