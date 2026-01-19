# Bidirectional Relay Pattern (양방향 릴레이 패턴)

> Proxy의 핵심: 클라이언트 ↔ 백엔드 양방향 중계 심층 분석

---

## 개념

**Bidirectional Relay**는 두 WebSocket 연결 사이에서 메시지를 양방향으로 중계하는 패턴입니다.

### 비유

🌉 **다리 비유:**
- Proxy = 강 위의 다리
- Client = 강 한쪽 끝
- Backend = 강 반대편 끝
- Messages = 다리를 건너는 차량

다리는 양쪽 방향으로 교통을 중계하지만, 한 방향(Client→Backend)에만 통행료 검사소(Filter)가 있습니다.

---

## 아키텍처

### 전체 흐름

```
┌──────────┐                ┌───────────┐                ┌──────────┐
│ Browser  │                │   Proxy   │                │ Backend  │
│  Client  │                │           │                │  Server  │
└─────┬────┘                └─────┬─────┘                └────┬─────┘
      │                           │                           │
      │  1. WebSocket Connect     │                           │
      │─────────────────────────→ │                           │
      │                           │                           │
      │                           │  2. Dial Backend          │
      │                           │─────────────────────────→ │
      │                           │                           │
      │                           │  3. Start Relay           │
      │                           │    (2 goroutines)         │
      │                           │                           │
      │  ┌─────────────────────── │ ← clientToBackend ──────→ │
      │  │  4. Message            │    - Filter               │
      │──┘                        │    - Queue                │
      │                           │    - Storage              │
      │                           │                           │
      │ ← backendToClient ──────  │ ←─────────────────────────│
      │    (no filtering)         │  5. Broadcast             │
      │                           │                           │
```

### 구성 요소

```go
type Proxy struct {
    clientConn  *websocket.Conn  // 브라우저 연결
    backendConn *websocket.Conn  // 백엔드 연결
    filter      *Filter          // 메시지 필터
    queue       *MessageQueue    // 메시지 큐
    storage     *Storage         // 감사 로그
}
```

**각 컴포넌트 역할:**
- `clientConn`: 브라우저와의 WebSocket 연결
- `backendConn`: 백엔드 서버와의 WebSocket 연결
- `filter`: 나쁜 단어, 스팸 차단
- `queue`: 급증 시 버퍼링
- `storage`: 모든 메시지 로깅 (감사 추적)

---

## Proxy 생성 (NewProxy)

### 과정

```go
func NewProxy(clientConn *websocket.Conn, filter *Filter,
              queue *MessageQueue, storage *Storage,
              username, room string) (*Proxy, error) {

    // Step 1: 백엔드 URL 파싱
    backendURL, err := url.Parse(*backendAddr)
    // 예: "ws://localhost:8080" → URL 객체
    if err != nil {
        return nil, err
    }

    // Step 2: Query 파라미터 추가
    query := backendURL.Query()
    query.Set("username", username)
    query.Set("room", room)
    backendURL.RawQuery = query.Encode()
    // 결과: "ws://localhost:8080/ws?username=mark&room=general"

    // Step 3: 백엔드에 WebSocket 연결
    backendConn, _, err := websocket.DefaultDialer.Dial(
        backendURL.String(), nil)
    if err != nil {
        return nil, err  // 백엔드 연결 실패
    }

    // Step 4: Proxy 객체 반환
    return &Proxy{
        clientConn:  clientConn,   // 이미 연결됨 (ProxyServer.handleWebSocket에서)
        backendConn: backendConn,  // 방금 연결됨
        filter:      filter,       // 공유 Filter (모든 Proxy가 같은 규칙 사용)
        queue:       queue,        // 공유 Queue
        storage:     storage,      // 공유 Storage
    }, nil
}
```

### 시퀀스 다이어그램

