# WebSocket & WebRTC Patterns

> Minigo 채팅 시스템의 실시간 통신 패턴 완전 가이드

## When This Skill Activates

이 스킬은 다음 상황에서 자동 활성화됩니다:
- `backend/*.go`, `proxy/*.go`, `voice/*.go` 파일 수정 시
- WebSocket/WebRTC 관련 키워드 사용 시
  - websocket, webrtc, pion, gorilla
  - hub, broadcast, relay, sfu
  - readPump, writePump, upgrader
- `/websocket`, `/webrtc` 명령어 입력 시

---

## Architecture Overview

### 전체 시스템 구조

```
┌─────────────────┐
│ Browser Client  │
│  (JavaScript)   │
└────────┬────────┘
         │ WebSocket ws://localhost:8081/ws
         ▼
┌─────────────────┐
│  Proxy Server   │◄─── HTTP API: /filter/rules (필터 규칙 관리)
│  (Port 8081)    │
│                 │
│  ┌───────────┐  │
│  │  Filter   │  │ ← 클라이언트 → 백엔드 방향만 필터링
│  └───────────┘  │
│  ┌───────────┐  │
│  │   Queue   │  │ ← 메시지 버퍼링 & 에러 처리
│  └───────────┘  │
│  ┌───────────┐  │
│  │  Storage  │  │ ← 감사 로깅 (logs/messages.jsonl)
│  └───────────┘  │
└────────┬────────┘
         │ WebSocket ws://localhost:8080/ws
         ▼
┌─────────────────┐
│ Backend Server  │
│  (Port 8080)    │
│                 │
│  ┌───────────┐  │
│  │    Hub    │  │ ← Hub-Spoke 패턴
│  │           │  │
│  │  clients  │  │ ← 연결된 모든 클라이언트
│  └───────────┘  │
└─────────────────┘

Voice Server (Port 9000) - WIP
Pion WebRTC SFU (Selective Forwarding Unit)
```

---

## Pattern 1: Hub-Spoke (중앙 집중식 브로드캐스팅)

### 📍 위치
`backend/hub.go` - Backend 서버의 핵심 패턴

### 개념
**중앙 Hub가 모든 클라이언트를 관리하고 메시지를 분배**

### 비유
🏢 **우체국 비유:**
- Hub = 우체국 중앙 분류소 (1명의 직원)
- Clients = 각 가정의 우편함
- Messages = 우편물
- Channels = 우편물 접수 창구

직원 한 명이 모든 우편물을 처리 → 충돌 없음, 효율적

### 구조

```go
type Hub struct {
    // 연결된 모든 클라이언트 (map을 set처럼 사용)
    clients    map[*Client]bool

    // 브로드캐스트할 메시지 채널
    broadcast  chan *Message

    // 신규 클라이언트 등록 채널
    register   chan *Client

    // 클라이언트 해제 채널
    unregister chan *Client
}
```

### 핵심 로직: Hub.run()

**단일 고루틴이 모든 것을 관리 → 동시성 안전**

```go
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
                close(client.send)  // 채널 닫기
            }

        case message := <-h.broadcast:
            // 모든 클라이언트에게 메시지 전송
            jsonMessage, _ := json.Marshal(message)

            for client := range h.clients {
                // 같은 방(room)에 있는 클라이언트만
                if client.room != message.Room {
                    continue
                }

                // Non-blocking send (중요!)
                select {
                case client.send <- jsonMessage:
                    // 성공
                default:
                    // 채널이 가득 참 → 느린 클라이언트 제거
                    close(client.send)
                    delete(h.clients, client)
                }
            }
        }
    }
}
```

### 왜 안전한가?

**✅ 핵심: `clients` map은 단 하나의 고루틴(run())만 접근**

```
hub.run() 고루틴만 clients에 접근
    ↓
register 채널에서 받음   → clients에 추가
unregister 채널에서 받음 → clients에서 제거
broadcast 채널에서 받음  → clients 순회하며 전송
    ↓
Mutex 필요 없음!
```

**다른 고루틴은 채널로만 소통:**
```go
// 클라이언트 등록 (다른 고루틴에서)
hub.register <- newClient  // 직접 접근 ❌, 채널 사용 ✅
```

### Non-Blocking Send의 중요성

