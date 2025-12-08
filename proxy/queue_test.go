package main

import (
	"testing"
	"time"  // for timeout test
)

func TestEnqueueDequeue(t *testing.T) {  // testing.T는 Go의 테스트 프레임워크가 제공하는 테스터 헬퍼 객체. 
                                         // 포인터(*)로 받는 이유는 테스트 상태를 계속 업데이트해야 하기 때문
	queue := NewMessageQueue(10)

	msg := &Message{
		Type: "chat",
		Username: "testuser",
		Content: "Hello",
	}
	// test enqueue
	success := queue.Enqueue(msg)
	if !success {
		t.Fatal("Enqueue failed on empty queue")
	}

	
}