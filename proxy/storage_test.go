package main

import (
	"bufio"
	"encoding/json"
	"os"
	"testing"
)

func createTempFile(t *testing.T) string {
	f, err := os.CreateTemp("", "storage_test_*.jsonl")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	filepath := f.Name()
	f.Close()
	return filepath
}

func TestLogMessage(t *testing.T) {
	filepath := createTempFile(t)
	defer os.Remove(filepath)

	storage, err := NewStorage(filepath)
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	defer storage.Close()

	msg := &Message{
		Type:     "chat",
		Username: "alice",
		Content:  "Hello World",
	}

	err = storage.LogMessage(msg)
	if err != nil {
		t.Fatalf("LogMessage failed : %v", err)
	}

	storage.Sync()

	file, err := os.Open(filepath)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		t.Fatal("Expected one line in file")
	}

	var logged Message
	err = json.Unmarshal(scanner.Bytes(), &logged)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if logged.Username != "alice" {
		t.Errorf("Expected username 'alice', got '%s'", logged.Username)
	}

	if logged.Content != "Hello World" {
		t.Errorf("Expected content 'Hello World', got '%s'", logged.Content)
	}
}

func TestSetEnabled(t *testing.T) {
	filepath := createTempFile(t)
	defer os.Remove(filepath)

	storage, err := NewStorage(filepath)
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	defer storage.Close()

	msg := &Message{Content: "Should not be logged"}

	storage.SetEnabled(false)
	err = storage.LogMessage(msg)
	if err != nil {
		t.Fatalf("LogMessage failed: %v", err)
	}
	storage.Sync()

	file, err := os.Open(filepath)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		t.Error("File should be empty when logging is disabled")
	}
}

func TestMultipleMessages(t *testing.T) {
	filepath := createTempFile(t)
	defer os.Remove(filepath)

	storage, err := NewStorage(filepath)
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	defer storage.Close()

	messages := []*Message{
		{Username: "alice", Content: "First"},
		{Username: "bob", Content: "Second"},
		{Username: "charlie", Content: "Third"},
	}

	for _, msg := range messages {
		err = storage.LogMessage(msg)
		if err != nil {
			t.Fatalf("LogMessage failed: %v", err)
		}
	}
	storage.Sync()

	file, err := os.Open(filepath)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		count++
	}

	if count != 3 {
		t.Errorf("Expected 3 lines, got %d", count)
	}
}
