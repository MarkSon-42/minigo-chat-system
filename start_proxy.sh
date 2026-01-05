#!/bin/bash
cd "$(dirname "$0")/proxy"
go run api.go filter.go main.go message.go proxy.go queue.go storage.go