```go
select {
case client.send <- jsonMessage:
    // 성공적으로 전송
default:
    // 채널이 가득 찼음 (느린 클라이언트)
    close(client.send)
    delete(h.clients, client)
}
```

**왜 default 케이스가 필요한가?**
1. **시스템 보호**: 느린 클라이언트 한 명 때문에 전체 Hub가 블록되면 안 됨
2. **공정성**: 모든 클라이언트에게 공평한 기회
3. **자동 정리**: 응답 없는 클라이언트는 자동 제거

### Room 기반 필터링

```go
for client := range h.clients {
    if client.room != message.Room {
        continue  // 다른 방은 스킵
    }
    // 같은 방에만 전송
}
```

**채팅방 격리**: "general" 방과 "private" 방의 메시지가 섞이지 않음

---

## Pattern 2: readPump/writePump (양방향 분리)

### 📍 위치
`backend/client.go` - 각 WebSocket 클라이언트

### 개념
**WebSocket 읽기와 쓰기를 별도 고루틴으로 분리**

### 왜 분리해야 하나?

**Gorilla WebSocket 제약사항:**
1. ❌ 동시 읽기 불가 (Multiple goroutines can't read simultaneously)
2. ❌ 동시 쓰기 불가 (Multiple goroutines can't write simultaneously)
3. ✅ 읽기와 쓰기는 동시 가능 (One reader + One writer = OK)

**분리하지 않으면:**
- 읽기 중에 쓰기 못 함 → 응답 지연
- 타임아웃 관리 복잡 (읽기 타임아웃 ≠ 쓰기 타임아웃)
- 데드락 위험

### Client 구조

```go
type Client struct {
    hub      *Hub
    conn     *websocket.Conn
    send     chan []byte        // writePump가 여기서 읽음
    username string
    room     string
}
```

**핵심: `send` 채널이 두 고루틴을 연결**
```
readPump  →  Hub  →  broadcast  →  Hub.run()  →  client.send  →  writePump
(읽기)                                                              (쓰기)
```

### readPump: WebSocket → Hub

**역할: 클라이언트에서 메시지 읽어서 Hub로 전달**

```go
func (c *Client) readPump() {
    // 함수 종료 시 정리 (어떤 경로로 끝나든 실행됨)
    defer func() {
        // 1. 퇴장 메시지 전송
        leaveMessage := NewMessage("leave", c.username, c.room,
                                   c.username+" left the room")
        c.hub.broadcast <- leaveMessage

        // 2. Hub에서 클라이언트 제거
        c.hub.unregister <- c

        // 3. WebSocket 연결 닫기
        c.conn.Close()
    }()

    // 타임아웃 설정 (좀비 연결 방지)
    c.conn.SetReadDeadline(time.Now().Add(pongWait))  // 60초
    c.conn.SetReadLimit(maxMessageSize)                 // 512 bytes

    // Pong 핸들러: 클라이언트가 살아있는지 확인
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(pongWait))
        return nil
    })

    // 무한 루프: 메시지 계속 읽기
    for {
        _, message, err := c.conn.ReadMessage()
        if err != nil {
            // 정상 종료 vs 비정상 종료 구분
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
            continue  // 이 메시지는 스킵, 계속 읽기
        }

        // 사용자 정보 설정
        msg.Username = c.username
        msg.Room = c.room
        msg.Timestamp = time.Now()

        // Hub로 전송 (브로드캐스트 채널로)
        c.hub.broadcast <- &msg
    }
}
```

**핵심 포인트:**
1. **defer로 cleanup 보장** - 어떻게 끝나든 정리됨
2. **타임아웃으로 좀비 방지** - 60초 동안 응답 없으면 종료
3. **Pong 핸들러** - 연결 생존 확인 (heartbeat)
4. **에러 구분** - 정상 종료는 로그 안 남김

### writePump: Hub → WebSocket

**역할: Hub에서 받은 메시지를 클라이언트로 전송**

```go
func (c *Client) writePump() {
    // Ping 전송용 ticker (54초마다)
    ticker := time.NewTicker(pingPeriod)
    defer func() {
        ticker.Stop()
        c.conn.Close()
    }()

    for {
        select {
        case message, ok := <-c.send:
            // 쓰기 타임아웃 설정 (10초)
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

            // 🚀 배치 처리: 큐에 쌓인 메시지를 한꺼번에 전송
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
1. **Ticker로 주기적 Ping** - 클라이언트 생존 확인
2. **배치 처리** - 성능 최적화 (여러 메시지를 한 번에)
3. **채널 닫힘 감지** - Hub가 제거 시 안전 종료

### Ping/Pong Heartbeat 메커니즘

```
Server (writePump)              Client (Browser)
      │                              │
      │────── Ping ─────────────────>│  (54초마다)
      │                              │
      │<────── Pong ─────────────────│  (자동 응답)
      │                              │
      │  (Pong 핸들러가 타임아웃 갱신) │
      │                              │
   60초 안에                      응답 없으면
   Pong 못 받으면                  연결 종료
   ReadDeadline 초과 ────────────────>
```

**타임아웃 설정:**
```go
const (
    pongWait   = 60 * time.Second        // Pong 대기 시간
    pingPeriod = (pongWait * 9) / 10     // 54초 (안전 마진)
    writeWait  = 10 * time.Second        // 쓰기 타임아웃
)
```

**왜 pingPeriod < pongWait 인가?**
- 네트워크 지연 고려 (6초 마진)
- Pong이 늦게 도착해도 타임아웃 안 됨

### 클라이언트 생명주기

```go
// main.go에서 WebSocket 연결 처리
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
    // 1. HTTP → WebSocket Upgrade
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println(err)
        return
    }

    // 2. Client 객체 생성
    client := &Client{
        hub:      hub,
        conn:     conn,
        send:     make(chan []byte, 256),  // ⭐ 버퍼 채널 (중요!)
        username: r.URL.Query().Get("username"),
        room:     r.URL.Query().Get("room"),
    }

    // 3. Hub에 등록
    client.hub.register <- client

    // 4. 두 고루틴 시작
    go client.writePump()  // Hub → WebSocket
    go client.readPump()   // WebSocket → Hub
}
```

**왜 send 채널이 버퍼 채널(256)인가?**
1. **급증 대응**: 메시지가 갑자기 많이 올 수 있음
2. **Hub 블록 방지**: Hub가 빠르게 보내고 다음 작업으로
3. **자동 조절**: 버퍼 가득 차면 Hub가 클라이언트 제거

---

## Pattern 3: Bidirectional Relay (양방향 릴레이)

### 📍 위치
`proxy/proxy.go` - Proxy 서버의 핵심 패턴

### 개념
**클라이언트 ↔ 백엔드 사이에서 메시지를 양방향 중계**

### 아키텍처

```
Client                Proxy                 Backend
  │                     │                     │
  │───── Message ──────>│                     │
  │                     │─── Filter ─────>    │  (필터링)
  │                     │─── Queue ──────>    │  (큐잉)
  │                     │─── Storage ────>    │  (로깅)
  │                     │                     │
  │                     │──── Forward ───────>│
  │                     │                     │
  │<──── Broadcast ─────│<──── Response ──────│  (필터링 안 함)
  │                     │                     │
