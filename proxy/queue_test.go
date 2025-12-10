package main

import (
	"testing"
	"time" // for timeout test
)

func TestEnqueueDequeue(t *testing.T) { // testing.T는 Go의 테스트 프레임워크가 제공하는 테스터 헬퍼 객체.
	// 포인터(*)로 받는 이유는 테스트 상태를 계속 업데이트해야 하기 때문
	queue := NewMessageQueue(10)

	msg := &Message{
		Type:     "chat",
		Username: "testuser",
		Content:  "Hello",
	}
	// test enqueue
	success := queue.Enqueue(msg)
	if !success {
		t.Fatal("Enqueue failed on empty queue")
	}

	// test dequeue
	retrieved, ok := queue.Dequeue()
	if !ok {
		t.Fatal("Dequeue failed on non-empty queue")
	}
	if retrieved.Content != "Hello" {
		t.Errorf("Expected 'Hello', got '%s'", retrieved.Content)
	}

}

func TestQueueFull(t *testing.T) {
	queue := NewMessageQueue(2)

	msg1 := &Message{Content: "First"}
	msg2 := &Message{Content: "Second"}
	msg3 := &Message{Content: "Third"}

	if !queue.Enqueue(msg1) {
		t.Fatal("First enqueue failed")
	}

	if !queue.Enqueue(msg2) {
		t.Fatal("Second enqueue failed")
	}

	// Queue is now full (size=2), third enqueue should fail
	if queue.Enqueue(msg3) {
		t.Error("Expected third enqueue to fail on full queue")
	}
}

func TestDequeueEmpty(t *testing.T) {
	queue := NewMessageQueue(10)

	start := time.Now()
	msg, ok := queue.Dequeue()
	elapsed := time.Since(start)

	if ok {
		t.Error("Dequeue should fail on empty queue")
	}
	if msg != nil {
		t.Error("Expected nil message")
	}

	// 1s timeout check
	if elapsed < 950*time.Millisecond || elapsed > 1100*time.Millisecond {
		t.Logf("Timeout took %v, expected ~1s", elapsed)
	}
}
