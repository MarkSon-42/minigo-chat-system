# Chat System Project

## Project Overview
Real-time chat system with WebSocket proxy and message filtering.
This is a **learning project** focused on understanding Go concurrency, WebSocket protocols, and proxy architectures.

## Go Development Rules
- ALWAYS check if `go.mod` exists before using `import` statements
- If seeing "undefined" errors, run: `go mod init chat-backend && go get <package>`
- Test with `go test ./...` before committing
- Format code with `go fmt ./...` after editing
- Use `go mod tidy` to clean up dependencies

## WebSocket Implementation Guidelines
- Follow Gorilla WebSocket library best practices
- ALWAYS implement proper connection cleanup with `defer`
- Use separate `readPump`/`writePump` goroutines for each client
- Set read/write deadlines for timeout protection (prevent zombie connections)
- Use channels for goroutine communication (never share memory directly)

## Code Structure
```
backend/  - WebSocket chat server (port 8080)
  ├── main.go     - Server entry point
  ├── hub.go      - Chat room hub (manages clients)
  ├── client.go   - WebSocket client management
  └── message.go  - Message data structures

proxy/    - Message filtering proxy (port 8081)
  ├── main.go     - Proxy entry point
  ├── proxy.go    - Bidirectional relay logic
  ├── filter.go   - Message filtering engine
  ├── queue.go    - Message queue implementation
  └── api.go      - HTTP API for filter management

frontend/ - Vanilla JavaScript client
  ├── index.html  - Chat UI
  ├── admin.html  - Admin dashboard
  ├── chat.js     - WebSocket client logic
  └── admin.js    - Admin API client
```

## Learning Mode Rules (IMPORTANT!)
- ALWAYS explain WHY before implementing code
- Break down complex functions line-by-line when asked
- Use Korean comments in code for explanations when requested
- NEVER assume user knows Go syntax:
  - Explain struct tags (e.g., `json:"fieldname"`)
  - Explain channels and goroutines with analogies
  - Explain pointers vs values
- Provide real-world analogies for concurrency concepts
- When user asks "what is X?", give concept + syntax + example + analogy

## Common Issues & Solutions
- **"undefined: websocket"** → Run: `go get github.com/gorilla/websocket`
- **"package X is not in GOROOT"** → Run: `go mod init <module-name>` first
- **Race conditions** → Use channels, not shared variables
- **Memory leaks** → Always close connections in defer blocks
- **Deadlocks** → Never send/receive on same goroutine without buffering

## Architecture Flow
```
[Browser Client]
    ↓ WebSocket
[Proxy Server :8081] ← HTTP API (filter rules)
    ↓ Message Queue (Go channels)
[Backend Server :8080]
    ↓ Hub (broadcast)
[In-Memory Storage]
```

## Testing Strategy
- Test multi-client scenarios (at least 3 concurrent connections)
- Verify message filtering works end-to-end
- Check for memory leaks with long-running connections
- Test proxy failover behavior
- Verify rate limiting functionality

## Implementation Phases
1. **Backend Foundation** (1 hour)
   - hub.go: Client management and broadcasting
   - client.go: WebSocket read/write pumps
   - message.go: Data structures

2. **Proxy & Filtering** (1 hour)
   - Bidirectional WebSocket relay
   - Message queue implementation
   - Filtering engine + HTTP API

3. **Frontend** (40 min)
   - Chat client UI
   - Admin dashboard

4. **Integration Testing** (20 min)
   - Multi-client testing
   - Filter verification

## Never Do This
- ❌ NEVER use `go run` without `go mod init` first
- ❌ NEVER create goroutines without handling cleanup
- ❌ NEVER ignore WebSocket close errors
- ❌ NEVER use shared variables across goroutines (use channels instead)

## Instead Do This
- ✅ Initialize module: `go mod init <name>` before coding
- ✅ Always use `defer` for cleanup (connections, files, etc.)
- ✅ Check and log WebSocket errors properly
- ✅ Use channels for communication: `ch := make(chan Type, bufferSize)`

## Git Commit Convention
- Follow Conventional Commits format: `<type>(<scope>): <subject>`
- Types: feat, fix, docs, style, refactor, test, chore, perf
- Scopes: backend, proxy, frontend, api, filter, docs, ci
- **IMPORTANT: DO NOT include AI-generated markers (e.g., "Generated with Claude")**
- Keep commit messages clean and professional
- Example: `feat(proxy): implement message filtering engine`

## Reference Documentation
- For complex Gorilla WebSocket usage patterns, see: https://github.com/gorilla/websocket/tree/main/examples
- For Go concurrency patterns, refer to: https://go.dev/blog/pipelines
- If stuck on channel deadlocks, check buffering and close() calls
