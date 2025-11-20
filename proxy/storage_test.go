package main

import (
	"os"
	"testing"
	"time"
)

// test storage creation
func TestStorageCreation(t *testing.T) {
	tmpfile := "test_storage_creation.jsonl"
	defer os.Remove(tmpfile)

	storage, err := NewStorage(tmpfile)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	if storage.filepath != tmpfile {
		t.Errorf("Expected filepath %s, got %s", tmpfile, storage.filepath)
	}

	if !storage.enabled {
		t.Error("Storage should be enabled by default")
	}

	if _, err := os.Stat(tmpfile); os.IsNotExist(err) {
		t.Error("Storage file was not created")
	}
}

func TestStorageSync(t *testing.T) {
	tmpfile := "test_sync_deadlock.jsonl"
	defer os.Remove(tmpfile)

	storage, err := NewStorage(tmpfile)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	done := make(chan error, 1)
	go func() {
		done <- storage.Sync()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Sync() failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("DEADLOCK DETECTED: Sync() did not complete 2 seconds")
	}
}
