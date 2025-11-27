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

func TestCheckMessage_EmptyContent(t *testing.T) {
	filter := NewFilter()

	testCases := []struct {
		name    string
		content string
	}{
		{"empty string", ""},
		{"only spaces", "   "},
		{"only tabs", "\t\t"},
		{"mixed whitespace", " \t \n"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msg := &Message{
				Type:    "message",
				Content: tc.content,
			}
			allowed, result := filter.CheckMessage(msg)
			if allowed {
				t.Error("Expected empty/whitespace message to be blocked")
			}
			if result != nil {
				t.Error("Expected result to be nil for blocked message")
			}
		})
	}
}
