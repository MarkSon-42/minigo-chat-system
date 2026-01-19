# Pion WebRTC SFU Pattern (프로덕션급 음성 채팅)

> 1만명 규모를 지원하는 확장 가능한 음성 채팅 서버 설계

---

## 목표 규모

**시스템 요구사항:**
- 동시 접속: 1만명
- 음성 사용 패턴: 방별 5-20명 소규모 그룹
- 예상 방 개수: 500-2000개
- 프로덕션급 확장성

**예시:**
```
10,000명 온라인
├─ 500개 방 × 평균 20명 = 10,000명
└─ 각 방은 독립적인 SFU 인스턴스
```

---

## 확장 가능한 아키텍처

### Room 기반 격리

```
┌─────────────────────────────────────────────┐
│         Voice Server (1대)                  │
│                                             │
│  Room "general" (15명)                      │
│  ┌─────────────────────────────────────┐   │
│  │ SFU Instance                        │   │
│  │ - 15명 × 14명 전송 = 210 streams    │   │
│  │ - 격리된 네임스페이스               │   │
│  └─────────────────────────────────────┘   │
│                                             │
│  Room "dev-team" (8명)                      │
│  ┌─────────────────────────────────────┐   │
│  │ SFU Instance                        │   │
│  │ - 8명 × 7명 전송 = 56 streams       │   │
│  └─────────────────────────────────────┘   │
│                                             │
│  ... (수백~수천개 방)                       │
└─────────────────────────────────────────────┘
```

**핵심:**
- 각 방은 완전히 독립적
- Room A의 음성이 Room B로 새지 않음
- 부하 = Σ(각 방의 인원²)

### 대역폭 계산

**방당 대역폭 (20명 음성):**
```
1명당 업로드: ~50 Kbps (Opus 코덱)
1명당 다운로드: 50 Kbps × 19명 = 950 Kbps

서버 부하 (20명 방):
  - 업로드 받기: 20명 × 50 Kbps = 1 Mbps
  - 다운로드 전송: 20명 × 950 Kbps = 19 Mbps
```

**전체 서버 부하 (500개 방 × 20명):**
```
총 대역폭: 500개 방 × 20 Mbps = 10 Gbps
```

**서버 1대로 가능한가?**
- 10 Gbps NIC 필요 (현실적으로 어려움)
- → **수평 확장 필요 (여러 서버)**

---

## 수평 확장 전략

### Option 1: Room 기반 샤딩 (추천)

```
┌──────────────────┐
│  Load Balancer   │
│  (Nginx/HAProxy) │
└────────┬─────────┘
         │
    ┌────┴────┐
    │         │
┌───▼────┐ ┌──▼─────┐
│Server 1│ │Server 2│
│        │ │        │
│Room A  │ │Room C  │
│Room B  │ │Room D  │
└────────┘ └────────┘
```

**구현:**
1. Room ID를 해싱
2. 특정 서버에 할당
3. 같은 방 사용자는 항상 같은 서버로

**장점:**
- 구현 간단
- 방별 완전 격리

**단점:**
- 방 크기 불균형 시 부하 불균형
- 서버 장애 시 전체 방 다운

### Option 2: Redis Pub/Sub (고급)

```
┌─────────┐   ┌─────────┐   ┌─────────┐
│Server 1 │◄─►│ Redis   │◄─►│Server 2 │
│Room A   │   │ Pub/Sub │   │Room A   │
└─────────┘   └─────────┘   └─────────┘

같은 방의 사용자가 다른 서버에 연결되어도
Redis를 통해 미디어 중계
```

**장점:**
- 유연한 부하 분산
- 서버 장애 대응

**단점:**
- 구현 복잡도 높음
- Redis 병목 가능

---

## 프로덕션급 구현

### Step 1: Room 관리 (메모리 효율)

