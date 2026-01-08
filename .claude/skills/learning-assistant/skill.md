# Learning Assistant (학습 도우미)

> Go 개발을 배우는 개발자를 위한 양질의 설명과 한글 지원

## When This Skill Activates

이 스킬은 다음 상황에서 **수동 활성화**됩니다:
- 사용자가 "왜", "what is", "explain" 등의 질문 키워드 사용 시
- 한글 주석 요청 시
- 개념 설명 요청 시
- `/explain`, `/learn` 명령어 입력 시

**자동 활성화되지 않습니다** - 사용자가 명시적으로 요청할 때만 동작합니다.

---

## Mission

**"Always explain WHY before implementing"**

이 프로젝트는 **학습 프로젝트**입니다. 코드를 작성하기 전에:
1. **왜** 이렇게 하는지 설명
2. **어떻게** 작동하는지 설명
3. **대안**은 무엇이고 왜 이 방법을 선택했는지 설명

---

## Core Principles

### 1. 가정하지 않기 (Never Assume)

❌ **나쁜 예:**
```go
// 메시지를 브로드캐스트합니다
c.hub.broadcast <- &msg
```

✅ **좋은 예:**
```go
// Hub에 메시지 전송 (broadcast 채널로)
// broadcast는 buffered channel이 아니므로
// Hub.run() 고루틴이 즉시 받을 때까지 이 라인에서 블록됨
// Hub가 빠르게 처리하므로 보통 문제 없음
c.hub.broadcast <- &msg
```

### 2. 개념 → 문법 → 예제 → 비유

**사용자가 "X가 뭐에요?" 물으면:**

```
1. 개념: X는 [한 문장으로 정의]
2. 문법: [기본 사용법 코드]
3. 예제: [프로젝트에서 실제 사용 예시]
4. 비유: [실생활 비유]
```

**예시:**

> **Q: "채널이 뭔가요?"**
>
> **A:**
>
> **개념:** 채널은 고루틴 간에 데이터를 안전하게 전송하는 파이프입니다.
>
> **문법:**
> ```go
> ch := make(chan int)        // unbuffered
> ch := make(chan int, 100)   // buffered
>
> ch <- 42      // 전송
> value := <-ch // 수신
> close(ch)     // 닫기
> ```
>
> **예제:** (backend/hub.go에서)
> ```go
> broadcast  chan *Message     // unbuffered
> register   chan *Client      // unbuffered
>
> // Hub.run()에서 수신
> case message := <-h.broadcast:
>     // 모든 클라이언트에게 전송
> ```
>
> **비유:**
> - **Unbuffered 채널**: 직접 손으로 전달 (보내는 사람과 받는 사람이 동시에 있어야 함)
> - **Buffered 채널**: 컨베이어 벨트 (N개까지 쌓아둘 수 있음, 받는 사람 없어도 보낼 수 있음)

---

## 한글 주석 작성 가이드

### 언제 한글 주석을 쓰나?

1. **학습 목적 코드**에 개념 설명
2. **복잡한 로직**의 흐름 설명
3. **왜 이렇게 했는지** 맥락 설명

### 좋은 한글 주석 vs 나쁜 한글 주석

❌ **나쁜 예 - 코드 번역:**
```go
// 클라이언트 맵
clients map[*Client]bool

// 브로드캐스트 채널
broadcast chan *Message
```

✅ **좋은 예 - 개념 설명:**
```go
// 연결된 모든 클라이언트를 추적하는 맵
// - key: Client 포인터, value: true (set처럼 사용)
// - map을 사용하는 이유: O(1) 삭제/조회
// - 이 map은 Hub.run() 고루틴만 접근 → mutex 불필요
clients map[*Client]bool

// 브로드캐스트할 메시지 채널
// - unbuffered: Hub가 즉시 받을 때까지 블록
// - 모든 클라이언트에게 전송되어야 하는 메시지
broadcast chan *Message
```

❌ **나쁜 예 - 불필요한 주석:**
```go
// i를 증가시킴
i++

// 만약 에러가 nil이 아니면
if err != nil {
    return err
}
```

