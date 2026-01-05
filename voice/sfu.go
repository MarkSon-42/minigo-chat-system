package main

import "sync"

type SFU struct {
	mu    sync.RWMutex
	peers map[string]*Peer // username -> Peer
}