```

**비대칭 필터링:**
- Client → Backend: 필터링 ✅ (나쁜 메시지 차단)
- Backend → Client: 필터링 ❌ (그대로 전달)

### Proxy 구조

```go
type Proxy struct {
    clientConn  *websocket.Conn  // 클라이언트 연결
    backendConn *websocket.Conn  // 백엔드 연결
    filter      *Filter          // 메시지 필터
    queue       *MessageQueue    // 메시지 큐
    storage     *Storage         // 감사 로그
}
```

### 생성자: NewProxy()

```go
func NewProxy(clientConn *websocket.Conn, filter *Filter,
              queue *MessageQueue, storage *Storage,
              username, room string) (*Proxy, error) {

    // 1. 백엔드 URL 구성
    backendURL, err := url.Parse(*backendAddr)  // ws://localhost:8080
    if err != nil {
        return nil, err
    }

    // 2. Query 파라미터 추가
    query := backendURL.Query()
    query.Set("username", username)
    query.Set("room", room)
    backendURL.RawQuery = query.Encode()
    // 결과: ws://localhost:8080/ws?username=mark&room=general

    // 3. 백엔드에 WebSocket 연결
    backendConn, _, err := websocket.DefaultDialer.Dial(
        backendURL.String(), nil)
    if err != nil {
        return nil, err
    }

    // 4. Proxy 객체 반환
    return &Proxy{
        clientConn:  clientConn,
        backendConn: backendConn,
        filter:      filter,
        queue:       queue,
        storage:     storage,
    }, nil
}
```

### 시작: Start()

**두 고루틴 실행 + 안전한 종료 보장**

```go
func (p *Proxy) Start() {
    done := make(chan struct{})      // 종료 시그널 채널
    var closeOnce sync.Once          // 한 번만 닫기 보장

    // 종료 함수 (어느 고루틴에서 호출해도 안전)
    closeDone := func() {
        closeOnce.Do(func() {
            close(done)  // 채널을 딱 한 번만 닫음
        })
    }

    // 두 고루틴 시작
    go p.clientToBackend(closeDone)  // Client → Backend
    go p.backendToClient(closeDone)  // Backend → Client

    // 둘 중 하나라도 종료되면 대기 종료
    <-done

    // 양쪽 연결 모두 닫기
    p.Close()
}
```

**✨ sync.Once의 마법:**
```
clientToBackend 종료 → closeDone() 호출 → done 채널 닫힘
    ↓
