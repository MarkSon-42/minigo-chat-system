# Testing Strategies (테스팅 전략)

> Minigo 채팅 시스템의 효과적인 테스트 전략

---

## Table of Contents
1. [Test File Organization](#test-file-organization)
2. [Race Detection](#race-detection)
3. [Table-Driven Tests](#table-driven-tests)
4. [Coverage Analysis](#coverage-analysis)
5. [Benchmarking](#benchmarking)
6. [Integration Testing](#integration-testing)

---

## Test File Organization

### 네이밍 규칙

```
filter.go       → filter_test.go
queue.go        → queue_test.go
storage.go      → storage_test.go
```

### 기본 구조

```go
package main

import "testing"

func TestFilterCheckMessage(t *testing.T) {
    // 1. Setup (준비)
    filter := NewFilter()
    filter.AddRule(FilterRule{
        Keywords: []string{"badword"},
        Action:   "block",
        Enabled:  true,
    })

    // 2. Execute (실행)
    allowed, _ := filter.CheckMessage(&Message{
        Type:    "message",
        Content: "This has badword in it",
    })

    // 3. Assert (검증)
    if allowed {
        t.Error("Expected message to be blocked")
    }
}
```

---

## Race Detection

### 왜 중요한가?

**Minigo는 동시성이 핵심입니다:**
- Hub가 여러 클라이언트 동시 관리
- Filter가 여러 고루틴에서 동시 호출
- Queue에 여러 고루틴이 push/pop

### 실행 방법

```bash
# 전체 테스트
go test -race ./...

# 특정 패키지
cd proxy
go test -race

# Verbose 모드
go test -race -v

# 타임아웃 설정 (긴 테스트)
go test -race -timeout 30s ./...
```

### Race Condition 예시

**테스트 코드: `proxy/filter_test.go`**

```go
func TestConcurrentFilterAccess(t *testing.T) {
    t.Parallel()  // 병렬 실행 가능

    filter := NewFilter()
    var wg sync.WaitGroup

    // 100개 고루틴이 동시에 규칙 추가
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(n int) {
            defer wg.Done()
            filter.AddRule(FilterRule{
                Keywords: []string{fmt.Sprintf("word%d", n)},
                Action:   "block",
                Enabled:  true,
            })
        }(i)
    }

    // 동시에 메시지 체크
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(n int) {
            defer wg.Done()
            filter.CheckMessage(&Message{
                Type:    "message",
                Content: fmt.Sprintf("test message %d", n),
            })
        }(i)
    }

    wg.Wait()

    // race detector가 문제 발견하면 테스트 실패
    rules := filter.GetRules()
    if len(rules) < 2 {  // 기본 규칙 2개 + 추가된 규칙들
        t.Errorf("Expected more than 2 rules, got %d", len(rules))
    }
}
```

**Race Detector 출력 예시:**
```
==================
WARNING: DATA RACE
Write at 0x00c0001a2000 by goroutine 7:
  main.(*Filter).AddRule()
      /path/to/filter.go:85 +0x12c

Previous read at 0x00c0001a2000 by goroutine 8:
  main.(*Filter).CheckMessage()
      /path/to/filter.go:39 +0x5c

Goroutine 7 (running) created at:
  main.TestConcurrentFilterAccess()
      /path/to/filter_test.go:45 +0x1ab
==================
```

### 실제 프로젝트에서 Race 찾기

```bash
# 프로젝트 스크립트 사용
./run_tests.sh race

# 또는 각 모듈별로
cd backend && go test -race ./...
cd proxy && go test -race ./...
cd voice && go test -race ./...
```

---

## Table-Driven Tests

### 개념

**여러 입력에 대해 동일한 테스트 로직 반복**

### 예시: Filter 테스트

```go
func TestFilterCheckMessage(t *testing.T) {
    filter := NewFilter()  // 기본 규칙: badword, spam, 욕설

    tests := []struct {
        name    string
        message Message
        want    bool  // true면 통과, false면 차단
    }{
        {
            name:    "clean message",
            message: Message{Type: "message", Content: "Hello world"},
            want:    true,
        },
        {
            name:    "contains badword",
            message: Message{Type: "message", Content: "This is badword"},
            want:    false,
        },
        {
            name:    "contains spam",
            message: Message{Type: "message", Content: "Click here spam"},
            want:    false,
        },
        {
            name:    "contains 욕설",
            message: Message{Type: "message", Content: "너무 욕설이네"},
            want:    false,
        },
        {
            name:    "empty message",
            message: Message{Type: "message", Content: "   "},
            want:    false,  // 빈 메시지도 차단
        },
        {
            name:    "non-message type",
            message: Message{Type: "join", Content: "badword"},
            want:    true,  // message 타입만 필터링
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            allowed, _ := filter.CheckMessage(&tt.message)

            if allowed != tt.want {
                t.Errorf("%s: got allowed=%v, want %v",
                    tt.name, allowed, tt.want)
            }
        })
    }
}
```

**실행 결과:**
```
=== RUN   TestFilterCheckMessage
=== RUN   TestFilterCheckMessage/clean_message
=== RUN   TestFilterCheckMessage/contains_badword
=== RUN   TestFilterCheckMessage/contains_spam
=== RUN   TestFilterCheckMessage/contains_욕설
=== RUN   TestFilterCheckMessage/empty_message
=== RUN   TestFilterCheckMessage/non-message_type
--- PASS: TestFilterCheckMessage (0.00s)
    --- PASS: TestFilterCheckMessage/clean_message (0.00s)
    --- PASS: TestFilterCheckMessage/contains_badword (0.00s)
    --- PASS: TestFilterCheckMessage/contains_spam (0.00s)
    --- PASS: TestFilterCheckMessage/contains_욕설 (0.00s)
    --- PASS: TestFilterCheckMessage/empty_message (0.00s)
    --- PASS: TestFilterCheckMessage/non-message_type (0.00s)
```

### 장점

- 새로운 테스트 케이스 추가 쉬움 (배열에 추가만)
- 실패한 케이스 이름으로 즉시 파악
- 코드 중복 최소화

---

## Coverage Analysis

### 커버리지 측정

```bash
# 커버리지 프로파일 생성
go test -coverprofile=coverage.out ./...

# 결과 보기 (터미널)
go tool cover -func=coverage.out

# 결과 보기 (브라우저 - 색상으로 표시)
go tool cover -html=coverage.out
```

### 출력 예시

```
main.go:18:         newHub          100.0%
main.go:27:         run             85.7%
filter.go:20:       NewFilter       100.0%
filter.go:39:       CheckMessage    92.3%
filter.go:85:       AddRule         100.0%
filter.go:101:      RemoveRule      75.0%
total:              (statements)    87.5%
```

### HTML 리포트

브라우저에서 열면:
- ✅ 녹색: 실행된 코드
- ❌ 빨강: 실행 안 된 코드
- ⚪ 회색: 커버리지 불가능 (주석 등)

### 목표 커버리지

```
권장:
- 비즈니스 로직: 80% 이상
- 유틸리티 함수: 90% 이상
- 에러 핸들링: 70% 이상 (모든 에러 경로 테스트 어려움)
```

### 프로젝트 스크립트

```bash
./run_tests.sh coverage
```

---

## Benchmarking

### 기본 구조

```go
func BenchmarkFilterCheck(b *testing.B) {
    filter := NewFilter()
    message := &Message{
        Type:    "message",
        Content: "This is a clean message",
    }

    b.ResetTimer()  // 셋업 시간 제외

    for i := 0; i < b.N; i++ {
        filter.CheckMessage(message)
    }
}
```

### 실행

```bash
# 벤치마크 실행
go test -bench=.

# 메모리 할당 포함
go test -bench=. -benchmem

# 특정 벤치마크만
go test -bench=BenchmarkFilterCheck

# 실행 시간 증가
go test -bench=. -benchtime=10s
```

### 출력 예시

```
BenchmarkFilterCheck-8           5000000       245 ns/op      48 B/op      2 allocs/op
BenchmarkFilterCheckBlocked-8    3000000       412 ns/op      96 B/op      4 allocs/op
```

**해석:**
- `5000000`: 반복 횟수 (b.N)
- `245 ns/op`: 평균 실행 시간 (나노초)
- `48 B/op`: 평균 메모리 할당 (바이트)
- `2 allocs/op`: 평균 할당 횟수

### 벤치마크 비교

```go
// Before: 규칙을 순회하며 체크
func BenchmarkFilterCheckBefore(b *testing.B) {
    filter := setupFilter()
    message := testMessage()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        filter.CheckMessage(message)
    }
}

// After: 규칙을 map으로 최적화
func BenchmarkFilterCheckAfter(b *testing.B) {
    filter := setupOptimizedFilter()
    message := testMessage()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        filter.CheckMessage(message)
    }
}
```

```bash
# 비교
go test -bench=. > old.txt
# 코드 최적화
go test -bench=. > new.txt
benchcmp old.txt new.txt
```

### 프로젝트 스크립트

```bash
./run_tests.sh bench
```

---

## Integration Testing

### 멀티 클라이언트 시나리오

```go
func TestMultiClientChat(t *testing.T) {
    // Hub 시작
    hub := newHub()
    go hub.run()

    // 3명의 클라이언트 시뮬레이션
    clients := make([]*Client, 3)
    for i := 0; i < 3; i++ {
        clients[i] = &Client{
            hub:      hub,
            send:     make(chan []byte, 10),
            username: fmt.Sprintf("user%d", i),
            room:     "testroom",
        }
        hub.register <- clients[i]
    }

    // 메시지 전송
    testMessage := &Message{
        Type:    "message",
        Content: "Hello everyone",
        Room:    "testroom",
    }
    hub.broadcast <- testMessage

    // 모든 클라이언트가 메시지를 받았는지 확인
    timeout := time.After(1 * time.Second)
    for i, client := range clients {
        select {
        case msg := <-client.send:
            var received Message
            if err := json.Unmarshal(msg, &received); err != nil {
                t.Errorf("Client %d: unmarshal error: %v", i, err)
            }
            if received.Content != "Hello everyone" {
                t.Errorf("Client %d: got %q, want %q",
                    i, received.Content, "Hello everyone")
            }
        case <-timeout:
            t.Errorf("Client %d: timeout waiting for message", i)
        }
    }

    // 정리
    for _, client := range clients {
        hub.unregister <- client
    }
}
```

### End-to-End 테스트

```go
func TestProxyFilterIntegration(t *testing.T) {
    // Backend 시작
    backend := startTestBackend(t)
    defer backend.Close()

    // Proxy 시작
    proxy := startTestProxy(t, backend.URL)
    defer proxy.Close()

    // 클라이언트 연결
    wsURL := "ws" + strings.TrimPrefix(proxy.URL, "http") + "/ws"
    ws, _, err := websocket.DefaultDialer.Dial(wsURL+"?username=test&room=test", nil)
    if err != nil {
        t.Fatalf("Failed to connect: %v", err)
    }
    defer ws.Close()

    // badword 포함 메시지 전송
    msg := Message{
        Type:    "message",
        Content: "This has badword in it",
    }
    if err := ws.WriteJSON(msg); err != nil {
        t.Fatalf("Failed to send: %v", err)
    }

    // 차단되어서 응답 없어야 함
    ws.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
    _, _, err = ws.ReadMessage()
    if err == nil {
        t.Error("Expected timeout (message should be blocked)")
    }

    // 정상 메시지 전송
    msg.Content = "Clean message"
    if err := ws.WriteJSON(msg); err != nil {
        t.Fatalf("Failed to send: %v", err)
    }

    // 응답 받아야 함
    ws.SetReadDeadline(time.Now().Add(1 * time.Second))
    _, data, err := ws.ReadMessage()
    if err != nil {
        t.Errorf("Expected message, got error: %v", err)
    }

    var received Message
    if err := json.Unmarshal(data, &received); err != nil {
        t.Errorf("Unmarshal error: %v", err)
    }
    if received.Content != "Clean message" {
        t.Errorf("Got %q, want %q", received.Content, "Clean message")
    }
}
```

---

## Test Helpers

### Setup/Teardown 패턴

```go
func setupTest(t *testing.T) (*Hub, func()) {
    hub := newHub()
    go hub.run()

    // cleanup 함수 반환
    cleanup := func() {
        // Hub 종료 로직 (필요 시)
        t.Log("Cleaning up hub")
    }

    return hub, cleanup
}

func TestWithSetup(t *testing.T) {
    hub, cleanup := setupTest(t)
    defer cleanup()

    // 테스트 로직
    client := &Client{hub: hub, send: make(chan []byte, 1)}
    hub.register <- client

    // ...
}
```

### Test Fixtures

```go
// testdata/messages.json
[
    {"type": "message", "content": "test1"},
    {"type": "message", "content": "test2"}
]

func loadTestMessages(t *testing.T) []Message {
    data, err := os.ReadFile("testdata/messages.json")
    if err != nil {
        t.Fatalf("Failed to load test data: %v", err)
    }

    var messages []Message
    if err := json.Unmarshal(data, &messages); err != nil {
        t.Fatalf("Failed to unmarshal: %v", err)
    }

    return messages
}
```

---

## Minigo Test Scripts

### run_tests.sh 사용법

```bash
# 빠른 테스트 (race detector 없이)
./run_tests.sh quick

# Race 조건 감지
./run_tests.sh race

# 커버리지 리포트
./run_tests.sh coverage

# 벤치마크
./run_tests.sh bench

# 파일 변경 감지 시 자동 테스트 (entr 필요)
./run_tests.sh watch
```

### CI/CD에서 테스트

**`.github/workflows/ci.yml`**

```yaml
name: CI

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Test Backend
        run: cd backend && go test -race ./...

      - name: Test Proxy
        run: cd proxy && go test -race ./...

      - name: Coverage
        run: cd proxy && go test -coverprofile=coverage.out ./...
```

---

## Best Practices

### ✅ DO

1. **테스트 이름은 명확하게**
   ```go
   func TestFilterBlocksBadword(t *testing.T) { ... }  // Good
   func TestFilter(t *testing.T) { ... }                // Bad
   ```

2. **t.Parallel() 사용**
   ```go
   func TestSomething(t *testing.T) {
       t.Parallel()  // 병렬 실행 가능
       // ...
   }
   ```

3. **Table-driven tests로 케이스 추가 쉽게**

4. **항상 race detector 실행**
   ```bash
   go test -race ./...
   ```

5. **에러 메시지는 구체적으로**
   ```go
   if got != want {
       t.Errorf("CheckMessage() = %v, want %v", got, want)
   }
   ```

### ❌ DON'T

1. **테스트에서 time.Sleep() 남용**
   ```go
   // Bad
   go doSomething()
   time.Sleep(1 * time.Second)  // 느리고 불안정

   // Good
   done := make(chan bool)
   go func() {
       doSomething()
       done <- true
   }()
   select {
   case <-done:
       // OK
   case <-time.After(1 * time.Second):
       t.Error("timeout")
   }
   ```

2. **전역 상태에 의존**
   ```go
   // Bad
   var globalHub *Hub

   func TestSomething(t *testing.T) {
       globalHub = newHub()  // 다른 테스트에 영향!
   }

   // Good
   func TestSomething(t *testing.T) {
       hub := newHub()  // 지역 변수
   }
   ```

3. **외부 의존성 (DB, 네트워크) 직접 사용**
   ```go
   // Mock이나 fake 객체 사용
   ```

---

## Quick Reference

### 자주 쓰는 명령어

```bash
# 기본 테스트
go test ./...

# Verbose
go test -v ./...

# 특정 테스트만
go test -run TestFilterCheck

# Race detection
go test -race ./...

# Coverage
go test -coverprofile=coverage.out
go tool cover -html=coverage.out

# Benchmark
go test -bench=. -benchmem

# 모듈별 테스트
cd backend && go test -race ./...
cd proxy && go test -race ./...
```

### 프로젝트 스크립트

```bash
./run_tests.sh quick     # 빠른 테스트
./run_tests.sh race      # Race 감지
./run_tests.sh coverage  # 커버리지
./run_tests.sh bench     # 벤치마크
./run_tests.sh watch     # 자동 테스트
```

---

**다음 학습:**
- [Concurrency Patterns](./concurrency-patterns.md) - 동시성 패턴 심화
- Back to [main skill](../skill.md) - 메인 스킬로 돌아가기
