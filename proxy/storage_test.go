package main

import (
	"os"
	"testing"
	"time"
)

// test storage creation
func TestStorageCreation(t *testing.T) {
	tmpfile := "test_storage_creation.jsonl"
	defer os.Remove(tmpfile)

	storage, err := NewStorage(tmpfile)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	if storage.filepath != tmpfile {
		t.Errorf("Expected filepath %s, got %s", tmpfile, storage.filepath)
	}

	if !storage.enabled {
		t.Error("Storage should be enabled by default")
	}

	if _, err := os.Stat(tmpfile); os.IsNotExist(err) {
		t.Error("Storage file was not created")
	}
}

func TestStorageSync(t *testing.T) {
	tmpfile := "test_sync_deadlock.jsonl"
	defer os.Remove(tmpfile)

	storage, err := NewStorage(tmpfile)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	done := make(chan error, 1)
	go func() {
		done <- storage.Sync()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Sync() failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("DEADLOCK DETECTED: Sync() did not complete 2 seconds")
	}
}

// // TestStorageConcurrentWrites - 동시 쓰기 안전성 검증
// func TestStorageConcurrentWrites(t *testing.T) {
// 	tmpfile := "test_concurrent_writes.jsonl"
// 	defer os.Remove(tmpfile)

// 	storage, err := NewStorage(tmpfile)
// 	if err != nil {
// 		t.Fatalf("Failed to create storage: %v", err)
// 	}
// 	defer storage.Close()

// 	const numGoroutines = 100
// 	const messagesPerGoroutine = 10
// 	var wg sync.WaitGroup

// 	// 100개 고루틴에서 각각 10개 메시지 쓰기
// 	for i := 0; i < numGoroutines; i++ {
// 		wg.Add(1)
// 		go func(id int) {
// 			defer wg.Done()
// 			for j := 0; j < messagesPerGoroutine; j++ {
// 				msg := &Message{
// 					Type:      "test",
// 					Username:  "user",
// 					Room:      "room1",
// 					Content:   "concurrent message",
// 					Timestamp: time.Now(),
// 				}
// 				if err := storage.LogMessage(msg); err != nil {
// 					t.Errorf("LogMessage failed: %v", err)
// 				}
// 			}
// 		}(i)
// 	}

// 	wg.Wait()

// 	// Sync도 동시성 안전해야 함
// 	err = storage.Sync()
// 	if err != nil {
// 		t.Errorf("Sync failed: %v", err)
// 	}

// 	// 파일에 정확히 1000줄이 기록되었는지 확인
// 	storage.Close()
// 	file, err := os.Open(tmpfile)
// 	if err != nil {
// 		t.Fatalf("Failed to open test file: %v", err)
// 	}
// 	defer file.Close()

// 	scanner := bufio.NewScanner(file)
// 	lineCount := 0
// 	for scanner.Scan() {
// 		lineCount++
// 	}

// 	expectedLines := numGoroutines * messagesPerGoroutine
// 	if lineCount != expectedLines {
// 		t.Errorf("Expected %d lines, got %d", expectedLines, lineCount)
// 	}
// }

// // TestStorageLogMessage - 메시지 로깅 기능 검증
// func TestStorageLogMessage(t *testing.T) {
// 	tmpfile := "test_log_message.jsonl"
// 	defer os.Remove(tmpfile)

// 	storage, err := NewStorage(tmpfile)
// 	if err != nil {
// 		t.Fatalf("Failed to create storage: %v", err)
// 	}
// 	defer storage.Close()

// 	msg := &Message{
// 		Type:      "chat",
// 		Username:  "testuser",
// 		Room:      "testroom",
// 		Content:   "Hello, World!",
// 		Timestamp: time.Now(),
// 	}

// 	err = storage.LogMessage(msg)
// 	if err != nil {
// 		t.Fatalf("LogMessage failed: %v", err)
// 	}

// 	storage.Close()

// 	// 파일 내용 검증
// 	file, err := os.Open(tmpfile)
// 	if err != nil {
// 		t.Fatalf("Failed to open test file: %v", err)
// 	}
// 	defer file.Close()

// 	var readMsg Message
// 	decoder := json.NewDecoder(file)
// 	err = decoder.Decode(&readMsg)
// 	if err != nil {
// 		t.Fatalf("Failed to decode message: %v", err)
// 	}

// 	if readMsg.Type != msg.Type {
// 		t.Errorf("Expected Type %s, got %s", msg.Type, readMsg.Type)
// 	}
// 	if readMsg.Username != msg.Username {
// 		t.Errorf("Expected Username %s, got %s", msg.Username, readMsg.Username)
// 	}
// 	if readMsg.Content != msg.Content {
// 		t.Errorf("Expected Content %s, got %s", msg.Content, readMsg.Content)
// 	}
// }

// // TestStorageSetEnabled - Storage enable/disable 기능 검증
// func TestStorageSetEnabled(t *testing.T) {
// 	tmpfile := "test_set_enabled.jsonl"
// 	defer os.Remove(tmpfile)

// 	storage, err := NewStorage(tmpfile)
// 	if err != nil {
// 		t.Fatalf("Failed to create storage: %v", err)
// 	}
// 	defer storage.Close()

