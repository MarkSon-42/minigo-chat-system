# Go Development Guidelines

> Go 개발 베스트 프랙티스 및 minigo 채팅 시스템 개발 가이드

## When This Skill Activates

이 스킬은 다음 상황에서 자동 활성화됩니다:
- `*.go` 파일 수정할 때
- `go.mod`, `go.sum` 파일 작업할 때
- Go 관련 키워드 사용 시 (goroutine, channel, mutex, defer, race condition 등)
- `/go-help`, `/concurrency` 명령어 입력 시

## Quick Reference Card

### 필수 명령어
```bash
# 모듈 초기화 (처음 한 번만)
go mod init <module-name>

# 의존성 설치
go get <package>

# 빌드
go build

# 테스트 (race detector 포함)
go test -race ./...

# 코드 포맷팅
go fmt ./...

# 의존성 정리
go mod tidy
```

### 체크리스트
프로젝트 시작 시:
- [ ] `go.mod` 파일 존재 확인
- [ ] import 전에 `go get` 실행

코드 작성 시:
- [ ] 고루틴 생성 시 cleanup 계획
- [ ] 채널 사용 시 close() 위치 확인
- [ ] defer로 리소스 정리 (Close, Unlock)
- [ ] 에러 체크 (`if err != nil`)

커밋 전:
- [ ] `go test -race ./...` 실행
- [ ] `go fmt ./...` 실행
- [ ] `go mod tidy` 실행

---

## Core Principles

### 1. 모듈 관리 (Module Management)

**규칙: ALWAYS check if `go.mod` exists before using `import`**

```bash
# "undefined" 에러 발생 시
cd backend
go mod init chat-backend
go get github.com/gorilla/websocket
```

**Minigo 프로젝트 모듈 구조:**
```
backend/go.mod  → module: chat-backend
proxy/go.mod    → module: chat-system
voice/go.mod    → module: voice
```

각 모듈은 독립적으로 관리됩니다. 빌드나 테스트도 각 디렉토리에서 별도로 실행해야 합니다.

### 2. 동시성 (Concurrency)

**핵심 원칙: Use channels for communication, not shared memory**

❌ **잘못된 예 - 공유 변수:**
```go
var counter int  // 여러 고루틴이 동시 접근 → Race!

go func() {
    counter++  // ⚠️ Race condition
}()
```

✅ **올바른 예 - 채널 사용:**
```go
ch := make(chan int, 1)

go func() {
    ch <- 1  // 안전한 전송
}()

value := <-ch  // 안전한 수신
```

✅ **올바른 예 - Mutex 사용:**
```go
var (
    counter int
    mu      sync.Mutex
)

go func() {
    mu.Lock()
    counter++
    mu.Unlock()
}()
```

**상세 패턴은:** [resources/concurrency-patterns.md](./resources/concurrency-patterns.md) 참조

### 3. 에러 핸들링 (Error Handling)

**규칙: ALWAYS check errors immediately**

❌ **잘못된 예:**
```go
conn, _ := upgrader.Upgrade(w, r, nil)  // 에러 무시!
```

✅ **올바른 예:**
```go
conn, err := upgrader.Upgrade(w, r, nil)
if err != nil {
    log.Printf("Upgrade failed: %v", err)
    return
}
```

**에러 래핑 (Go 1.13+):**
```go
if err != nil {
    return fmt.Errorf("failed to connect to backend: %w", err)
}
```

### 4. 리소스 관리 (Resource Management)

**규칙: ALWAYS use `defer` for cleanup**

✅ **패턴:**
```go
func handleConnection(conn *websocket.Conn) {
    // 함수가 끝나면 반드시 실행됨
    defer func() {
        log.Printf("Closing connection")
        conn.Close()
    }()

    // 연결 처리 로직...
}
```

**Defer 순서 (LIFO - Last In First Out):**
```go
func example() {
    defer fmt.Println("3")  // 마지막 실행
    defer fmt.Println("2")
    defer fmt.Println("1")  // 첫 번째 실행
}
// 출력: 1, 2, 3
```

### 5. 고루틴 생명주기 (Goroutine Lifecycle)

