package main

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"
)

type Storage struct {
	mu       sync.Mutex
	file     *os.File
	filepath string
	enabled  bool
}

// NewStorage creates a new storage instance
func NewStorage(filepath string) (*Storage, error) {
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	log.Printf("[Storage] Initialized: %s", filepath)

	return &Storage{
		file:     file,
		filepath: filepath,
		enabled:  true,
	}, nil
}

func (s *Storage) LogMessage(msg *Message) error {
	if !s.enabled {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	_, err = s.file.Write(data)
	if err != nil {
		log.Printf("[Storage] Write failed: %v", err)
		return err
	}

	return nil
}

func (s *Storage) SetEnabled(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enabled = enabled
	log.Printf("[Storage] Enabled: %v", enabled)
}

func (s *Storage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.file != nil {
		log.Printf("[Storage] Closing %s", s.filepath)
		return s.file.Close()
	}
	return nil
}

func (s *Storage) Sync() error {
	s.mu.Lock()
	defer s.mu.Lock()

	if s.file != nil {
		return s.file.Sync()
	}
	return nil
}