```go
// Room: 방별 격리된 SFU 인스턴스
type Room struct {
    ID        string
    CreatedAt time.Time

    // Peer 관리
    peersMu sync.RWMutex
    peers   map[string]*Peer  // peerID -> Peer

    // 통계
    stats RoomStats
}

type RoomStats struct {
    PeakPeers     int       // 최대 동시 접속
    TotalJoins    int64     // 누적 입장
    TotalLeaves   int64     // 누적 퇴장
    BytesSent     uint64    // 전송 바이트
    BytesReceived uint64    // 수신 바이트
}

// 전역 Room 관리
var (
    roomsMu sync.RWMutex
    rooms   = make(map[string]*Room)
)

func getOrCreateRoom(roomID string) *Room {
    roomsMu.Lock()
    defer roomsMu.Unlock()

    room, exists := rooms[roomID]
    if !exists {
        room = &Room{
            ID:        roomID,
            CreatedAt: time.Now(),
            peers:     make(map[string]*Peer),
        }
        rooms[roomID] = room
        log.Printf("[Room %s] Created", roomID)
    }

    return room
}

// 빈 방 자동 정리 (메모리 누수 방지)
func cleanupEmptyRooms() {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        roomsMu.Lock()
        for roomID, room := range rooms {
            room.peersMu.RLock()
            isEmpty := len(room.peers) == 0
            room.peersMu.RUnlock()

            // 30분간 비어있으면 삭제
            if isEmpty && time.Since(room.CreatedAt) > 30*time.Minute {
                delete(rooms, roomID)
                log.Printf("[Room %s] Deleted (empty for 30min)", roomID)
            }
        }
        roomsMu.Unlock()
    }
}
```

### Step 2: Peer 관리 (리소스 제한)

```go
const (
    maxPeersPerRoom = 50      // 방당 최대 인원
    maxRooms        = 5000    // 서버당 최대 방 개수
    maxTracksPerPeer = 2      // Peer당 최대 트랙 (audio + video)
)

type Peer struct {
    ID       string
    RoomID   string
    Username string

    // WebSocket (Signaling)
    Conn   *websocket.Conn
    connMu sync.Mutex

    // WebRTC (Media)
    PeerConn *webrtc.PeerConnection
    pcMu     sync.Mutex

    // Tracks
    tracksMu sync.RWMutex
    tracks   map[string]*webrtc.TrackLocalStaticRTP

    // 통계
    stats PeerStats
}

type PeerStats struct {
    JoinedAt      time.Time
    BytesSent     uint64
    BytesReceived uint64
    PacketsSent   uint64
    PacketsLost   uint64
}

func (r *Room) addPeer(peer *Peer) error {
    r.peersMu.Lock()
    defer r.peersMu.Unlock()

    // 인원 제한 체크
    if len(r.peers) >= maxPeersPerRoom {
        return fmt.Errorf("room is full (max %d peers)", maxPeersPerRoom)
    }

    r.peers[peer.ID] = peer
    r.stats.TotalJoins++

    // 피크 갱신
    if len(r.peers) > r.stats.PeakPeers {
        r.stats.PeakPeers = len(r.peers)
    }

    log.Printf("[Room %s] Peer %s joined (%d/%d)",
        r.ID, peer.ID, len(r.peers), maxPeersPerRoom)

    return nil
}
```

### Step 3: Track 중계 최적화

```go
func (r *Room) broadcastTrack(sender *Peer, remoteTrack *webrtc.TrackRemote) error {
    // 1. Track 수 제한 체크
    sender.tracksMu.RLock()
    trackCount := len(sender.tracks)
    sender.tracksMu.RUnlock()

    if trackCount >= maxTracksPerPeer {
        return fmt.Errorf("max tracks reached (%d)", maxTracksPerPeer)
    }

    // 2. Local Track 생성
    localTrack, err := webrtc.NewTrackLocalStaticRTP(
        remoteTrack.Codec().RTPCodecCapability,
        fmt.Sprintf("track-%s-%s", sender.ID, remoteTrack.ID()),
        fmt.Sprintf("stream-%s", sender.ID),
    )
    if err != nil {
        return fmt.Errorf("create local track: %w", err)
    }

    // 3. Sender의 tracks에 추가
    sender.tracksMu.Lock()
    sender.tracks[localTrack.ID()] = localTrack
    sender.tracksMu.Unlock()

    // 4. 같은 방의 다른 모든 peer에게 추가
    r.peersMu.RLock()
    receivers := make([]*Peer, 0, len(r.peers)-1)
    for _, otherPeer := range r.peers {
        if otherPeer.ID != sender.ID {
            receivers = append(receivers, otherPeer)
        }
    }
    r.peersMu.RUnlock()

    // 5. Track 추가 (병렬 처리)
    var wg sync.WaitGroup
    for _, receiver := range receivers {
        wg.Add(1)
        go func(p *Peer) {
            defer wg.Done()

            p.pcMu.Lock()
            defer p.pcMu.Unlock()

            if p.PeerConn != nil {
                if _, err := p.PeerConn.AddTrack(localTrack); err != nil {
                    log.Printf("[Room %s] Failed to add track to peer %s: %v",
                        r.ID, p.ID, err)
                }
            }
        }(receiver)
    }
    wg.Wait()

    // 6. RTP 패킷 중계 (백그라운드)
    go r.relayRTP(sender, remoteTrack, localTrack)

    return nil
}

func (r *Room) relayRTP(sender *Peer, remote *webrtc.TrackRemote,
                         local *webrtc.TrackLocalStaticRTP) {
    defer func() {
        // Track 정리
        sender.tracksMu.Lock()
        delete(sender.tracks, local.ID())
        sender.tracksMu.Unlock()

        log.Printf("[Room %s] Track relay stopped for peer %s",
            r.ID, sender.ID)
    }()

    rtpBuf := make([]byte, 1500)  // MTU 크기
    for {
        // Remote track에서 읽기
        n, _, readErr := remote.Read(rtpBuf)
        if readErr != nil {
            return  // 연결 종료
        }

        // 통계 업데이트
        atomic.AddUint64(&sender.stats.BytesReceived, uint64(n))
        atomic.AddUint64(&sender.stats.PacketsSent, 1)

        // Local track으로 쓰기
        if _, writeErr := local.Write(rtpBuf[:n]); writeErr != nil {
            return  // 쓰기 실패
        }

        atomic.AddUint64(&sender.stats.BytesSent, uint64(n))
    }
}
```