**규칙: NEVER create goroutines without handling cleanup**

❌ **위험한 예 - 고루틴 누수:**
```go
func handleRequest() {
    go func() {
        for {
            // 무한 루프, 종료 조건 없음!
            doWork()
        }
    }()
}
```

✅ **안전한 예 - Context로 종료 제어:**
```go
func handleRequest(ctx context.Context) {
    go func() {
        for {
            select {
            case <-ctx.Done():
                log.Println("Goroutine shutting down")
                return
            default:
                doWork()
            }
        }
    }()
}
```

✅ **안전한 예 - Done 채널:**
```go
func handleRequest(done chan struct{}) {
    go func() {
        defer log.Println("Goroutine exiting")
        for {
            select {
            case <-done:
                return
            default:
                doWork()
            }
        }
    }()
}
```

---

## WebSocket Development

### Gorilla WebSocket 베스트 프랙티스

**1. Upgrader 설정**
```go
var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        // 개발 환경: 모든 origin 허용
        return true
        // 프로덕션: origin 체크 필수!
        // origin := r.Header.Get("Origin")
        // return origin == "https://yourdomain.com"
    },
}
```

**2. 연결 정리 (Cleanup)**
```go
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Printf("Upgrade error: %v", err)
        return
    }

    // ⭐ 필수: defer로 정리
    defer conn.Close()

    // WebSocket 로직...
}
```

**3. readPump/writePump 패턴**

**이 프로젝트의 핵심 패턴입니다!**

```go
// 각 클라이언트마다 2개의 고루틴 실행
go client.readPump()   // WebSocket에서 읽기만
go client.writePump()  // WebSocket에 쓰기만
```

**왜 분리하나요?**
- WebSocket은 동시 읽기/쓰기를 지원하지 않음
- 동시 접근 시 데드락이나 race condition 발생
- 분리하면 안전하고 독립적인 타임아웃 관리 가능

**상세 구현:** [websocket-webrtc-patterns](../websocket-webrtc-patterns/skill.md) 스킬 참조

**4. Timeout 설정 (좀비 연결 방지)**
```go
const (
    pongWait   = 60 * time.Second
    pingPeriod = (pongWait * 9) / 10  // pongWait보다 짧게
    writeWait  = 10 * time.Second
)

// readPump에서
c.conn.SetReadDeadline(time.Now().Add(pongWait))
c.conn.SetPongHandler(func(string) error {
    c.conn.SetReadDeadline(time.Now().Add(pongWait))
    return nil
})

// writePump에서
ticker := time.NewTicker(pingPeriod)
defer ticker.Stop()

for {
    select {
    case <-ticker.C:
        c.conn.SetWriteDeadline(time.Now().Add(writeWait))
        if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
            return
        }
    }
}
```

---

## Testing Strategy

### 1. 단위 테스트 (Unit Tests)

**파일 네이밍:**
```
filter.go       → filter_test.go
queue.go        → queue_test.go
```

**기본 구조:**
```go
func TestFilterCheck(t *testing.T) {
    filter := NewFilter()
    filter.AddRule("badword", "block")

    result := filter.CheckMessage("This has badword in it")

    if result.Action != "block" {
        t.Errorf("Expected block, got %s", result.Action)
    }
}
```

### 2. Race Detector (⭐ 매우 중요!)

**이 프로젝트는 동시성이 핵심이므로 race detector는 필수입니다!**

```bash
# 모든 테스트를 race detector와 함께 실행
go test -race ./...

# 특정 패키지만
cd proxy
go test -race

# 긴 테스트는 타임아웃 설정
go test -race -timeout 30s ./...
```

**Race condition 예시:**
```go
// filter_test.go에서 동시성 테스트
func TestConcurrentFilterAccess(t *testing.T) {
    t.Parallel()  // 병렬 실행

    filter := NewFilter()
    var wg sync.WaitGroup

    // 100개 고루틴이 동시에 규칙 추가
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(n int) {
            defer wg.Done()
            filter.AddRule(fmt.Sprintf("word%d", n), "block")
        }(i)
    }

    wg.Wait()
    // race detector가 문제 발견하면 실패
}
```

