package main

import (
	"log"
	"os"
	"sync"
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