### Step 4: Graceful Shutdown

```go
func main() {
    // Room 정리 백그라운드 작업
    go cleanupEmptyRooms()

    // HTTP 서버
    server := &http.Server{
        Addr:         ":9000",
        ReadTimeout:  10 * time.Second,
        WriteTimeout: 10 * time.Second,
    }

    // 시그널 핸들러
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

    go func() {
        log.Println("Voice server starting on :9000")
        if err := server.ListenAndServe(); err != http.ErrServerClosed {
            log.Fatalf("Server error: %v", err)
        }
    }()

    // 종료 시그널 대기
    <-sigChan
    log.Println("Shutting down gracefully...")

    // 1. HTTP 서버 종료 (새 연결 거부)
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := server.Shutdown(ctx); err != nil {
        log.Printf("Server shutdown error: %v", err)
    }

    // 2. 모든 방의 연결 종료
    closeAllRooms()

    log.Println("Shutdown complete")
}

func closeAllRooms() {
    roomsMu.Lock()
    defer roomsMu.Unlock()

    for roomID, room := range rooms {
        room.peersMu.Lock()
        for _, peer := range room.peers {
            if peer.PeerConn != nil {
                peer.PeerConn.Close()
            }
            if peer.Conn != nil {
                peer.Conn.Close()
            }
        }
        room.peersMu.Unlock()

        log.Printf("[Room %s] Closed (%d peers)", roomID, len(room.peers))
    }

    rooms = make(map[string]*Room)
}
```

---

## 모니터링 및 메트릭

### Prometheus 메트릭

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    roomsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "voice_rooms_total",
        Help: "Total number of active rooms",
    })

    peersGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
        Name: "voice_peers_total",
        Help: "Total number of peers per room",
    }, []string{"room"})

    bytesCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
        Name: "voice_bytes_total",
        Help: "Total bytes transferred",
    }, []string{"direction"})  // sent, received
)

func init() {
    prometheus.MustRegister(roomsGauge, peersGauge, bytesCounter)
}