```
Client                Proxy                 Backend
  │                     │                     │
  │  HTTP Upgrade       │                     │
  │────────────────────>│                     │
  │                     │                     │
  │  (clientConn 생성)  │                     │
  │                     │                     │
  │                     │  WebSocket Dial     │
  │                     │────────────────────>│
  │                     │                     │
  │                     │  (backendConn 생성) │
  │                     │<────────────────────│
  │                     │                     │
  │  NewProxy() 완료    │                     │
  │<────────────────────│                     │
```

---

## Relay 시작 (Start)

### sync.Once를 이용한 안전한 종료

```go
func (p *Proxy) Start() {
    // 종료 시그널 채널
    done := make(chan struct{})

    // ⭐ sync.Once: 한 번만 실행 보장
    var closeOnce sync.Once

    // 종료 함수 (여러 곳에서 호출 가능)
    closeDone := func() {
        closeOnce.Do(func() {
            close(done)  // 채널을 딱 한 번만 닫음
        })
    }

    // 두 고루틴 시작
    go p.clientToBackend(closeDone)  // 고루틴 1
    go p.backendToClient(closeDone)  // 고루틴 2

    // 둘 중 하나라도 종료되면 블록 해제
    <-done

    // 양쪽 연결 모두 닫기
    p.Close()
}
```

### sync.Once 동작 원리

**시나리오 1: 정상 종료 (클라이언트 종료)**

```
1. Client closes WebSocket
   ↓
2. clientToBackend() reads error
   ↓
3. defer closeDone() executes
   ↓
4. closeOnce.Do() → close(done) ✅ (첫 호출)
   ↓
5. backendToClient() 계속 실행 중...
   ↓
6. Backend reads error (Client disconnected)
   ↓
7. defer closeDone() executes
   ↓
8. closeOnce.Do() → (이미 실행됨, 무시) ❌
   ↓
9. <-done unblocks (done already closed)
   ↓
10. p.Close() → 양쪽 연결 정리
```

**시나리오 2: 백엔드 장애**

```
1. Backend crashes
   ↓
2. backendToClient() reads error
   ↓
3. defer closeDone() executes
   ↓
4. closeOnce.Do() → close(done) ✅
   ↓
5. clientToBackend() reads Client message
   ↓
6. Tries to send to Backend → error
   ↓
7. defer closeDone() executes
   ↓
8. closeOnce.Do() → (already executed) ❌
   ↓
9. <-done unblocks
   ↓
10. p.Close()
```

**만약 sync.Once 없다면?**

```go
// ❌ 위험한 코드
func closeDoneWrong() {
    close(done)  // 첫 번째 호출: OK
    close(done)  // 두 번째 호출: panic: close of closed channel
}
```

### 왜 done 채널을 사용하나?

**대안 1: WaitGroup 사용?**
```go
// ❌ 문제: 둘 다 끝날 때까지 대기
var wg sync.WaitGroup
wg.Add(2)

go func() {
    defer wg.Done()
    p.clientToBackend()
}()

go func() {
    defer wg.Done()
    p.backendToClient()
}()

wg.Wait()  // 둘 다 끝나야 종료
```

**현재 방식: 채널 사용**
```go
// ✅ 하나만 끝나도 종료
<-done  // 첫 번째 close(done)에 즉시 unblock
```

**왜 이게 맞나?**
- 한쪽이 끊어지면 중계 불가능 → 바로 종료해야 함
- 계속 기다리면 리소스 낭비

---

## clientToBackend: 필터링 중계

### 전체 흐름

```
Client Message
    ↓
1. Read from clientConn
    ↓
2. JSON Unmarshal
    ↓
3. Filter Check ──→ Blocked? → Drop
    ↓ Allowed
4. Queue Enqueue ──→ Full? → Drop
    ↓ OK
5. Storage Log (best-effort)
    ↓
6. JSON Marshal
    ↓
7. Write to backendConn
    ↓
Backend receives
```

### 코드 상세 분석

