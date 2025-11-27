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

func TestCheckMessage_PasswordBlock(t *testing.T) {
	filter := NewFilter()
	msg := &Message{
		Type:    "message",
		Content: "My password is 12345",
	}
	allowed, result := filter.CheckMessage(msg)
	if !allowed {
		t.Error("Expected message to be blocked")
	}
	if result == nil {
		t.Fatal("Expected result to be nil when blocked")
	}
}