// 	// Disable storage
// 	storage.SetEnabled(false)

// 	msg := &Message{
// 		Type:     "chat",
// 		Username: "testuser",
// 		Room:     "testroom",
// 		Content:  "This should not be logged",
// 	}

// 	err = storage.LogMessage(msg)
// 	if err != nil {
// 		t.Fatalf("LogMessage should not fail when disabled: %v", err)
// 	}

// 	storage.Close()

// 	// 파일이 비어있어야 함
// 	file, err := os.Open(tmpfile)
// 	if err != nil {
// 		t.Fatalf("Failed to open test file: %v", err)
// 	}
// 	defer file.Close()

// 	scanner := bufio.NewScanner(file)
// 	if scanner.Scan() {
// 		t.Error("File should be empty when storage is disabled")
// 	}
// }

// // TestStorageTimestamp - 자동 Timestamp 할당 검증
// func TestStorageTimestamp(t *testing.T) {
// 	tmpfile := "test_timestamp.jsonl"
// 	defer os.Remove(tmpfile)

// 	storage, err := NewStorage(tmpfile)
// 	if err != nil {
// 		t.Fatalf("Failed to create storage: %v", err)
// 	}
// 	defer storage.Close()

// 	// Timestamp가 없는 메시지
// 	msg := &Message{
// 		Type:     "chat",
// 		Username: "testuser",
// 		Room:     "testroom",
// 		Content:  "Test message",
// 	}

// 	before := time.Now()
// 	err = storage.LogMessage(msg)
// 	if err != nil {
// 		t.Fatalf("LogMessage failed: %v", err)
// 	}
// 	after := time.Now()

// 	storage.Close()

// 	// 파일에서 읽어서 Timestamp 확인
// 	file, err := os.Open(tmpfile)
// 	if err != nil {
// 		t.Fatalf("Failed to open test file: %v", err)
// 	}
// 	defer file.Close()

// 	var readMsg Message
// 	decoder := json.NewDecoder(file)
// 	err = decoder.Decode(&readMsg)
// 	if err != nil {
// 		t.Fatalf("Failed to decode message: %v", err)
// 	}

// 	// Timestamp가 자동으로 할당되었는지 확인
// 	if readMsg.Timestamp.IsZero() {
// 		t.Error("Timestamp should be automatically assigned")
// 	}

// 	// Timestamp가 합리적인 범위인지 확인
// 	if readMsg.Timestamp.Before(before) || readMsg.Timestamp.After(after) {
// 		t.Errorf("Timestamp %v is out of expected range [%v, %v]",
// 			readMsg.Timestamp, before, after)
// 	}
// }

// // TestStorageClose - Close 중복 호출 안전성 검증
// func TestStorageClose(t *testing.T) {
// 	tmpfile := "test_close.jsonl"
// 	defer os.Remove(tmpfile)

// 	storage, err := NewStorage(tmpfile)
// 	if err != nil {
// 		t.Fatalf("Failed to create storage: %v", err)
// 	}

// 	// 첫 번째 Close
// 	err = storage.Close()
// 	if err != nil {
// 		t.Errorf("First Close() failed: %v", err)
// 	}

// 	// 두 번째 Close (중복 호출 안전성 테스트)
// 	// 참고: os.File.Close()는 중복 호출 시 에러를 반환할 수 있음
// 	err = storage.Close()
// 	if err != nil {
// 		t.Logf("Second Close() returned error (expected): %v", err)
// 	}
// }

// // BenchmarkStorageLogMessage - 메시지 로깅 성능 벤치마크
// func BenchmarkStorageLogMessage(b *testing.B) {
// 	tmpfile := "bench_log_message.jsonl"
// 	defer os.Remove(tmpfile)

// 	storage, err := NewStorage(tmpfile)
// 	if err != nil {
// 		b.Fatalf("Failed to create storage: %v", err)
// 	}
// 	defer storage.Close()

// 	msg := &Message{
// 		Type:      "chat",
// 		Username:  "benchuser",
// 		Room:      "benchroom",
// 		Content:   "Benchmark message",
// 		Timestamp: time.Now(),
// 	}

// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		storage.LogMessage(msg)
// 	}
// }

// // BenchmarkStorageConcurrentWrites - 동시 쓰기 성능 벤치마크
// func BenchmarkStorageConcurrentWrites(b *testing.B) {
// 	tmpfile := "bench_concurrent.jsonl"
// 	defer os.Remove(tmpfile)

// 	storage, err := NewStorage(tmpfile)
// 	if err != nil {
// 		b.Fatalf("Failed to create storage: %v", err)
// 	}
// 	defer storage.Close()

// 	msg := &Message{
// 		Type:      "chat",
// 		Username:  "benchuser",
// 		Room:      "benchroom",
// 		Content:   "Concurrent benchmark message",
// 		Timestamp: time.Now(),
// 	}

// 	b.ResetTimer()
// 	b.RunParallel(func(pb *testing.PB) {
// 		for pb.Next() {
// 			storage.LogMessage(msg)
// 		}
// 	})
// }