```go
func (p *Proxy) clientToBackend(closeDone func()) {
    defer closeDone()  // ⭐ 종료 시 반드시 실행

    for {
        // ═══════════════════════════════════════════════
        // Step 1: 클라이언트에서 메시지 읽기
        // ═══════════════════════════════════════════════
        _, data, err := p.clientConn.ReadMessage()
        if err != nil {
            // 예상치 못한 종료만 로깅
            if websocket.IsUnexpectedCloseError(err,
                websocket.CloseGoingAway,        // 브라우저 탭 닫기
                websocket.CloseAbnormalClosure) { // 네트워크 끊김
                log.Printf("[Proxy] Client read error: %v", err)
            }
            return  // for 루프 종료 → defer closeDone() 실행
        }

        // ═══════════════════════════════════════════════
        // Step 2: JSON 파싱
        // ═══════════════════════════════════════════════
        var msg Message
        if err := json.Unmarshal(data, &msg); err != nil {
            log.Printf("[Proxy] Invalid message format: %v", err)
            continue  // 이 메시지는 버리고 다음 메시지 기다림
        }

        // ═══════════════════════════════════════════════
        // Step 3: 필터링 검사
        // ═══════════════════════════════════════════════
        allowed, filteredMsg := p.filter.CheckMessage(&msg)

        if !allowed {
            // 차단된 메시지 (예: "badword" 포함)
            log.Printf("[Proxy] Message blocked from %s: %s",
                msg.Username, msg.Content)
            continue  // 전송하지 않고 다음 메시지로
        }

        if filteredMsg != nil {
            // 필터링됨 (예: "욕설" → "***")
            msg = *filteredMsg
        }

        // ═══════════════════════════════════════════════
        // Step 4: 큐에 추가 (버퍼링)
        // ═══════════════════════════════════════════════
        if !p.queue.Enqueue(&msg) {
            // 큐가 가득 참 (급증 상황)
            log.Printf("[Proxy] Queue full, message dropped from %s",
                msg.Username)
            continue  // 드롭
        }

        // ═══════════════════════════════════════════════
        // Step 5: 감사 로그 저장
        // ═══════════════════════════════════════════════
        if p.storage != nil {
            if err := p.storage.LogMessage(&msg); err != nil {
                // ⚠️ 로깅 실패해도 전송은 계속
                log.Printf("[Proxy] Failed to log message: %v", err)
            }
        }

        // ═══════════════════════════════════════════════
        // Step 6: JSON 직렬화
        // ═══════════════════════════════════════════════
        filteredData, err := json.Marshal(msg)
        if err != nil {
            log.Printf("[Proxy] Failed to marshal message: %v", err)
            continue
        }

        // ═══════════════════════════════════════════════
        // Step 7: 백엔드로 전송
        // ═══════════════════════════════════════════════
        if err := p.backendConn.WriteMessage(
            websocket.TextMessage, filteredData); err != nil {
            log.Printf("[Proxy] Failed to send to backend: %v", err)
            return  // 백엔드 연결 끊김 → 종료
        }
    }
}
```

### 에러 처리 전략

| 에러 위치 | 처리 | 이유 |
|----------|------|------|
| ReadMessage | return | Client 연결 끊김 → 중계 불가 |
| Unmarshal | continue | 잘못된 JSON → 건너뛰기 |
| Filter blocked | continue | 차단 규칙 → 전송 안 함 |
| Queue full | continue | 급증 → 드롭 (QoS) |
| Storage fail | log only | 로깅은 best-effort |
| Marshal | continue | 이론상 불가능 (이미 파싱됨) |
| WriteMessage | return | Backend 연결 끊김 → 중계 불가 |

### 필터링 상세

**Filter.CheckMessage() 반환값:**
```go
allowed, filteredMsg := p.filter.CheckMessage(&msg)

// Case 1: 통과
allowed = true, filteredMsg = nil
→ 원본 메시지 그대로 전송

// Case 2: 치환
allowed = true, filteredMsg = &Message{Content: "***"}
→ 필터링된 메시지 전송

// Case 3: 차단
allowed = false, filteredMsg = nil
→ 전송 안 함
```