✅ **좋은 예 - 왜/맥락:**
```go
// Hub의 비동기 처리를 위해 고루틴 시작
// run()은 무한 루프라 main이 블록되지 않게 백그라운드 실행
go hub.run()

// 예상치 못한 종료만 로깅 (정상 종료는 로그 안 남김)
// CloseGoingAway: 브라우저 탭 닫기
// CloseAbnormalClosure: 네트워크 끊김
if websocket.IsUnexpectedCloseError(err,
    websocket.CloseGoingAway,
    websocket.CloseAbnormalClosure) {
    log.Printf("error: %v", err)
}
```

### 함수 주석 템플릿

```go
// readPump : WebSocket에서 메시지를 읽어서 Hub로 전달하는 고루틴
//
// 동작:
// 1. WebSocket에서 메시지 읽기 (무한 루프)
// 2. JSON 파싱
// 3. Hub.broadcast 채널로 전송
//
// 종료 조건:
// - WebSocket 연결 끊김 (ReadMessage 에러)
// - defer로 자동 정리 (Hub에서 unregister, 연결 Close)
//
// Timeout:
// - pongWait (60초) 안에 Pong 응답 없으면 종료
// - Ping은 writePump가 전송
func (c *Client) readPump() {
    defer func() {
        // 정리 작업...
    }()

    // 타임아웃 설정
    c.conn.SetReadDeadline(time.Now().Add(pongWait))

    // ...
}
```

---

## Go 동시성 개념 비유 모음

### 1. Goroutine

**비유: 직원**
```
main 함수 = 사장
goroutine = 직원

사장이 직원에게 일을 시키고(go func())
본인은 다른 일을 계속함
직원이 끝날 때까지 기다리려면 WaitGroup 사용
```

**예시:**
```go
// 사장: "이 일 좀 해줘" (고루틴 시작)
go processData()

// 사장은 기다리지 않고 바로 다음 일 진행
doOtherWork()
```

### 2. Channel (Unbuffered)

**비유: 직접 손으로 물건 전달**
```
보내는 사람: "여기 물건이야" (손에 들고 대기)
받는 사람: "받았어" (손으로 받음)
→ 둘 다 동시에 있어야 전달 완료
```

**예시:**
```go
ch := make(chan int)  // unbuffered

// Goroutine 1 (보내는 사람)
go func() {
    ch <- 42  // 받는 사람 나타날 때까지 여기서 대기
    fmt.Println("전달 완료!")
}()

// Goroutine 2 (받는 사람)
value := <-ch  // 보내는 사람 나타날 때까지 여기서 대기
fmt.Println("받음:", value)
```

### 3. Channel (Buffered)

**비유: 컨베이어 벨트**
```
보내는 사람: 컨베이어 벨트에 물건 올림 (벨트가 가득 차지 않으면 계속 올릴 수 있음)
받는 사람: 컨베이어 벨트에서 물건 가져감
→ 둘의 속도가 다를 수 있음
```

**예시:**
```go
ch := make(chan int, 3)  // 버퍼 크기 3

// 받는 사람 없어도 3개까지 보낼 수 있음
ch <- 1  // OK
ch <- 2  // OK
ch <- 3  // OK
ch <- 4  // 블록! (버퍼 가득 참)

// 받으면 공간 생김
<-ch  // 1 받음
ch <- 4  // 이제 OK
```

### 4. Mutex

**비유: 화장실 자물쇠**
```
Lock:   들어가서 문 잠금
작업:   화장실 사용
Unlock: 나오면서 문 열기

다른 사람: 문 잠겨있으면 기다림
```

**예시:**
```go
var (
    counter int
    mu      sync.Mutex
)

// 여러 고루틴이 동시에 실행
mu.Lock()           // 문 잠금
counter++           // 안전하게 증가
mu.Unlock()         // 문 열기
```

### 5. RWMutex

**비유: 도서관**
```
RLock:  여러 명이 동시에 책 읽기 가능
RUnlock: 읽기 끝

Lock:   한 명만 책 수정 가능 (다른 사람 모두 나가야 함)
Unlock: 수정 끝
```