backendToClient도 종료 → closeDone() 호출 → (이미 닫혀서 무시)
    ↓
done 채널 닫혔으므로 <-done 블록 해제
    ↓
p.Close() 실행
```

**만약 sync.Once 없으면?**
```go
// ❌ 위험한 코드
close(done)  // 첫 번째 호출: OK
close(done)  // 두 번째 호출: panic: close of closed channel
```

### clientToBackend: 필터링 & 전달

**Client → Backend 방향 (필터링 적용)**

```go
func (p *Proxy) clientToBackend(closeDone func()) {
    defer closeDone()  // 종료 시 시그널

    for {
        // 1. 클라이언트에서 메시지 읽기
        _, data, err := p.clientConn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err,
                websocket.CloseGoingAway,
                websocket.CloseAbnormalClosure) {
                log.Printf("[Proxy] Client read error: %v", err)
            }
            return  // 종료 → closeDone() 실행
        }

        // 2. JSON 파싱
        var msg Message
        if err := json.Unmarshal(data, &msg); err != nil {
            log.Printf("[Proxy] Invalid message format: %v", err)
            continue  // 이 메시지는 스킵
        }

        // 3. 필터링 검사
        allowed, filteredMsg := p.filter.CheckMessage(&msg)
        if !allowed {
            log.Printf("[Proxy] Message blocked from %s: %s",
                msg.Username, msg.Content)
            continue  // 차단된 메시지는 전송 안 함
        }
        if filteredMsg != nil {
            msg = *filteredMsg  // 필터링된 메시지 사용 (예: 욕설 ***로 치환)
        }

        // 4. 큐에 추가 (버퍼링)
        if !p.queue.Enqueue(&msg) {
            log.Printf("[Proxy] Queue full, message dropped from %s",
                msg.Username)
            continue
        }

        // 5. 감사 로그 저장
        if p.storage != nil {
            if err := p.storage.LogMessage(&msg); err != nil {
                log.Printf("[Proxy] Failed to log message: %v", err)
            }
        }

        // 6. 백엔드로 전송
        filteredData, err := json.Marshal(msg)
        if err != nil {
            log.Printf("[Proxy] Failed to marshal message: %v", err)
            continue
        }

        if err := p.backendConn.WriteMessage(
            websocket.TextMessage, filteredData); err != nil {
            log.Printf("[Proxy] Failed to send to backend: %v", err)
            return  // 백엔드 연결 끊김 → 종료
        }
    }
}
```

**처리 흐름:**
```
1. 읽기 → 2. 파싱 → 3. 필터링 → 4. 큐잉 → 5. 로깅 → 6. 전송
```

**각 단계에서 실패 가능:**
- 필터링: blocked → continue (전송 안 함)
- 큐: full → continue (드롭)
- 로깅: 실패해도 전송 계속 (로그는 best-effort)

### backendToClient: 직통 전달

**Backend → Client 방향 (필터링 없음)**

```go
func (p *Proxy) backendToClient(closeDone func()) {
    defer closeDone()

    // 타임아웃 설정
    p.backendConn.SetReadDeadline(time.Now().Add(60 * time.Second))
    p.clientConn.SetWriteDeadline(time.Now().Add(10 * time.Second))

    // Pong 핸들러 (백엔드 생존 확인)
    p.backendConn.SetPongHandler(func(string) error {
        p.backendConn.SetReadDeadline(time.Now().Add(60 * time.Second))
        return nil
    })

    for {
        // 1. 백엔드에서 읽기
        _, data, err := p.backendConn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err,
                websocket.CloseGoingAway,
                websocket.CloseAbnormalClosure) {
                log.Printf("[Proxy] Backend read error: %v", err)
            }
            return
        }

        // 2. 클라이언트로 그대로 전송 (필터링 없음!)
        if err := p.clientConn.WriteMessage(
            websocket.TextMessage, data); err != nil {
            log.Printf("[Proxy] Failed to send to client: %v", err)
            return
        }
    }
}
```

**왜 필터링 안 하나?**
- Backend는 이미 Hub를 거쳐서 검증됨
- 불필요한 중복 처리 방지
- 성능 최적화

### Close: 정리

```go
func (p *Proxy) Close() {
    if p.clientConn != nil {
        p.clientConn.Close()
    }
    if p.backendConn != nil {
        p.backendConn.Close()
    }
}
```

**nil 체크 이유:**
- NewProxy()에서 백엔드 연결 실패 시 backendConn == nil
- 안전한 종료 보장

---

## Pattern 4: WebRTC SFU (진행 중)

### 📍 위치
`voice/main.go` - 음성 채팅 서버 (실험적)

### 개념
**SFU (Selective Forwarding Unit): 미디어 중계 서버**

### Mesh vs SFU

**Mesh (P2P):**
```
User1 ←─────→ User2
  ↑   ╲     ╱   ↑
  │     ╲ ╱     │
  │      ╳      │
  │     ╱ ╲     │
  ↓   ╱     ╲   ↓
