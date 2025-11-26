## [deepwiki docs] (https://deepwiki.com/MarkSon-42/minigo-chat-system)

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![WebSocket](https://img.shields.io/badge/WebSocket-Gorilla-blue)](https://github.com/gorilla/websocket)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

A learning-oriented real-time chat application built in Go, designed to demonstrate practical implementations of WebSocket protocols, concurrent programming patterns, and proxy-based architectures.

## 🎯 Purpose

This project provides hands-on experience with:

- **WebSocket Protocol**: Real-time bidirectional communication using `gorilla/websocket`
- **Go Concurrency**: Practical use of goroutines, channels, and mutexes
- **Proxy Architecture**: Security boundary implementation with filtering and logging
- **Hub-and-Spoke Pattern**: Efficient message distribution to multiple clients

> **Note**: This is explicitly a learning project, not production-ready. The focus is on understanding Go concurrency patterns rather than performance optimization.

## 🏗️ Architecture

The system implements a three-tier architecture with clear separation of concerns:

```
┌─────────────┐
│   Browser   │  (Client Layer - Event-driven UI)
└──────┬──────┘
       │ WebSocket (:8081)
       ▼
┌─────────────────────┐
│   Proxy Service     │  (Security boundary, filtering, logging)
│   Port: 8081        │
│                     │
│  ┌───────────────┐  │
│  │ Filter Engine │  │
│  │ Message Queue │  │
│  │ Storage Logger│  │
│  └───────────────┘  │
└──────────┬──────────┘
           │ WebSocket (:8080)
           ▼
┌─────────────────────┐
│  Backend Service    │  (Core chat functionality)
│  Port: 8080         │
│                     │
│  ┌───────────────┐  │
│  │  Hub Pattern  │  │
│  │  Broadcasting │  │
│  │  Room Manager │  │
│  └───────────────┘  │
└─────────────────────┘
```

### Layer Responsibilities

| Layer | Port | Primary Responsibility | Key Pattern |
|-------|------|------------------------|-------------|
| **Proxy Service** | 8081 | Security boundary, filtering, logging, rate limiting | Gateway Pattern |
| **Backend Service** | 8080 | Core chat functionality, client management | Hub-and-Spoke Pattern |
| **Client Layer** | Browser | User interface, WebSocket client implementation | Event-driven UI |

## 📁 Repository Structure

```
minigo-chat-system/
├── backend/                # Backend chat server (port 8080)
│   ├── main.go            # Server entry point, serveWs handler
│   ├── hub.go             # Hub pattern implementation
│   ├── client.go          # Client connection management
│   ├── message.go         # Message data structure
│   └── *_test.go          # Backend unit tests
│
├── proxy/                 # Proxy service (port 8081)
│   ├── main.go            # Proxy server entry point
│   ├── proxy.go           # Bidirectional WebSocket relay
│   ├── filter.go          # Message filtering engine
│   ├── queue.go           # Message queue implementation
│   ├── storage.go         # Message logging system
│   ├── api.go             # Filter management REST API
│   └── *_test.go          # Proxy unit tests
│
├── frontend/              # Client-side applications
│   ├── index.html         # Chat client UI
│   ├── chat.js            # WebSocket client logic
│   ├── admin.html         # Admin dashboard UI
│   ├── admin.js           # Admin API client
│   └── test-client.html   # WebSocket test client
│
├── logs/                  # Runtime logs
│   └── messages.jsonl     # Message audit trail (JSON Lines)
│
├── go.mod                 # Go module definition
├── go.sum                 # Dependency checksums
├── AGENTS.md              # Development guidelines
├── run_tests.sh           # Test orchestration script
└── setup_tests.sh         # Test environment setup
```

## 🚀 Getting Started

### Prerequisites

- Go 1.21 or higher
- Modern web browser with WebSocket support

### Installation

1. Clone the repository:
```bash
git clone https://github.com/MarkSon-42/minigo-chat-system.git
cd minigo-chat-system
```

2. Initialize Go modules:
```bash
go mod init
go mod tidy
```

### Running the System

1. **Start the Backend Service** (Terminal 1):
```bash
go run backend/*.go
```
Server starts on `http://localhost:8080`

2. **Start the Proxy Service** (Terminal 2):
```bash
go run proxy/*.go
```
Proxy starts on `http://localhost:8081`

3. **Open the Chat Client**:
```bash
# Open in your browser
open frontend/index.html
```

### Testing

Run all unit tests:
```bash
./run_tests.sh quick
```

Run with race detection:
```bash
go test -race ./...
```

## 🔑 Key Features

### Real-time Communication
- WebSocket-based bidirectional communication
- Multiple concurrent users support
- Room-based message isolation

### Message Processing
- Content filtering and moderation
- Message queuing with backpressure handling
- Persistent audit logging (JSON Lines format)

### Administration
- REST API for filter rule management
- Health check endpoints
- Runtime statistics monitoring

### HTTP Endpoints

#### Proxy Service (8081)
| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/ws` | WebSocket | Client WebSocket connections |
| `/health` | GET | Health check (returns "OK") |
| `/stats` | GET | Service statistics (JSON) |
| `/filter/rules` | GET/POST | Filter rule management |

#### Backend Service (8080)
| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/ws` | WebSocket | Backend WebSocket connections |

## 💡 Learning Objectives

### 1. Hub-and-Spoke Pattern
The Hub runs in a single goroutine processing three event types:
- `register chan *Client` - Client connection events
- `unregister chan *Client` - Client disconnection events
- `broadcast chan *Message` - Message distribution events

This pattern serializes access to shared state, eliminating race conditions.

### 2. Per-Connection Goroutine Pairs
Each client spawns two goroutines:
- `readPump()` - Reads from WebSocket, sends to Hub
- `writePump()` - Reads from send channel, writes to WebSocket

This separation prevents concurrent read/write operations on the same connection.

### 3. Channel-Based Message Queue
Uses buffered Go channels for asynchronous message processing:
- `Enqueue()` with timeout prevents sender blocking
- `Dequeue()` with timeout prevents receiver blocking
- Buffer capacity (1000) provides backpressure handling

### 4. Mutex-Protected Shared State
- **Filter**: Uses `sync.RWMutex` for concurrent rule access
  - Multiple readers can check rules simultaneously (`RLock()`)
  - Writers block all readers during updates (`Lock()`)
- **Storage**: Uses `sync.Mutex` for file operations
  - Critical section protection prevents deadlocks

## 📊 Core Data Structures

### Hub
```go
type Hub struct {
    clients    map[*Client]bool  // Active client registry
    broadcast  chan *Message     // Message distribution channel
    register   chan *Client      // Client registration channel
    unregister chan *Client      // Client cleanup channel
}
```

### Client
```go
type Client struct {
    hub      *Hub              // Reference to Hub
    conn     *websocket.Conn   // WebSocket connection
    send     chan []byte       // Outbound message buffer (capacity 256)
    username string            // User identifier
    room     string            // Room identifier
}
```

### Message
```go
type Message struct {
    Type      string    // Message type (chat, join, leave)
    Username  string    // Sender username
    Room      string    // Target room
    Content   string    // Message payload
    Timestamp time.Time // Server timestamp
}
```

## 🔍 Message Flow

1. **Client → Proxy**: WebSocket connection upgrade at `:8081/ws`
2. **Proxy Filtering**: `filter.CheckMessage()` validates content
3. **Queue Buffering**: `Enqueue()` with 5-second timeout
4. **Storage Logging**: `LogMessage()` appends to JSON Lines file
5. **Proxy → Backend**: Forwarded to `:8080/ws`
6. **Hub Broadcasting**: Distributed to all clients in the same room
7. **Backend → Clients**: Messages sent through individual client channels

## 🎓 Project Goals

| Goal Category | Specific Goals |
|---------------|----------------|
| **Learning** | Demonstrate Go concurrency patterns (goroutines, channels, mutexes) |
| | Illustrate WebSocket protocol implementation |
| | Show proxy architecture patterns for security and logging |
| **Functional** | Real-time bidirectional chat with multiple concurrent users |
| | Room-based message isolation |
| | Message filtering and content moderation |
| **Operational** | Message audit logging to persistent storage |
| | Runtime filter rule management via REST API |
| | Health monitoring and statistics endpoints |
| **Quality** | Race condition detection through Go's race detector |
| | Deadlock prevention through proper mutex usage |
| | Memory leak prevention through connection cleanup |


## 📝 License

This project is licensed under the MIT License.

## 👨‍💻 Author

**Mark Son**
- GitHub: [@MarkSon-42](https://github.com/MarkSon-42)

## 📚 Additional Resources

- [Gorilla WebSocket Documentation](https://github.com/gorilla/websocket)
- [Go Concurrency Patterns](https://go.dev/blog/pipelines)
- [Effective Go](https://go.dev/doc/effective_go)

---


## KR ver

# minigo-chat-system

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![WebSocket](https://img.shields.io/badge/WebSocket-Gorilla-blue)](https://github.com/gorilla/websocket)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

WebSocket 프로토콜, 동시성 프로그래밍 패턴, 프록시 기반 아키텍처의 실용적인 구현을 보여주기 위해 Go로 작성된 학습 지향적 실시간 채팅 애플리케이션입니다.

## 🎯 목적

이 프로젝트는 다음의 실습 경험을 제공합니다:

- **WebSocket 프로토콜**: `gorilla/websocket`을 사용한 실시간 양방향 통신
- **Go 동시성**: 고루틴, 채널, 뮤텍스의 실용적 활용
- **프록시 아키텍처**: 필터링 및 로깅을 포함한 보안 경계 구현
- **허브-스포크 패턴**: 여러 클라이언트로의 효율적인 메시지 배포

> **참고**: 이것은 명시적으로 학습 프로젝트이며, 프로덕션 환경에 적합하지 않습니다. 성능 최적화보다는 Go 동시성 패턴의 이해에 초점을 맞추고 있습니다.

## 🏗️ 아키텍처

시스템은 명확한 관심사 분리를 통한 3계층 아키텍처를 구현합니다:

```
┌─────────────┐
│   브라우저   │  (클라이언트 계층 - 이벤트 기반 UI)
└──────┬──────┘
       │ WebSocket (:8081)
       ▼
┌─────────────────────┐
│   프록시 서비스      │  (보안 경계, 필터링, 로깅)
│   포트: 8081        │
│                     │
│  ┌───────────────┐  │
│  │ 필터 엔진     │  │
│  │ 메시지 큐     │  │
│  │ 스토리지 로거 │  │
│  └───────────────┘  │
└──────────┬──────────┘
           │ WebSocket (:8080)
           ▼
┌─────────────────────┐
│  백엔드 서비스       │  (핵심 채팅 기능)
│  포트: 8080         │
│                     │
│  ┌───────────────┐  │
│  │  허브 패턴    │  │
│  │  브로드캐스팅 │  │
│  │  룸 관리자    │  │
│  └───────────────┘  │
└─────────────────────┘
```

### 계층별 책임

| 계층 | 포트 | 주요 책임 | 핵심 패턴 |
|------|------|-----------|-----------|
| **프록시 서비스** | 8081 | 보안 경계, 필터링, 로깅, 속도 제한 | 게이트웨이 패턴 |
| **백엔드 서비스** | 8080 | 핵심 채팅 기능, 클라이언트 관리 | 허브-스포크 패턴 |
| **클라이언트 계층** | 브라우저 | 사용자 인터페이스, WebSocket 클라이언트 구현 | 이벤트 기반 UI |

## 📁 레포지토리 구조

```
minigo-chat-system/
├── backend/                # 백엔드 채팅 서버 (포트 8080)
│   ├── main.go            # 서버 진입점, serveWs 핸들러
│   ├── hub.go             # 허브 패턴 구현
│   ├── client.go          # 클라이언트 연결 관리
│   ├── message.go         # 메시지 데이터 구조
│   └── *_test.go          # 백엔드 단위 테스트
│
├── proxy/                 # 프록시 서비스 (포트 8081)
│   ├── main.go            # 프록시 서버 진입점
│   ├── proxy.go           # 양방향 WebSocket 릴레이
│   ├── filter.go          # 메시지 필터링 엔진
│   ├── queue.go           # 메시지 큐 구현
│   ├── storage.go         # 메시지 로깅 시스템
│   ├── api.go             # 필터 관리 REST API
│   └── *_test.go          # 프록시 단위 테스트
│
├── frontend/              # 클라이언트 사이드 애플리케이션
│   ├── index.html         # 채팅 클라이언트 UI
│   ├── chat.js            # WebSocket 클라이언트 로직
│   ├── admin.html         # 관리자 대시보드 UI
│   ├── admin.js           # 관리자 API 클라이언트
│   └── test-client.html   # WebSocket 테스트 클라이언트
│
├── logs/                  # 런타임 로그
│   └── messages.jsonl     # 메시지 감사 추적 (JSON Lines)
│
├── go.mod                 # Go 모듈 정의
├── go.sum                 # 의존성 체크섬
├── AGENTS.md              # 개발 가이드라인
├── run_tests.sh           # 테스트 오케스트레이션 스크립트
└── setup_tests.sh         # 테스트 환경 설정
```

## 🚀 시작하기

### 필수 요구사항

- Go 1.21 이상
- WebSocket을 지원하는 최신 웹 브라우저

### 설치

1. 레포지토리 복제:
```bash
git clone https://github.com/MarkSon-42/minigo-chat-system.git
cd minigo-chat-system
```

2. Go 모듈 초기화:
```bash
go mod init
go mod tidy
```

### 시스템 실행

1. **백엔드 서비스 시작** (터미널 1):
```bash
go run backend/*.go
```
서버가 `http://localhost:8080`에서 시작됩니다

2. **프록시 서비스 시작** (터미널 2):
```bash
go run proxy/*.go
```
프록시가 `http://localhost:8081`에서 시작됩니다

3. **채팅 클라이언트 열기**:
```bash
# 브라우저에서 열기
open frontend/index.html
```

### 테스트

모든 단위 테스트 실행:
```bash
./run_tests.sh quick
```

레이스 감지와 함께 실행:
```bash
go test -race ./...
```

## 🔑 주요 기능

### 실시간 통신
- WebSocket 기반 양방향 통신
- 다중 동시 사용자 지원
- 룸 기반 메시지 격리

### 메시지 처리
- 콘텐츠 필터링 및 조정
- 백프레셔 처리를 포함한 메시지 큐잉
- 영구 감사 로깅 (JSON Lines 형식)

### 관리
- 필터 규칙 관리를 위한 REST API
- 헬스 체크 엔드포인트
- 런타임 통계 모니터링

### HTTP 엔드포인트

#### 프록시 서비스 (8081)
| 엔드포인트 | 메서드 | 목적 |
|-----------|--------|------|
| `/ws` | WebSocket | 클라이언트 WebSocket 연결 |
| `/health` | GET | 헬스 체크 ("OK" 반환) |
| `/stats` | GET | 서비스 통계 (JSON) |
| `/filter/rules` | GET/POST | 필터 규칙 관리 |

#### 백엔드 서비스 (8080)
| 엔드포인트 | 메서드 | 목적 |
|-----------|--------|------|
| `/ws` | WebSocket | 백엔드 WebSocket 연결 |

## 💡 학습 목표

### 1. 허브-스포크 패턴
허브는 단일 고루틴에서 세 가지 이벤트 유형을 처리합니다:
- `register chan *Client` - 클라이언트 연결 이벤트
- `unregister chan *Client` - 클라이언트 연결 해제 이벤트
- `broadcast chan *Message` - 메시지 배포 이벤트

이 패턴은 공유 상태에 대한 접근을 직렬화하여 레이스 컨디션을 제거합니다.

### 2. 연결당 고루틴 쌍
각 클라이언트는 두 개의 고루틴을 생성합니다:
- `readPump()` - WebSocket에서 읽고, 허브로 전송
- `writePump()` - send 채널에서 읽고, WebSocket에 작성

이 분리는 동일한 연결에서 동시 읽기/쓰기 작업을 방지합니다.

### 3. 채널 기반 메시지 큐
비동기 메시지 처리를 위해 버퍼링된 Go 채널을 사용합니다:
- `Enqueue()` 타임아웃으로 발신자 차단 방지
- `Dequeue()` 타임아웃으로 수신자 차단 방지
- 버퍼 용량 (1000)으로 백프레셔 처리 제공

### 4. 뮤텍스로 보호되는 공유 상태
- **필터**: 동시 규칙 접근을 위해 `sync.RWMutex` 사용
  - 여러 reader가 동시에 규칙을 확인할 수 있음 (`RLock()`)
  - writer는 업데이트 중 모든 reader를 차단 (`Lock()`)
- **스토리지**: 파일 작업을 위해 `sync.Mutex` 사용
  - 임계 영역 보호로 데드락 방지


## 📊 핵심 데이터 구조

### Hub
```go
type Hub struct {
    clients    map[*Client]bool  // 활성 클라이언트 레지스트리
    broadcast  chan *Message     // 메시지 배포 채널
    register   chan *Client      // 클라이언트 등록 채널
    unregister chan *Client      // 클라이언트 정리 채널
}
```

### Client
```go
type Client struct {
    hub      *Hub              // 허브 참조
    conn     *websocket.Conn   // WebSocket 연결
    send     chan []byte       // 아웃바운드 메시지 버퍼 (용량 256)
    username string            // 사용자 식별자
    room     string            // 룸 식별자
}
```

### Message
```go
type Message struct {
    Type      string    // 메시지 유형 (chat, join, leave)
    Username  string    // 발신자 사용자명
    Room      string    // 대상 룸
    Content   string    // 메시지 페이로드
    Timestamp time.Time // 서버 타임스탬프
}
```

## 🔍 메시지 흐름

1. **클라이언트 → 프록시**: `:8081/ws`에서 WebSocket 연결 업그레이드
2. **프록시 필터링**: `filter.CheckMessage()`로 콘텐츠 검증
3. **큐 버퍼링**: 5초 타임아웃과 함께 `Enqueue()`
4. **스토리지 로깅**: JSON Lines 파일에 `LogMessage()` 추가
5. **프록시 → 백엔드**: `:8080/ws`로 전달
6. **허브 브로드캐스팅**: 같은 룸의 모든 클라이언트에게 배포
7. **백엔드 → 클라이언트**: 개별 클라이언트 채널을 통해 메시지 전송

## 🎓 프로젝트 목표

| 목표 범주 | 구체적 목표 |
|----------|------------|
| **학습** | Go 동시성 패턴 시연 (고루틴, 채널, 뮤텍스) |
| | WebSocket 프로토콜 구현 설명 |
| | 보안 및 로깅을 위한 프록시 아키텍처 패턴 표시 |
| **기능** | 다중 동시 사용자와의 실시간 양방향 채팅 |
| | 룸 기반 메시지 격리 |
| | 메시지 필터링 및 콘텐츠 조정 |
| **운영** | 영구 스토리지에 메시지 감사 로깅 |
| | REST API를 통한 런타임 필터 규칙 관리 |
| | 헬스 모니터링 및 통계 엔드포인트 |
| **품질** | Go의 레이스 감지기를 통한 레이스 컨디션 감지 |
| | 적절한 뮤텍스 사용을 통한 데드락 방지 |
| | 연결 정리를 통한 메모리 누수 방지 |

## 📝 라이선스

이 프로젝트는 MIT 라이선스에 따라 라이선스가 부여됩니다.

## 👨‍💻 작성자

**Mark Son**
- GitHub: [@MarkSon-42](https://github.com/MarkSon-42)

## 📚 추가 리소스

- [Gorilla WebSocket 문서](https://github.com/gorilla/websocket)
- [Go 동시성 패턴](https://go.dev/blog/pipelines)
- [Effective Go](https://go.dev/doc/effective_go)

---

**참고**: 이 프로젝트는 프로덕션 준비 기능보다 명확성과 교육적 가치를 우선시합니다. Go 동시성 및 WebSocket 패턴 학습을 위한 참고 자료로 사용하세요.