**예시:**
```go
var (
    data []int
    mu   sync.RWMutex
)

// 여러 고루틴이 동시에 읽기 OK
mu.RLock()
sum := 0
for _, v := range data {
    sum += v
}
mu.RUnlock()

// 쓰기는 독점 (다른 모든 고루틴 대기)
mu.Lock()
data = append(data, 42)
mu.Unlock()
```

### 6. Select

**비유: 여러 창구 대기**
```
은행에 창구가 3개
어느 창구든 먼저 열리면 그쪽으로 감
```

**예시:**
```go
select {
case msg := <-ch1:
    // ch1에서 메시지 왔을 때
case msg := <-ch2:
    // ch2에서 메시지 왔을 때
case <-timeout:
    // 타임아웃
default:
    // 어느 채널도 준비 안 됐을 때 (non-blocking)
}
```

### 7. WaitGroup

**비유: 출석부**
```
Add(n):  "n명 일 시작"
Done():  "1명 끝남" (완료 체크)
Wait():  "모두 끝날 때까지 대기"
```

**예시:**
```go
var wg sync.WaitGroup

// 3명에게 일 시킴
for i := 0; i < 3; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()  // "끝났어요!"
        doWork(id)
    }(i)
}

wg.Wait()  // 3명 모두 끝날 때까지 대기
fmt.Println("모두 완료!")
```

---

## 효과적인 질문 방법

### ❌ 나쁜 질문

```
"리시버가 뭐에요?"
"에러가 나요."
"작동 안 해요."
```

**문제점:**
- 맥락 없음
- 무엇을 알고 무엇을 모르는지 불명확
- 시행착오 내용 없음

### ✅ 좋은 질문

```
"리시버를 사용하는 이유를 알겠는데,
왜 (ps *ProxyServer)처럼 포인터로 받나요?
값으로 받으면 안 되나요?"
```

```
"upgrader.Upgrade() 실행 시
'websocket: request origin not allowed' 에러가 나는데,
CheckOrigin 함수는 true를 반환하도록 했습니다.
왜 이런 에러가 나는 걸까요?

시도한 것:
1. CheckOrigin 로그 추가 → 호출 안 됨
2. upgrader 설정 확인 → CheckOrigin 설정됨
3. 브라우저 콘솔 → CORS 에러 없음
"
```

**좋은 질문 체크리스트:**
- [ ] 무엇을 하려고 했는지 명확
- [ ] 무엇을 시도했는지 나열
- [ ] 에러 메시지 전체 포함
- [ ] 이해한 부분과 헷갈리는 부분 구분

---

## 디버깅 전략 (from AGENTS.md)

### 1. VSCode 디버거 활용 (가장 효과적!)

```
F5 또는 "Run and Debug" 클릭

브레이크포인트 설정:
- 코드 라인 왼쪽 빨간 점 클릭

단계별 실행:
- F10: Step Over (함수를 한 번에)
- F11: Step Into (함수 내부로)
- Shift+F11: Step Out (함수에서 나오기)

확인할 사항:
- 변수 값이 어떻게 변하는지
- 함수 호출 스택 (Call Stack)
- 고루틴이 언제 생성/종료되는지
- 채널에 데이터가 언제 전송/수신되는지
```

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

```bash
cd /tmp
go mod init test
# 위 코드를 test_receiver.go로 저장
go run test_receiver.go
```

### 3. 로깅으로 흐름 추적

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

**실행 후 로그 순서 확인:**
```
[DEBUG] 1. Entering handleWebSocket
[DEBUG] 2. username=mark
[DEBUG] 4. Upgrade success, clientConn=0xc0001a2000
[DEBUG] 5. Closing connection
```

### 4. 공식 문서와 병행 학습

| 학습 자료 | 용도 | 링크 |
|----------|------|------|
| Go Tour | 대화형 기초 문법 | https://go.dev/tour/ |
| Go by Example | 실용적인 예제 | https://gobyexample.com/ |
| Effective Go | Go 관용구와 패턴 | https://go.dev/doc/effective_go |
| Go Blog | 고급 주제 | https://go.dev/blog/pipelines |