User3 ←─────→ User4

4명 = 6개 연결 (N*(N-1)/2)
각자 3개 업로드, 3개 다운로드 → 대역폭 폭발
```

**SFU:**
```
User1 ──────→ SFU ──────→ User2
              ↑ │
User3 ────────┘ └────────→ User4

4명 = 4개 연결 (N개)
각자 1개 업로드, 3개 다운로드 → 대역폭 절약
```

### 현재 상태 (WIP)

```go
// voice/main.go
package main

import (
    "log"
    "net/http"

    "github.com/pion/webrtc/v4"
)

func main() {
    http.HandleFunc("/ws", handleWebSocket)

    log.Println("Voice server starting on :9000")
    log.Fatal(http.ListenAndServe(":9000", nil))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
    // TODO: WebRTC SFU 구현
    // - Peer 관리
    // - Track 중계
    // - ICE candidate 처리
}
```

### SFU 구현 계획

**필요한 구성 요소:**
1. **Peer 관리**: 연결된 모든 WebRTC peer
2. **Track 중계**: 한 peer의 audio/video → 다른 모든 peer
3. **Signaling**: WebSocket으로 SDP 교환
4. **ICE**: NAT 통과를 위한 candidate 교환

**상세 내용:** [resources/pion-webrtc-sfu.md](./resources/pion-webrtc-sfu.md)

---

## Best Practices

### ✅ DO

1. **WebSocket Upgrader 올바르게 설정**
   ```go
   var upgrader = websocket.Upgrader{
       ReadBufferSize:  1024,
       WriteBufferSize: 1024,
       CheckOrigin: func(r *http.Request) bool {
           // 개발: 모든 origin 허용
           return true
           // 프로덕션: origin 검증 필수!
       },
   }
   ```

2. **readPump/writePump 분리**
   ```go
   go client.readPump()   // 읽기 전용
   go client.writePump()  // 쓰기 전용
   ```

3. **타임아웃 설정**
   ```go
   conn.SetReadDeadline(time.Now().Add(pongWait))
   conn.SetWriteDeadline(time.Now().Add(writeWait))
   ```

4. **Pong 핸들러로 생존 확인**
   ```go
   conn.SetPongHandler(func(string) error {
       conn.SetReadDeadline(time.Now().Add(pongWait))
       return nil
   })
   ```

5. **Non-blocking send (Hub 패턴)**
   ```go
   select {
   case client.send <- message:
   default:
       // 느린 클라이언트 제거
   }
   ```

### ❌ DON'T

1. **동시 읽기/쓰기**
   ```go
   // ❌ 위험!
   conn.ReadMessage()   // 고루틴 1
   conn.WriteMessage()  // 고루틴 2 (같은 시간에)
   ```

2. **CheckOrigin 무시**
   ```go
   // ❌ 보안 위험! (CSRF 공격)
   CheckOrigin: func(r *http.Request) bool {
       return true  // 프로덕션에서 절대 금지!
   }
   ```

3. **타임아웃 없이 운영**
   ```go
   // ❌ 좀비 연결 생성
   conn.ReadMessage()  // 영원히 대기 가능
   ```

4. **채널 닫기 없음**
   ```go
   // ❌ 고루틴 누수
   delete(h.clients, client)  // 채널은 안 닫음
   ```

5. **Blocking send (Hub)**
   ```go
   // ❌ 전체 시스템 블록 위험
   client.send <- message  // 채널이 가득 차면 영원히 대기
   ```

---

## Common Issues

### Issue 1: "websocket: bad handshake"

**증상:** WebSocket 연결 실패

**원인:**
1. CheckOrigin 거부
2. 잘못된 Upgrade 헤더

**해결:**
```go
var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        origin := r.Header.Get("Origin")
        log.Printf("Origin: %s", origin)
        return true  // 개발 환경
    },
}
```

### Issue 2: 메시지 유실

**증상:** 클라이언트가 일부 메시지를 못 받음

**원인:**
1. `client.send` 채널 버퍼 작음
2. Hub의 default 케이스가 클라이언트 제거

**해결:**
```go
// 버퍼 크기 증가
send: make(chan []byte, 256)  // 기본값
send: make(chan []byte, 1024) // 증가
```

### Issue 3: Race Condition

**증상:** `go test -race` 경고

**원인:** Hub 외부에서 `clients` map 접근

**해결:**
```go
// ❌ Race
func (h *Hub) ClientCount() int {
    return len(h.clients)  // 위험!
}

