# minigo Chat System

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![CI](https://github.com/MarkSon-42/minigo-chat-system/workflows/CI/badge.svg)](https://github.com/MarkSon-42/minigo-chat-system/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A real-time chat system built to learn Go concurrency patterns, WebSocket protocol, and proxy architectures. This is a hands-on project for understanding how to build scalable messaging systems.

> **Note**: This learning project is now complete (January 2026). The core chat system (Backend + Proxy + Frontend) is fully functional. Voice server (WebRTC) remains incomplete.

**[Explore the docs »](https://deepwiki.com/MarkSon-42/minigo-chat-system)**

<!-- TABLE OF CONTENTS -->
<details>
  <summary>Table of Contents</summary>
  <ol>
    <li><a href="#about">About</a></li>
    <li><a href="#architecture">Architecture</a></li>
    <li>
      <a href="#getting-started">Getting Started</a>
      <ul>
        <li><a href="#prerequisites">Prerequisites</a></li>
        <li><a href="#installation">Installation</a></li>
        <li><a href="#running-the-system">Running the System</a></li>
      </ul>
    </li>
    <li><a href="#usage">Usage</a></li>
    <li><a href="#testing">Testing</a></li>
    <li><a href="#roadmap">Roadmap</a></li>
    <li><a href="#contributing">Contributing</a></li>
    <li><a href="#license">License</a></li>
  </ol>
</details>

## About

This project started as a way to understand WebSocket connections and Go's concurrency primitives. Instead of just reading documentation, I wanted to build something real.

Here's why this exists:
* Learning Go's concurrency patterns (channels, goroutines, mutexes) through practical implementation
* Understanding how production chat systems handle filtering, logging, and scaling
* Experimenting with proxy architectures for security and observability

This isn't meant for production use. It's a learning tool, and the code prioritizes clarity over performance.

### Built With

* [Go](https://golang.org/) - 1.21+
* [Gorilla WebSocket](https://github.com/gorilla/websocket) - WebSocket implementation
* [Pion WebRTC](https://github.com/pion/webrtc) - Voice chat support (experimental)

## Architecture

The system uses a 3-tier architecture:

```
Browser ──WebSocket:8081──> Proxy ──WebSocket:8080──> Backend
   ↑                          ↓
   └──────────────────────────┘
```

**Backend** (`:8080`) - Chat server using hub-and-spoke pattern for message broadcasting
**Proxy** (`:8081`) - Security layer with filtering, rate limiting, and logging
**Voice** (`:9000`) - WebRTC SFU for audio channels (work in progress)
**Frontend** - Web client with real-time UI updates

Each service runs independently and communicates via WebSocket.

## Getting Started

### Prerequisites

* Go 1.21 or later
* A modern web browser (Chrome, Firefox, Safari)

### Installation

1. Clone the repo
   ```sh
   git clone https://github.com/MarkSon-42/minigo-chat-system.git
   cd minigo-chat-system
   ```

2. Each service has its own Go module (backend, proxy, voice)
   ```sh
   cd backend && go mod download
   cd ../proxy && go mod download
   ```

### Quick Start

**Start the servers:**
```sh
# Terminal 1
./start_backend.sh

# Terminal 2
./start_proxy.sh

# Open the chat UI
open frontend/index.html
```

Enter a username and room name, then click "입장하기" to start chatting. Open multiple browser windows to test multi-user chat.

### Running the System

You need 2 terminals:

**Terminal 1 - Backend**
```sh
./start_backend.sh
# Or: cd backend && go run *.go
```

**Terminal 2 - Proxy**
```sh
./start_proxy.sh
# Or: cd proxy && go run api.go filter.go main.go message.go proxy.go queue.go storage.go
```

**Terminal 3 - Open the UI**
```sh
open frontend/index.html
```

The chat client will connect to `localhost:8081`, which proxies to the backend at `localhost:8080`.

## Usage

### Basic Chat
1. Open `frontend/index.html` in your browser
2. Enter username and room name
3. Click "입장하기" to join
4. Start chatting!

### Testing Message Filtering
Try typing these words to see filtering in action:
- `password` - blocked entirely
- `badword` - blocked entirely
- `spam` - blocked entirely

### Managing Filter Rules
```sh
# Get current rules
curl http://localhost:8081/filter/rules

# Add a new rule
curl -X POST http://localhost:8081/filter/rules \
  -H "Content-Type: application/json" \
  -d '{
    "keywords": ["test"],
    "action": "block",
    "enabled": true
  }'
```

### Checking Logs
All messages are logged to `logs/messages.jsonl` in JSON Lines format:
```sh
tail -f logs/messages.jsonl
```

## Testing

Run tests with race detection (important for concurrent code):
```sh
# Quick test
./run_tests.sh quick

# Full suite with race detection
./run_tests.sh race

# Coverage report
./run_tests.sh coverage
```

Or run tests manually:
```sh
cd proxy && go test -v -race ./...
cd backend && go test -v -race ./...
```

## Features

### Completed
- [x] Backend WebSocket server with hub pattern
- [x] Proxy layer with filtering and logging
- [x] Web frontend for chat
- [x] CI/CD with GitHub Actions
- [x] Real-time message broadcasting
- [x] Keyword-based message filtering
- [x] JSON Lines message logging

### Incomplete
- [ ] Voice server (WebRTC SFU) - basic structure only
- [ ] Admin dashboard UI
- [ ] Rate limiting
- [ ] Message persistence/history
- [ ] User authentication

## Project Structure

```
minigo-chat-system/
├── backend/           # Chat backend (:8080)
├── proxy/             # Security proxy (:8081)
├── voice/             # Voice server (:9000, WIP)
├── frontend/          # Web client
│   ├── index.html     # Chat UI
│   └── chat.js        # WebSocket client
├── .github/
│   └── workflows/     # CI/CD
└── logs/              # Message audit logs
```

## What I Learned

Building this taught me:
- **Goroutine lifecycle management** - Proper startup/shutdown, avoiding leaks
- **Channel patterns** - Buffering strategies, select statements, timeouts
- **Mutex usage** - RWMutex for read-heavy workloads, deadlock prevention
- **WebSocket gotchas** - Concurrent read/write issues, ping/pong, close handshakes
- **Proxy patterns** - Why Netflix/Slack use them for observability

The code isn't perfect, but each bug I fixed taught me something about concurrent programming.

## Contributing

This is a learning project, but suggestions are welcome!

1. Fork the repo
2. Create a feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## License

Distributed under the MIT License. See `LICENSE` for more information.

## Contact

Mark Son - [@MarkSon-42](https://github.com/MarkSon-42)

Project Link: [https://github.com/MarkSon-42/minigo-chat-system](https://github.com/MarkSon-42/minigo-chat-system)

## Acknowledgments

Resources that helped:
* [Gorilla WebSocket Examples](https://github.com/gorilla/websocket/tree/master/examples/chat)
* [Go Concurrency Patterns](https://go.dev/blog/pipelines)
* [Effective Go](https://go.dev/doc/effective_go)
* [Pion WebRTC Docs](https://github.com/pion/webrtc)
