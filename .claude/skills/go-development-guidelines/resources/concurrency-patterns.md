# Go Concurrency Patterns (동시성 패턴)

> Minigo 채팅 시스템에서 사용되는 실제 동시성 패턴 심층 분석

---

## Table of Contents
1. [Hub-Spoke Pattern](#hub-spoke-pattern) - 중앙 집중식 브로드캐스팅
2. [readPump/writePump Pattern](#readpumpwritepump-pattern) - WebSocket 양방향 통신
3. [RWMutex Pattern](#rwmutex-pattern) - 읽기 중심 동시성 제어
4. [Channel Communication](#channel-communication) - 채널 기반 통신
5. [Graceful Shutdown](#graceful-shutdown) - 안전한 종료

---

## Hub-Spoke Pattern

### 개념
**Hub-Spoke 패턴**은 중앙 허브가 모든 클라이언트를 관리하고 메시지를 분배하는 패턴입니다.

### 비유
🏢 **우체국 비유:**
- Hub = 우체국 중앙 분류소
- Clients = 각 가정의 우편함
- Messages = 우편물
- 우체국 직원 한 명이 모든 우편물을 분류하고 배달 → 충돌 없음

### 실제 코드: `backend/hub.go`

```go
type Hub struct {
    // 연결된 모든 클라이언트 (set처럼 사용)
    clients    map[*Client]bool

    // 브로드캐스트할 메시지 채널
    broadcast  chan *Message

    // 신규 클라이언트 등록 채널
    register   chan *Client

    // 클라이언트 해제 채널
    unregister chan *Client
}

func (h *Hub) run() {
    for {
        select {
        case client := <-h.register:
            // 새 클라이언트 추가
            h.clients[client] = true

        case client := <-h.unregister:
            // 클라이언트 제거
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                close(client.send)
            }

        case message := <-h.broadcast:
            // 모든 클라이언트에게 메시지 전송
            jsonMessage, _ := json.Marshal(message)

            for client := range h.clients {
                // 같은 방에 있는 클라이언트만
                if client.room != message.Room {
                    continue
                }

                // Non-blocking send
                select {
                case client.send <- jsonMessage:
                    // 성공
                default:
                    // 채널이 가득 찼으면 클라이언트 제거
                    close(client.send)
                    delete(h.clients, client)
                }
            }
        }
    }
}
```

### 왜 이 패턴이 안전한가?

**핵심: 단일 고루틴이 `clients` map을 독점 관리**

```go
// ✅ 안전: hub.run() 고루틴만 clients에 접근
// → Mutex 필요 없음!

// run() 고루틴이 하는 일:
// 1. register 채널에서 받음 → clients에 추가
// 2. unregister 채널에서 받음 → clients에서 제거
// 3. broadcast 채널에서 받음 → clients 순회하며 전송
```

**다른 고루틴들은 채널로만 소통:**
```go
// 다른 곳에서 클라이언트 등록
hub.register <- newClient  // 직접 접근 ❌, 채널 사용 ✅
```

### Non-Blocking Send의 중요성

```go
select {
case client.send <- jsonMessage:
    // 성공적으로 전송됨
default:
    // 채널이 가득 찼음 (느린 클라이언트)
    close(client.send)
    delete(h.clients, client)
}
```

**왜 필요한가?**
- 느린 클라이언트 한 명 때문에 전체 시스템이 멈추면 안 됨
- `default` 케이스로 즉시 탈출 → Hub는 계속 작동
- 느린 클라이언트는 연결 해제 (공정한 자원 관리)

### Hub 패턴 활용 예시

**새 기능: 클라이언트 수 조회**

❌ **잘못된 방법 (Race!):**
```go
func (h *Hub) ClientCount() int {
    return len(h.clients)  // ⚠️ 다른 고루틴에서 호출 시 race!
}
```

✅ **올바른 방법 1: Mutex 추가**
```go
type Hub struct {
    clients map[*Client]bool
    mu      sync.RWMutex  // 추가
    // ...
}

func (h *Hub) run() {
    for {
        select {
        case client := <-h.register:
            h.mu.Lock()
            h.clients[client] = true
            h.mu.Unlock()
        // ...
        }
    }
}

func (h *Hub) ClientCount() int {
    h.mu.RLock()
    defer h.mu.RUnlock()
    return len(h.clients)
}
```

✅ **올바른 방법 2: 채널 사용 (더 Go다운 방식)**
```go
type Hub struct {
    // ... 기존 필드
    clientCount chan chan int  // 요청 채널
}

func (h *Hub) run() {
    for {
        select {
        // ... 기존 케이스들
        case req := <-h.clientCount:
            req <- len(h.clients)  // 응답 전송
        }
    }
}

func (h *Hub) ClientCount() int {
    response := make(chan int)
    h.clientCount <- response
    return <-response
}
```

---

## readPump/writePump Pattern

### 개념
**WebSocket 연결에서 읽기와 쓰기를 별도의 고루틴으로 분리**

### 왜 분리해야 하나?

**WebSocket 라이브러리 제약:**
- Gorilla WebSocket은 **동시 읽기 또는 동시 쓰기를 허용하지 않음**
- 한 고루틴이 읽고 쓰기를 동시에 하면 데드락 발생 가능
- 타임아웃 관리가 독립적으로 필요 (읽기 타임아웃 ≠ 쓰기 타임아웃)

### 실제 코드: `backend/client.go`

#### Client 구조체
```go
type Client struct {
    hub      *Hub
    conn     *websocket.Conn
    send     chan []byte        // writePump가 여기서 읽음
    username string
    room     string
}
```

#### readPump: WebSocket → Hub

```go
func (c *Client) readPump() {
    // 함수 종료 시 정리 작업
    defer func() {
        // 퇴장 메시지 전송
        leaveMessage := NewMessage("leave", c.username, c.room,
                                   c.username+" left the room")
        c.hub.broadcast <- leaveMessage

        // Hub에서 클라이언트 제거
        c.hub.unregister <- c

        // WebSocket 연결 닫기
        c.conn.Close()
    }()

    // 타임아웃 설정
    c.conn.SetReadDeadline(time.Now().Add(pongWait))
    c.conn.SetReadLimit(maxMessageSize)

    // Pong 핸들러: 클라이언트가 살아있는지 확인
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(pongWait))
        return nil
    })

    // 무한 루프: 메시지 읽기
    for {
        _, message, err := c.conn.ReadMessage()
        if err != nil {
            // 예상치 못한 종료 에러만 로깅
            if websocket.IsUnexpectedCloseError(err,
                websocket.CloseGoingAway,
                websocket.CloseAbnormalClosure) {
                log.Printf("error: %v", err)
            }
            break  // 루프 종료 → defer 실행
        }

        // JSON 파싱
        var msg Message
        if err := json.Unmarshal(message, &msg); err != nil {
            log.Printf("json unmarshal error: %v", err)
            continue
        }

        // 사용자 정보 설정
        msg.Username = c.username
        msg.Room = c.room
        msg.Timestamp = time.Now()

        // Hub로 전송
        c.hub.broadcast <- &msg
    }
}
```

**핵심 포인트:**
1. **defer로 cleanup** - 함수가 어떻게 끝나든 정리 보장
2. **타임아웃 설정** - 좀비 연결 방지
3. **Pong 핸들러** - 연결 생존 확인 (heartbeat)
4. **에러 처리** - 정상 종료 vs 비정상 종료 구분

#### writePump: Hub → WebSocket

```go
func (c *Client) writePump() {
    // Ping 전송용 ticker
    ticker := time.NewTicker(pingPeriod)
    defer func() {
        ticker.Stop()
        c.conn.Close()
    }()

    for {
        select {
        case message, ok := <-c.send:
            // 쓰기 타임아웃 설정
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))

            if !ok {
                // 채널이 닫혔음 (Hub가 클라이언트 제거)
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }

            // 메시지 작성 시작
            w, err := c.conn.NextWriter(websocket.TextMessage)
            if err != nil {
                return
            }
            w.Write(message)

            // 배치 처리: 큐에 쌓인 메시지를 한꺼번에 전송
            n := len(c.send)
            for i := 0; i < n; i++ {
                w.Write([]byte{'\n'})
                w.Write(<-c.send)
            }

            // Writer 닫기 (실제 전송)
            if err := w.Close(); err != nil {
                return
            }

        case <-ticker.C:
            // 주기적으로 Ping 전송 (연결 생존 확인)
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}
```

**핵심 포인트:**
1. **Ticker로 주기적 Ping** - 클라이언트가 살아있는지 확인
2. **배치 처리** - 성능 최적화 (여러 메시지를 한 번에 전송)
3. **채널 닫힘 감지** - Hub가 클라이언트 제거 시 안전 종료

### Ping/Pong 메커니즘

```
                Client                          Server
                  │                              │
                  │<────────── Ping ──────────────│ (pingPeriod마다)
                  │                              │
                  │───────── Pong ──────────────>│ (자동 응답)
                  │                              │
                  │      (pongWait 안에 응답)    │
                  │                              │
    응답 없으면   │                              │ ReadDeadline 초과
    연결 종료 ────│                              │────> 연결 종료
```

**타임아웃 관계:**
```go
const (
    pongWait   = 60 * time.Second
    pingPeriod = (pongWait * 9) / 10  // 54초 (안전 마진)
    writeWait  = 10 * time.Second
)
```

- Server가 54초마다 Ping 전송
- Client는 자동으로 Pong 응답
- 60초 안에 Pong 못 받으면 연결 종료

### 클라이언트 생명주기

```go
// main.go에서 WebSocket 연결 처리
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println(err)
        return
    }

    // 클라이언트 생성
    client := &Client{
        hub:      hub,
        conn:     conn,
        send:     make(chan []byte, 256),  // 버퍼 채널!
        username: r.URL.Query().Get("username"),
        room:     r.URL.Query().Get("room"),
    }

    // Hub에 등록
    client.hub.register <- client

    // 2개의 고루틴 시작
    go client.writePump()  // Hub → WebSocket
    go client.readPump()   // WebSocket → Hub
}
```

**생명주기:**
1. WebSocket 연결 Upgrade
2. Client 객체 생성
3. Hub에 등록 (register 채널로)
4. readPump, writePump 고루틴 시작
5. 연결 종료 시:
   - readPump의 defer 실행 → Hub에서 unregister
   - writePump는 send 채널 닫힘 감지 → 종료

---

## RWMutex Pattern

### 개념
**읽기가 훨씬 많은 경우 성능 최적화**

### 비유
🏛️ **도서관 비유:**
- `sync.Mutex`: 한 번에 한 명만 입장 (읽기도 쓰기도)
- `sync.RWMutex`: 여러 명이 동시에 읽기 가능, 쓰기는 독점

### 실제 코드: `proxy/filter.go`

```go
type Filter struct {
    mu    sync.RWMutex  // 읽기/쓰기 분리 락
    rules []FilterRule
}

// 읽기 (Read Lock): 여러 고루틴이 동시 실행 가능
func (f *Filter) CheckMessage(msg *Message) (bool, *Message) {
    f.mu.RLock()         // 읽기 락 획득
    defer f.mu.RUnlock() // 함수 끝날 때 해제

    // 규칙 순회 (읽기만)
    for _, rule := range f.rules {
        if !rule.Enabled {
            continue
        }
        // ... 필터링 로직
    }

    return true, msg
}

// 쓰기 (Write Lock): 독점 접근
func (f *Filter) AddRule(rule FilterRule) int {
    f.mu.Lock()         // 쓰기 락 획득 (배타적)
    defer f.mu.Unlock()

    // rules 수정
    rule.ID = maxID + 1
    f.rules = append(f.rules, rule)
    return rule.ID
}

func (f *Filter) RemoveRule(id int) bool {
    f.mu.Lock()
    defer f.mu.Unlock()

    for i, rule := range f.rules {
        if rule.ID == id {
            f.rules = append(f.rules[:i], f.rules[i+1:]...)
            return true
        }
    }
    return false
}

// 읽기 전용: GetRules
func (f *Filter) GetRules() []FilterRule {
    f.mu.RLock()
    defer f.mu.RUnlock()

    // 복사본 반환 (외부에서 수정 못 하게)
    rules := make([]FilterRule, len(f.rules))
    copy(rules, f.rules)
    return rules
}
```

### RLock vs Lock 비교

| 작업 | Lock 타입 | 동시 실행 | 용도 |
|------|-----------|----------|------|
| CheckMessage() | RLock | ✅ 가능 (여러 고루틴) | 읽기만 |
| GetRules() | RLock | ✅ 가능 | 읽기만 |
| AddRule() | Lock | ❌ 독점 | 쓰기 |
| RemoveRule() | Lock | ❌ 독점 | 쓰기 |

**성능 차이:**
```
100개 고루틴이 동시에 CheckMessage() 호출:
- sync.Mutex:   순차 실행 (100배 느림)
- sync.RWMutex: 병렬 실행 (빠름!)

1개 고루틴이 AddRule() 호출:
- sync.Mutex:   즉시 락
- sync.RWMutex: 모든 RLock 해제 대기 후 락
```

### 주의사항

❌ **Deadlock 위험:**
```go
func (f *Filter) BadMethod() {
    f.mu.RLock()
    defer f.mu.RUnlock()

    // ⚠️ 이미 RLock 잡은 상태에서 Lock 시도 → Deadlock!
    f.AddRule(FilterRule{...})
}
```

✅ **올바른 방법:**
```go
func (f *Filter) GoodMethod() {
    // 필요한 데이터만 읽기
    f.mu.RLock()
    needsUpdate := len(f.rules) == 0
    f.mu.RUnlock()

    // 락 해제 후 쓰기
    if needsUpdate {
        f.AddRule(FilterRule{...})
    }
}
```

---

## Channel Communication

### 채널 종류

#### 1. Unbuffered Channel (버퍼 없음)
```go
ch := make(chan int)

// 보내는 쪽과 받는 쪽이 동시에 준비될 때까지 블록
go func() {
    ch <- 42  // 받는 사람 없으면 여기서 대기
}()

value := <-ch  // 보내는 사람 없으면 여기서 대기
```

**용도:** 동기화가 필요한 경우

#### 2. Buffered Channel (버퍼 있음)
```go
ch := make(chan int, 100)

ch <- 1  // 버퍼가 가득 찰 때까지 블록 안 됨
ch <- 2
ch <- 3

value := <-ch  // 1
```

**용도:** 일시적인 속도 차이 흡수

#### Minigo 프로젝트 예시

```go
// Hub 채널들
broadcast  chan *Message       // Unbuffered (즉시 처리)
register   chan *Client        // Unbuffered (즉시 등록)
unregister chan *Client        // Unbuffered (즉시 해제)

// Client 채널
send       chan []byte         // Buffered (256) - 급증 대응
```

**왜 client.send는 버퍼 채널인가?**
- 메시지가 폭증할 수 있음 (여러 사람이 동시에 채팅)
- Hub는 빠르게 보내고 싶음 (블록 안 되게)
- Client가 느리면 버퍼에 쌓임
- 버퍼 가득 차면 Hub가 클라이언트 제거 (default 케이스)

### Channel 방향성

```go
// 양방향 (기본)
var ch chan int

// 송신 전용
var sendCh chan<- int

// 수신 전용
var recvCh <-chan int
```

**함수 시그니처에서 활용:**
```go
// producer: 채널에 쓰기만
func producer(ch chan<- int) {
    ch <- 42  // OK
    // <-ch   // 컴파일 에러!
}

// consumer: 채널에서 읽기만
func consumer(ch <-chan int) {
    value := <-ch  // OK
    // ch <- 1     // 컴파일 에러!
}
```

### Channel 닫기

**규칙: 송신자가 닫는다**

```go
ch := make(chan int, 10)

// 송신자
go func() {
    for i := 0; i < 5; i++ {
        ch <- i
    }
    close(ch)  // 더 이상 보낼 게 없음
}()

// 수신자
for value := range ch {
    fmt.Println(value)  // 0, 1, 2, 3, 4
}
// 채널이 닫히면 range 종료
```

**닫힌 채널 특성:**
```go
close(ch)

value, ok := <-ch
// ok == false: 채널이 닫혔고 비어있음
// ok == true:  채널이 열려있거나, 닫혔지만 버퍼에 데이터 남음

ch <- 42  // panic: send on closed channel
```

**Minigo에서:**
```go
// Hub가 클라이언트 제거 시
close(client.send)

// writePump에서 감지
message, ok := <-c.send
if !ok {
    // 채널 닫힘 → CloseMessage 전송 후 종료
    c.conn.WriteMessage(websocket.CloseMessage, []byte{})
    return
}
```

---

## Graceful Shutdown

### Context를 이용한 종료 제어

```go
func main() {
    ctx, cancel := context.WithCancel(context.Background())

    // 여러 워커 시작
    for i := 0; i < 5; i++ {
        go worker(ctx, i)
    }

    // 시그널 대기 (Ctrl+C)
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    <-sigChan

    log.Println("Shutting down...")
    cancel()  // 모든 워커에게 종료 시그널

    time.Sleep(2 * time.Second)  // 정리 시간
    log.Println("Shutdown complete")
}

func worker(ctx context.Context, id int) {
    for {
        select {
        case <-ctx.Done():
            log.Printf("Worker %d: shutting down", id)
            return
        default:
            // 작업 수행
            doWork(id)
        }
    }
}
```

### WaitGroup을 이용한 완료 대기

```go
func main() {
    var wg sync.WaitGroup

    for i := 0; i < 5; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            worker(id)
        }(i)
    }

    wg.Wait()  // 모든 워커 종료 대기
    log.Println("All workers finished")
}
```

### Minigo에서의 정리

```go
// readPump의 defer
defer func() {
    leaveMessage := NewMessage("leave", c.username, c.room,
                               c.username+" left the room")
    c.hub.broadcast <- leaveMessage  // 1. 퇴장 메시지
    c.hub.unregister <- c             // 2. Hub에서 제거
    c.conn.Close()                    // 3. 연결 닫기
}()
```

**정리 순서 중요:**
1. 퇴장 메시지 전송 (다른 사용자에게 알림)
2. Hub에서 클라이언트 제거
3. WebSocket 연결 닫기

---

## Best Practices Summary

### ✅ DO

1. **채널로 통신, 메모리 공유 금지**
   ```go
   // Good
   ch <- data
   ```

2. **Mutex는 최소 범위에서**
   ```go
   mu.Lock()
   criticalSection()
   mu.Unlock()
   ```

3. **defer로 락 해제 보장**
   ```go
   mu.Lock()
   defer mu.Unlock()
   ```

4. **버퍼 채널로 속도 차이 흡수**
   ```go
   send := make(chan []byte, 256)
   ```

5. **읽기 중심이면 RWMutex**
   ```go
   var mu sync.RWMutex
   mu.RLock()  // 읽기
   mu.Lock()   // 쓰기
   ```

### ❌ DON'T

1. **고루틴 누수**
   ```go
   // Bad: 종료 조건 없음
   go func() {
       for {
           doWork()
       }
   }()
   ```

2. **락 중복 획득 (Deadlock)**
   ```go
   mu.Lock()
   defer mu.Unlock()
   anotherFunc()  // 내부에서 mu.Lock() 호출 → Deadlock
   ```

3. **닫힌 채널에 전송**
   ```go
   close(ch)
   ch <- 42  // panic!
   ```

4. **수신자가 채널 닫기**
   ```go
   // Bad: 송신자가 닫아야 함
   for value := range ch {
       if done {
           close(ch)  // 다른 수신자가 panic!
       }
   }
   ```

---

## Testing Concurrency

### Race Detector

```bash
go test -race ./...
```

**찾아내는 문제:**
- 여러 고루틴이 같은 변수에 접근
- 하나라도 쓰기면 race

### 동시성 테스트 예시

```go
func TestConcurrentBroadcast(t *testing.T) {
    hub := newHub()
    go hub.run()

    // 100개 클라이언트 동시 등록
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()

            client := &Client{
                hub:  hub,
                send: make(chan []byte, 10),
                room: "test",
            }
            hub.register <- client

            // 메시지 전송
            msg := &Message{Room: "test", Content: fmt.Sprintf("msg%d", id)}
            hub.broadcast <- msg

            time.Sleep(10 * time.Millisecond)
            hub.unregister <- client
        }(i)
    }

    wg.Wait()
    // race detector가 문제 찾으면 실패
}
```

---

## Further Reading

- [Go Concurrency Patterns](https://go.dev/blog/pipelines) - 공식 블로그
- [Share Memory By Communicating](https://go.dev/blog/codelab-share) - 철학
- [Go Memory Model](https://go.dev/ref/mem) - 메모리 모델 상세

---

**다음 학습:**
- [Testing Strategies](./testing-strategies.md) - 테스팅 전략
- [WebSocket Patterns](../../websocket-webrtc-patterns/skill.md) - WebSocket 상세