**추천 학습 순서:**
1. 개념이 헷갈림 → Go Tour에서 해당 챕터
2. 예제 코드 작성 → Go by Example 참고
3. 디버거로 실행 → 변수/흐름 확인
4. 실제 프로젝트에 적용 → 이 프로젝트에서 실습

---

## 설명 템플릿 모음

### Template 1: 구조체 설명

```
**구조체 이름: [이름]**

**역할:** [한 문장 설명]

**필드:**
- `필드명 타입`: [무엇을 저장하는지, 왜 이 타입인지]
- `필드명 타입`: [설명]

**왜 이렇게 설계했나:**
[설계 이유, 대안과의 비교]

**사용 예시:**
[코드 예제]
```

**예시:**
```
**구조체 이름: Hub**

**역할:** 채팅방 내 모든 클라이언트를 관리하고 메시지를 브로드캐스트

**필드:**
- `clients map[*Client]bool`: 연결된 클라이언트 집합 (map을 set처럼 사용, O(1) 조회/삭제)
- `broadcast chan *Message`: 브로드캐스트할 메시지 (unbuffered, 즉시 처리)
- `register chan *Client`: 새 클라이언트 등록 요청
- `unregister chan *Client`: 클라이언트 해제 요청

**왜 이렇게 설계했나:**
- 채널 사용 → 고루틴 간 안전한 통신
- map을 Hub.run() 고루틴만 접근 → mutex 불필요
- 단순하고 효율적

**사용 예시:**
hub := newHub()
go hub.run()
hub.register <- client
```

### Template 2: 함수 설명

```
**함수: [이름]**

**입력:** [파라미터 설명]
**출력:** [반환값 설명]

**하는 일:**
1. [단계 1]
2. [단계 2]
3. [단계 3]

**예제:**
[코드 + 결과]

**주의사항:**
- [주의할 점]
```

### Template 3: 패턴 설명

```
**패턴: [이름]**

**문제:** [이 패턴이 해결하는 문제]
**해결:** [어떻게 해결하는지]

**구현:**
[코드 예제]

**언제 사용:**
[사용 시나리오]

**대안:**
[다른 방법과 비교]
```

---

## Quick Tips

### 복잡한 코드 이해하기

1. **큰 그림부터:** 전체 흐름 파악
2. **작은 조각으로:** 함수 하나씩 이해
3. **실행해보기:** 디버거로 단계별 확인
4. **그림 그리기:** 고루틴/채널 흐름도
5. **질문하기:** 모르는 부분 명확히

### 에러 메시지 읽기

```
panic: send on closed channel

고루틴 추적:
goroutine 19 [running]:
main.(*Client).writePump(...)
    /path/to/client.go:95

해석:
- "send on closed channel": 닫힌 채널에 전송 시도
- goroutine 19: 19번 고루틴에서 발생
- client.go:95: 문제 발생 위치
```

**대응:**
1. 파일:라인으로 이동 (client.go:95)
2. 채널이 어디서 닫혔는지 확인
3. 채널 닫기 전 전송 완료 보장

---

## Related Skills

- **[go-development-guidelines](../go-development-guidelines/skill.md)** - Go 베스트 프랙티스
- **[websocket-webrtc-patterns](../websocket-webrtc-patterns/skill.md)** - 실시간 통신 패턴

---

## Remember

**학습은 과정입니다:**
- ❌ "몰라서 부끄러워" → 질문 안 함
- ✅ "모르는 걸 알아가는 중" → 적극적으로 질문

**좋은 질문이 좋은 답을 만듭니다:**
- 구체적으로 질문하기
- 시도한 것 공유하기
- 이해한 부분 명시하기

**천천히, 확실하게:**
- 코드 복사/붙여넣기보다 타이핑
- 디버거로 한 줄씩 실행
- 변수 값 확인하며 흐름 이해

---

**이 스킬은 여러분의 학습을 돕기 위해 존재합니다. 주저하지 말고 질문하세요!**