**예시:**
```go
// 입력: "This has badword in it"
allowed = false  // → continue (전송 안 함)

// 입력: "This has 욕설"
allowed = true, filteredMsg.Content = "This has ***"
→ "This has ***" 전송

// 입력: "Clean message"
allowed = true, filteredMsg = nil
→ "Clean message" 전송
```

---

## backendToClient: 직통 중계

### 왜 필터링 안 하나?

**이유 1: 중복 방지**
- Client → Backend: 이미 필터링됨
- Backend → Clients: Hub가 브로드캐스트 (다시 필터링 불필요)

**이유 2: 성능**
- 필터링은 CPU 집약적 (정규식, 문자열 검색)
- 불필요한 처리 제거

**이유 3: 신뢰**
- Backend에서 오는 메시지는 이미 검증됨
- 시스템 메시지 (join/leave)도 통과해야 함

### 코드 분석

```go
func (p *Proxy) backendToClient(closeDone func()) {
    defer closeDone()

    // ═══════════════════════════════════════════════
    // 타임아웃 설정
    // ═══════════════════════════════════════════════
    p.backendConn.SetReadDeadline(time.Now().Add(60 * time.Second))
    p.clientConn.SetWriteDeadline(time.Now().Add(10 * time.Second))

    // Pong 핸들러: 백엔드 생존 확인
    p.backendConn.SetPongHandler(func(string) error {
        p.backendConn.SetReadDeadline(time.Now().Add(60 * time.Second))
        return nil
    })

    for {
        // ═══════════════════════════════════════════════
        // Step 1: 백엔드에서 읽기
        // ═══════════════════════════════════════════════
        _, data, err := p.backendConn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err,
                websocket.CloseGoingAway,
                websocket.CloseAbnormalClosure) {
                log.Printf("[Proxy] Backend read error: %v", err)
            }
            return  // 백엔드 종료 → 중계 종료
        }

        // ═══════════════════════════════════════════════
        // Step 2: 클라이언트로 그대로 전송
        // ═══════════════════════════════════════════════
        if err := p.clientConn.WriteMessage(
            websocket.TextMessage, data); err != nil {
            log.Printf("[Proxy] Failed to send to client: %v", err)
            return  // 클라이언트 종료 → 중계 종료
        }

        // 끝! 간단함
    }
}
```

**clientToBackend와 비교:**
- ❌ JSON 파싱 없음
- ❌ 필터링 없음
- ❌ 큐잉 없음
- ❌ 로깅 없음
- ✅ 그냥 중계만

**성능 이점:**
```
clientToBackend:  읽기 → 파싱 → 필터 → 큐 → 로그 → 직렬화 → 쓰기
backendToClient:  읽기 → 쓰기

약 5-10배 빠름
```

---

## Close: 정리

### 안전한 종료

```go
func (p *Proxy) Close() {
    // ⭐ nil 체크 (방어적 프로그래밍)
    if p.clientConn != nil {
        p.clientConn.Close()
    }
    if p.backendConn != nil {
        p.backendConn.Close()
    }
}
```

**왜 nil 체크?**

```go
// NewProxy()에서 백엔드 연결 실패 시
backendConn, _, err := websocket.DefaultDialer.Dial(...)
if err != nil {
    return nil, err  // ← backendConn 생성 안 됨
}

// Close() 호출 시 backendConn == nil
// nil.Close() → panic!
// if backendConn != nil 체크 → 안전
```

### 종료 흐름

```
1. 한쪽 고루틴 에러 발생
   ↓
2. defer closeDone() 실행
   ↓
3. sync.Once → close(done)
   ↓
4. <-done unblocks
   ↓
5. p.Close() 호출
   ↓
6. clientConn.Close()
   backendConn.Close()
   ↓
7. 다른 고루틴도 ReadMessage() 에러
   ↓
8. defer closeDone() 실행 (but already closed)
   ↓
9. 두 고루틴 모두 종료
   ↓
10. Proxy 객체 정리 완료
```

---

## 성능 최적화