### 3. 커버리지 (Coverage)

```bash
# 커버리지 리포트 생성
go test -coverprofile=coverage.out ./...

# 브라우저에서 보기
go tool cover -html=coverage.out
```

### 4. 벤치마크 (Benchmarks)

```go
func BenchmarkFilterCheck(b *testing.B) {
    filter := NewFilter()
    filter.AddRule("badword", "block")
    message := "This is a clean message"

    b.ResetTimer()  // 셋업 시간 제외
    for i := 0; i < b.N; i++ {
        filter.CheckMessage(message)
    }
}
```

```bash
# 벤치마크 실행
go test -bench=. -benchmem
```

**프로젝트 테스트 스크립트:**
```bash
# 빠른 테스트
./run_tests.sh quick

# Race 조건 감지
./run_tests.sh race

# 커버리지
./run_tests.sh coverage

# 벤치마크
./run_tests.sh bench

# 파일 변경 감지 시 자동 테스트 (entr 필요)
./run_tests.sh watch
```

---

## Common Issues & Solutions

### Issue 1: "undefined: websocket"
```bash
# 원인: 패키지가 설치되지 않음
# 해결:
go get github.com/gorilla/websocket
```

### Issue 2: "package X is not in GOROOT"
```bash
# 원인: go.mod 파일 없음
# 해결:
go mod init <module-name>
go mod tidy
```

### Issue 3: Race Condition
```bash
# 증상: go test -race에서 경고
# 원인: 여러 고루틴이 같은 변수에 접근
# 해결: 채널 사용 또는 sync.Mutex 추가
```

**예시:**
```go
// ❌ Race
type Hub struct {
    clients map[*Client]bool
}

func (h *Hub) ClientCount() int {
    return len(h.clients)  // ⚠️ Race!
}

// ✅ 해결 방법 1: Mutex
type Hub struct {
    clients map[*Client]bool
    mu      sync.RWMutex
}

func (h *Hub) ClientCount() int {
    h.mu.RLock()
    defer h.mu.RUnlock()
    return len(h.clients)
}

// ✅ 해결 방법 2: 채널 (이 프로젝트 방식)
// Hub.run() 단일 고루틴에서만 clients 접근
// 외부에서는 채널로 요청 전송
```

### Issue 4: Memory Leak (고루틴 누수)
```bash
# 증상: 메모리 사용량이 계속 증가
# 원인: 종료되지 않는 고루틴
# 해결: Context 또는 done 채널로 종료 시그널
```

### Issue 5: Deadlock
```bash
# 증상: 프로그램이 멈춤, "all goroutines are asleep"
# 원인: 채널에서 대기 중인데 아무도 보내지/받지 않음
# 해결:
# 1. 버퍼 채널 사용: make(chan Type, size)
# 2. select with default 사용
# 3. 채널을 close()로 닫기
```

**예시:**
```go
// ❌ Deadlock
ch := make(chan int)
ch <- 42  // 버퍼 없고 받는 사람 없음 → 영원히 대기

// ✅ 해결 1: 버퍼 채널
ch := make(chan int, 1)
ch <- 42  // OK!

// ✅ 해결 2: 별도 고루틴
ch := make(chan int)
go func() {
    ch <- 42
}()
value := <-ch
```

---

## Git Commit Convention

**Format:** `<type>(<scope>): <subject>`

### Types
- `feat`: 새로운 기능
- `fix`: 버그 수정
- `docs`: 문서 변경
- `style`: 코드 포맷팅 (기능 변경 없음)
- `refactor`: 리팩토링
- `test`: 테스트 추가/수정
- `chore`: 빌드, 설정 등
- `perf`: 성능 개선

### Scopes (이 프로젝트)
- `backend`: 백엔드 서버
- `proxy`: 프록시 서버
- `voice`: 음성 서버
- `frontend`: 프론트엔드
- `api`: API 관련
- `filter`: 필터 엔진
- `docs`: 문서
- `ci`: CI/CD

### Examples
```bash
feat(proxy): implement message filtering engine
fix(backend): prevent race condition in hub broadcast
docs(readme): update architecture diagram
test(proxy): add concurrent filter rule updates test
refactor(backend): extract client cleanup logic
perf(proxy): optimize message queue with buffered channel
```