// ✅ 채널 사용
type clientCountRequest struct {
    response chan int
}

// Hub.run() select에 추가:
case req := <-h.clientCount:
    req.response <- len(h.clients)
```

---

## Testing WebSocket

### 브라우저 테스트

```javascript
const ws = new WebSocket('ws://localhost:8081/ws?username=test&room=general');

ws.onopen = () => console.log('Connected');
ws.onmessage = (e) => console.log('Received:', e.data);
ws.onerror = (e) => console.error('Error:', e);
ws.onclose = (e) => console.log('Closed:', e.code, e.reason);

// 메시지 전송
ws.send(JSON.stringify({type: 'message', content: 'Hello'}));
```

### websocat CLI

```bash
# 설치
# https://github.com/vi/websocat

# 백엔드 직접 테스트
websocat ws://localhost:8080/ws?username=test&room=general

# 프록시 통해 테스트
websocat ws://localhost:8081/ws?username=test&room=general

# 메시지 전송
echo '{"type":"message","content":"test"}' | websocat ws://localhost:8081/ws?username=test&room=general
```

---

## Quick Reference

### 연결 흐름

```
Browser
   │
   │ 1. WebSocket handshake
   ▼
Proxy (port 8081)
   │
   │ 2. NewProxy() - 백엔드 연결
   ▼
Backend (port 8080)
   │
   │ 3. Hub에 등록
   ▼
Hub.run()
```

### 메시지 흐름

**송신:**
```
Browser → Proxy.clientToBackend → Filter → Queue → Backend.readPump → Hub.broadcast
```

**수신:**
```
Hub.broadcast → Hub.run() → client.send → Backend.writePump → Proxy.backendToClient → Browser
```

### 타임아웃 값

```go
pongWait   = 60 * time.Second    // Pong 대기
pingPeriod = 54 * time.Second    // Ping 주기
writeWait  = 10 * time.Second    // 쓰기 타임아웃
```

---

## Related Resources

- **[Concurrency Patterns](../go-development-guidelines/resources/concurrency-patterns.md)** - 동시성 심화
- **[Bidirectional Relay Deep Dive](./resources/bidirectional-relay.md)** - Proxy 패턴 상세
- **[Pion WebRTC SFU](./resources/pion-webrtc-sfu.md)** - 음성 서버 계획
- **[Gorilla WebSocket Examples](https://github.com/gorilla/websocket/tree/main/examples)** - 공식 예제

---

**이 스킬은 프로젝트와 함께 진화합니다. Voice 서버 구현 시 WebRTC 패턴이 추가될 예정입니다!**