// 정기 업데이트
func updateMetrics() {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        roomsMu.RLock()
        roomsGauge.Set(float64(len(rooms)))

        for roomID, room := range rooms {
            room.peersMu.RLock()
            peersGauge.WithLabelValues(roomID).Set(float64(len(room.peers)))
            room.peersMu.RUnlock()
        }
        roomsMu.RUnlock()
    }
}
```

### Health Check API

```go
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
    roomsMu.RLock()
    totalRooms := len(rooms)

    var totalPeers int
    for _, room := range rooms {
        room.peersMu.RLock()
        totalPeers += len(room.peers)
        room.peersMu.RUnlock()
    }
    roomsMu.RUnlock()

    json.NewEncoder(w).Encode(map[string]interface{}{
        "status":      "healthy",
        "rooms":       totalRooms,
        "peers":       totalPeers,
        "maxRooms":    maxRooms,
        "maxPerRoom":  maxPeersPerRoom,
        "uptime":      time.Since(startTime).String(),
    })
}
```

---

## 성능 최적화

### 1. Connection Pooling

```go
// WebRTC PeerConnection 재사용 (renegotiation)
func (p *Peer) addTrack(track *webrtc.TrackLocalStaticRTP) error {
    p.pcMu.Lock()
    defer p.pcMu.Unlock()

    if p.PeerConn == nil {
        return fmt.Errorf("peer connection not initialized")
    }

    // 기존 연결에 track 추가
    sender, err := p.PeerConn.AddTrack(track)
    if err != nil {
        return err
    }

    // Renegotiation 트리거
    return p.renegotiate()
}
```

### 2. Batch Processing

```go
// 여러 peer에게 track을 병렬로 추가
func (r *Room) addTrackToAll(track *webrtc.TrackLocalStaticRTP, exclude string) {
    r.peersMu.RLock()
    peers := make([]*Peer, 0, len(r.peers))
    for _, peer := range r.peers {
        if peer.ID != exclude {
            peers = append(peers, peer)
        }
    }
    r.peersMu.RUnlock()

    // 병렬 처리 (채널 사용)
    results := make(chan error, len(peers))
    for _, peer := range peers {
        go func(p *Peer) {
            results <- p.addTrack(track)
        }(peer)
    }

    // 결과 수집
    for i := 0; i < len(peers); i++ {
        if err := <-results; err != nil {
            log.Printf("Add track error: %v", err)
        }
    }
}
```

### 3. Memory Pool

```go
var rtpBufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 1500)
    },
}

func (r *Room) relayRTPOptimized(remote *webrtc.TrackRemote,
                                  local *webrtc.TrackLocalStaticRTP) {
    rtpBuf := rtpBufferPool.Get().([]byte)
    defer rtpBufferPool.Put(rtpBuf)

    for {
        n, _, err := remote.Read(rtpBuf)
        if err != nil {
            return
        }

        local.Write(rtpBuf[:n])
    }
}
```

---

## 배포 전략

### Docker 컨테이너화

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o voice-server main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/voice-server .

EXPOSE 9000
CMD ["./voice-server"]
```

### Kubernetes 배포

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: voice-server
spec:
  replicas: 3  # 수평 확장
  selector:
    matchLabels:
      app: voice-server
  template:
    metadata:
      labels:
        app: voice-server
    spec:
      containers:
      - name: voice-server
        image: voice-server:latest
        ports:
        - containerPort: 9000
        resources:
          requests:
            memory: "1Gi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 9000
          initialDelaySeconds: 30
          periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: voice-server
spec:
  type: LoadBalancer
  selector:
    app: voice-server
  ports:
  - port: 9000
    targetPort: 9000
```

---

## 체크리스트

### 기본 구현
- [ ] Room 관리 (생성/삭제/격리)
- [ ] Peer 생명주기 (join/leave)
- [ ] Track 중계 (RTP relay)
- [ ] Signaling (Offer/Answer/ICE)
- [ ] 리소스 제한 (방당 인원, 트랙 수)

### 프로덕션 준비
- [ ] Graceful shutdown
- [ ] 빈 방 자동 정리
- [ ] 통계 및 메트릭 (Prometheus)
- [ ] Health check API
- [ ] 에러 복구 (reconnection)
- [ ] 로깅 (structured logging)

### 확장성
- [ ] 수평 확장 (여러 서버)
- [ ] Load balancing (Room 샤딩)
- [ ] Connection pooling
- [ ] Memory pool (RTP 버퍼)

### 모니터링
- [ ] Room/Peer 카운트
- [ ] 대역폭 사용량
- [ ] CPU/메모리 프로파일링
- [ ] 에러율 추적

---

## 다음 단계

1. **기본 SFU 구현** (1-2주)
   - Room 기반 격리
   - Track 중계
   - 20명 방 테스트

2. **리소스 제한 추가** (1주)
   - 방당 인원 제한
   - 메모리 모니터링
   - 부하 테스트

3. **모니터링 구축** (1주)
   - Prometheus 메트릭
   - Grafana 대시보드
   - 알람 설정

4. **수평 확장** (2주)
   - Room 샤딩 구현
   - Load balancer 설정
   - 1만명 시뮬레이션

---

**프로덕션급 확장성을 고려한 설계로 1만명 규모를 지원할 수 있습니다!**
