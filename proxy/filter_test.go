package main

import "testing"

func TestCheckMessage_Block(t *testing.T) {
	filter := NewFilter()
	msg := &Message{
		Type:    "message",
		Content: "This contains badword in text",
	}
	allowed, _ := filter.CheckMessage(msg)
	if allowed {
		t.Error("Expected message to be blocked")
	}
}
