package main

import (
	"testing"
)

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
func TestCheckMessage_Repalce(t *testing.T) {

}

func TestCheckMessage_EmptyContent(t *testing.T) {

}

func TestAddRule(t *testing.T) {

}

func TestRemoveRule(t *testing.T) {

}

func TestUpdateRule(t *testing.T) {

}

func TestConcurrentAccess(t *testing.T) {
	filter := NewFilter()
	done := make(chan bool)

	// concurrent read
	for i := 0; i < 10; i++ {
		go func() {
			msg := &Message{Type: "message", Content: "test"}
			filter.CheckMessage(msg)
			done <- true
		}()
	}

	// concurrent write
}
