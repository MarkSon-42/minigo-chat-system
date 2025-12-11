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

	storage.Sync() // > how to work...?

	file, err := os.Open(filepath)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close() // why use defer to here?

	scanner := bufio.NewScanner(file) // newscanner() < need to explain ( how it works, what is it )
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

}
