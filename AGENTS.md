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

## Effective Learning Strategies

### 1. Debugging Mode (Most Effective!)
**실행 흐름을 직접 확인하며 학습하는 것이 가장 효과적**

#### VSCode 디버거 활용
```bash
# 1. VSCode에서 F5 또는 "Run and Debug" 클릭
# 2. 브레이크포인트 설정 (코드 라인 왼쪽 클릭)
# 3. 단계별 실행:
#    - F10: Step Over (함수 호출을 한 번에)
#    - F11: Step Into (함수 내부로 들어가기)
#    - Shift+F11: Step Out (함수에서 나가기)
```

#### 확인할 사항들
- 변수 값이 어떻게 변하는지
- 함수 호출 스택 (Call Stack)
- 고루틴이 언제 생성되고 종료되는지
- 채널에 데이터가 언제 전송/수신되는지

### 2. 작은 단위로 실험하기
**헷갈리는 개념을 별도 파일로 테스트**

```go
// test_receiver.go 예제
package main

import "fmt"

type Counter struct {
    count int
}

// 값 리시버 (복사본 사용)
func (c Counter) IncrementValue() {
    c.count++  // 원본은 변경 안 됨
    fmt.Printf("Inside IncrementValue: %d\n", c.count)
}

// 포인터 리시버 (원본 사용)
func (c *Counter) IncrementPointer() {
    c.count++  // 원본이 변경됨
    fmt.Printf("Inside IncrementPointer: %d\n", c.count)
}

func main() {
    counter := Counter{count: 0}

    counter.IncrementValue()     // 브레이크포인트 1
    fmt.Printf("After Value: %d\n", counter.count)  // 0 (변경 안 됨)

    counter.IncrementPointer()   // 브레이크포인트 2
    fmt.Printf("After Pointer: %d\n", counter.count) // 1 (변경됨!)
}
```

실행 방법:
```bash
cd /tmp
go mod init test
# 위 코드를 test_receiver.go로 저장
go run test_receiver.go
```

### 3. 로깅으로 흐름 추적
**디버거가 불편하면 로그로 흐름 파악**

```go
func (ps *ProxyServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
    log.Printf("[DEBUG] 1. Entering handleWebSocket")

    username := r.URL.Query().Get("username")
    log.Printf("[DEBUG] 2. username=%s", username)

    clientConn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Printf("[DEBUG] 3. Upgrade failed: %v", err)
        return
    }
    log.Printf("[DEBUG] 4. Upgrade success, clientConn=%p", clientConn)

    defer func() {
        log.Printf("[DEBUG] 5. Closing connection")
        clientConn.Close()
    }()
}
```

실행 후 로그 순서 확인:
```
[DEBUG] 1. Entering handleWebSocket
[DEBUG] 2. username=mark
[DEBUG] 4. Upgrade success, clientConn=0xc0001a2000
[DEBUG] 5. Closing connection
```

### 4. 공식 문서와 병행 학습
**이론 + 실습 조합**

| 학습 자료 | 용도 | 링크 |
|----------|------|------|
| Go Tour | 대화형 기초 문법 | https://go.dev/tour/ |
| Go by Example | 실용적인 예제 모음 | https://gobyexample.com/ |
| Effective Go | Go 관용구와 패턴 | https://go.dev/doc/effective_go |
| Go Blog | 고급 주제 (채널, 동시성) | https://go.dev/blog/pipelines |

**추천 학습 순서:**
1. 개념이 헷갈림 → Go Tour에서 해당 챕터 학습
2. 예제 코드 작성 → Go by Example 참고
3. 디버거로 실행 → 변수/흐름 확인
4. 실제 프로젝트에 적용 → 이 프로젝트에서 실습

### 5. 질문하기 좋은 형식
**막혔을 때 효과적으로 질문하는 법**

❌ 나쁜 질문:
```
"리시버가 뭐에요?"
```

✅ 좋은 질문:
```
"리시버를 사용하는 이유를 알겠는데,
왜 (ps *ProxyServer)처럼 포인터로 받는 건가요?
값으로 받으면 안 되나요?"
```

❌ 나쁜 질문:
```
"에러가 나요."
```

✅ 좋은 질문:
```
"upgrader.Upgrade() 실행 시
'websocket: request origin not allowed' 에러가 나는데,
CheckOrigin 함수는 true를 반환하도록 했습니다.
왜 이런 에러가 나는 걸까요?"
```

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