### ⚠️ Important
**DO NOT include AI-generated markers!**
- ❌ `feat(backend): add user authentication 🤖 Generated with Claude`
- ✅ `feat(backend): add user authentication`

커밋 메시지는 깔끔하고 전문적으로 작성하세요.

---

## Project Structure Reference

```
chat-system/
├── backend/          # WebSocket 채팅 서버 (port 8080)
│   ├── main.go       # 서버 진입점
│   ├── hub.go        # Hub-Spoke 패턴 (클라이언트 관리)
│   ├── client.go     # WebSocket 클라이언트 관리 (readPump/writePump)
│   ├── message.go    # 메시지 데이터 구조
│   └── go.mod        # module: chat-backend
│
├── proxy/            # 메시지 필터링 프록시 (port 8081)
│   ├── main.go       # 프록시 진입점
│   ├── proxy.go      # 양방향 릴레이 로직
│   ├── filter.go     # 메시지 필터링 엔진 (sync.RWMutex)
│   ├── queue.go      # 메시지 큐 구현
│   ├── storage.go    # 감사 로그 저장
│   ├── api.go        # HTTP API (필터 규칙 관리)
│   ├── *_test.go     # 테스트 파일들
│   └── go.mod        # module: chat-system
│
├── voice/            # WebRTC SFU (port 9000, WIP)
│   ├── main.go       # WebRTC 서버 (실험적)
│   └── go.mod        # module: voice
│
└── frontend/         # 바닐라 JavaScript 클라이언트
    ├── index.html    # 채팅 UI
    ├── chat.js       # WebSocket 클라이언트
    └── admin.html    # 관리자 대시보드
```

---

## Reference Documentation

### Official Go Resources
- [Go Tour](https://go.dev/tour/) - 대화형 기초 문법
- [Go by Example](https://gobyexample.com/) - 실용적인 예제
- [Effective Go](https://go.dev/doc/effective_go) - Go 관용구와 패턴
- [Go Blog - Concurrency](https://go.dev/blog/pipelines) - 동시성 패턴

### Gorilla WebSocket
- [Examples](https://github.com/gorilla/websocket/tree/main/examples)
- [Chat Example](https://github.com/gorilla/websocket/tree/main/examples/chat) - 이 프로젝트와 유사

### Related Skills
- **[websocket-webrtc-patterns](../websocket-webrtc-patterns/skill.md)** - WebSocket 상세 패턴
- **[learning-assistant](../learning-assistant/skill.md)** - 개념 설명 및 한글 주석

### Detailed Resources
- [Concurrency Patterns](./resources/concurrency-patterns.md) - 동시성 상세 패턴
- [Testing Strategies](./resources/testing-strategies.md) - 테스팅 전략

---

## Quick Tips

### 디버깅
```go
// 변수 값 확인
log.Printf("[DEBUG] variable=%v", variable)

// 타입 확인
log.Printf("[DEBUG] type=%T", variable)

// 포인터 주소 확인
log.Printf("[DEBUG] pointer=%p", pointer)

// 고루틴 수 확인
import "runtime"
log.Printf("[DEBUG] goroutines=%d", runtime.NumGoroutine())
```

### 성능 프로파일링
```bash
# CPU 프로파일
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof

# 메모리 프로파일
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof
```

### 유용한 도구
```bash
# 코드 정적 분석
go vet ./...

# 더 엄격한 린터
go install golang.org/x/lint/golint@latest
golint ./...

# 의존성 그래프
go mod graph
```

---

## Remember

✅ **DO:**
- 항상 `go.mod` 먼저 확인
- Race detector로 테스트
- defer로 리소스 정리
- 채널로 통신
- 에러 즉시 체크

❌ **DON'T:**
- go mod 없이 import
- 고루틴 cleanup 없이 생성
- 공유 변수로 통신
- 에러 무시
- WebSocket 동시 읽기/쓰기

---

**이 스킬은 실시간으로 업데이트됩니다. 프로젝트에서 새로운 패턴을 발견하면 추가하세요!**