### Queue의 역할

**급증 상황:**
```
100 clients → Proxy → Backend (single connection)

Client 1 ─┐
Client 2 ─┤
  ...     ├─→ Queue (buffer 100) → Backend
Client 100─┘

Backend가 처리 속도보다 메시지가 빠르면?
→ Queue에 쌓임
→ 큐 가득 차면 드롭 (공정한 자원 관리)
```

**코드:**
```go
if !p.queue.Enqueue(&msg) {
    // 큐 가득 참 → 드롭
    log.Printf("[Proxy] Queue full, message dropped")
    continue
}
```

### Storage의 역할

**감사 로깅 (Audit Trail):**
```
모든 메시지를 logs/messages.jsonl에 저장
→ 나중에 분석, 규제 준수, 디버깅
```

**Best-effort 전략:**
```go
if err := p.storage.LogMessage(&msg); err != nil {
    // 로깅 실패해도 메시지 전송은 계속
    log.Printf("[Proxy] Failed to log message: %v", err)
}
```

**왜 best-effort?**
- 디스크 가득 참 → 로깅 실패
- 하지만 채팅은 계속 작동해야 함 (가용성 우선)
- 로깅은 부가 기능

---

## 일반적인 문제

### 문제 1: 백엔드 연결 실패

**증상:**
```
[Proxy] Failed to connect to backend: dial tcp 127.0.0.1:8080: connect: connection refused
```

**원인:**
- Backend 서버 미실행
- 잘못된 주소 설정

**해결:**
```bash
# Backend 실행 확인
./start_backend.sh

# 포트 확인
lsof -i :8080

# Proxy 주소 설정 확인
./start_proxy.sh --backend=ws://localhost:8080
```

### 문제 2: 메시지 드롭

**증상:**
```
[Proxy] Queue full, message dropped from user1
```

**원인:**
- 급증 (너무 많은 메시지)
- Backend 느림

**해결:**
```go
// queue.go에서 큐 크기 증가
const queueSize = 100   // 기본값
const queueSize = 1000  // 증가
```

### 문제 3: 필터링 안 됨

**증상:** "badword" 포함 메시지가 통과됨

**디버깅:**
```go
// proxy.go에 로그 추가
log.Printf("[Proxy] Filter check: allowed=%v, msg=%+v", allowed, msg)

// filter.go에서 규칙 확인
log.Printf("[Filter] Rules: %+v", f.GetRules())
```

---

## 테스팅

### 단위 테스트

```go
func TestProxyClientToBackend(t *testing.T) {
    // Mock Filter (차단하지 않음)
    filter := &Filter{rules: []FilterRule{}}

    // Mock Queue
    queue := NewMessageQueue(10)

    // Mock WebSocket 연결 (httptest 사용)
    // ...

    // Test
}
```

### 통합 테스트

```bash
# Terminal 1: Backend 시작
./start_backend.sh

# Terminal 2: Proxy 시작
./start_proxy.sh

# Terminal 3: 메시지 전송
echo '{"type":"message","content":"test"}' | \
  websocat ws://localhost:8081/ws?username=test&room=general
```

---

## 요약

### 핵심 개념

1. **양방향 중계**: Client ↔ Proxy ↔ Backend
2. **비대칭 필터링**: Client → Backend만 필터링
3. **sync.Once**: 안전한 종료 보장
4. **Done 채널**: 하나만 종료해도 전체 종료

### 구현 포인트

✅ **DO:**
- sync.Once로 채널 중복 닫기 방지
- nil 체크로 방어적 프로그래밍
- 로깅은 best-effort

❌ **DON'T:**
- 양방향 모두 필터링 (성능 낭비)
- WaitGroup으로 둘 다 대기 (불필요)
- Storage 실패 시 전송 중단 (가용성 저하)

---

**다음 학습:**
- [Main Skill](../skill.md) - WebSocket 패턴 메인
- [Pion WebRTC SFU](./pion-webrtc-sfu.md) - 음성 서버 구현
