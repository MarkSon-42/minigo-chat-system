// WebSocket 연결 및 채팅 클라이언트 로직

let ws = null;
let currentUsername = '';
let currentRoom = '';

// 로그인/입장
function joinChat() {
    const usernameInput = document.getElementById('usernameInput');
    const roomInput = document.getElementById('roomInput');
    
    const username = usernameInput.value.trim();
    const room = roomInput.value.trim();
    
    if (!username || !room) {
        alert('사용자 이름과 채팅방 이름을 모두 입력해주세요.');
        return;
    }
    
    currentUsername = username;
    currentRoom = room;
    
    // UI 전환
    document.getElementById('loginScreen').style.display = 'none';
    document.getElementById('chatScreen').style.display = 'flex';
    
    // 채팅방 정보 표시
    document.getElementById('chatTitle').textContent = `# ${room}`;
    document.getElementById('userInfo').textContent = username;
    
    // WebSocket 연결
    connectWebSocket();
}

// WebSocket 연결
function connectWebSocket() {
    const wsUrl = `ws://localhost:8081/ws?username=${encodeURIComponent(currentUsername)}&room=${encodeURIComponent(currentRoom)}`;
    
    console.log('Connecting to:', wsUrl);
    ws = new WebSocket(wsUrl);
    
    ws.onopen = () => {
        console.log('✅ WebSocket 연결됨');
        addSystemMessage(`${currentRoom} 채팅방에 입장했습니다.`);
        document.getElementById('sendButton').disabled = false;
    };
    
    ws.onmessage = (event) => {
        console.log('📨 메시지 수신:', event.data);
        try {
            const message = JSON.parse(event.data);
            handleIncomingMessage(message);
        } catch (err) {
            console.error('메시지 파싱 에러:', err);
        }
    };
    
    ws.onerror = (error) => {
        console.error('❌ WebSocket 에러:', error);
        addSystemMessage('연결 오류가 발생했습니다.');
    };
    
    ws.onclose = () => {
        console.log('🔌 WebSocket 연결 종료');
        addSystemMessage('연결이 종료되었습니다.');
        document.getElementById('sendButton').disabled = true;
    };
}

// 수신 메시지 처리
function handleIncomingMessage(message) {
    const { type, username, content, timestamp } = message;
    
    switch (type) {
        case 'message':
            addChatMessage(username, content, username === currentUsername, timestamp);
            break;
        case 'join':
            addSystemMessage(`${username}님이 입장했습니다.`);
            break;
        case 'leave':
            addSystemMessage(`${username}님이 퇴장했습니다.`);
            break;
        default:
            console.log('알 수 없는 메시지 타입:', type);
    }
}

// 메시지 전송
function sendMessage() {
    const messageInput = document.getElementById('messageInput');
    const content = messageInput.value.trim();
    
    if (!content) {
        return;
    }
    
    if (!ws || ws.readyState !== WebSocket.OPEN) {
        alert('연결이 끊어졌습니다. 다시 입장해주세요.');
        return;
    }
    
    const message = {
        type: 'message',
        username: currentUsername,
        room: currentRoom,
        content: content,
        timestamp: new Date().toISOString()
    };
    
    console.log('📤 메시지 전송:', message);
    ws.send(JSON.stringify(message));
    
    // 입력창 초기화
    messageInput.value = '';
    messageInput.focus();
}

// Enter 키로 전송
function handleKeyPress(event) {
    if (event.key === 'Enter') {
        sendMessage();
    }
}

// 채팅 메시지 추가
function addChatMessage(username, content, isOwn, timestamp) {
    const container = document.getElementById('messagesContainer');
    
    const messageDiv = document.createElement('div');
    messageDiv.className = `message ${isOwn ? 'own' : 'other'}`;
    
    const bubbleDiv = document.createElement('div');
    bubbleDiv.className = 'message-bubble';
    
    // 사용자 이름 표시 (다른 사람 메시지만)
    if (!isOwn) {
        const usernameSpan = document.createElement('div');
        usernameSpan.style.fontSize = '12px';
        usernameSpan.style.opacity = '0.7';
        usernameSpan.style.marginBottom = '4px';
        usernameSpan.textContent = username;
        bubbleDiv.appendChild(usernameSpan);
    }
    
    const contentDiv = document.createElement('div');
    contentDiv.textContent = content;
    bubbleDiv.appendChild(contentDiv);
    
    // 시간 표시
    if (timestamp) {
        const timeDiv = document.createElement('div');
        timeDiv.style.fontSize = '11px';
        timeDiv.style.opacity = '0.6';
        timeDiv.style.marginTop = '4px';
        timeDiv.textContent = formatTime(timestamp);
        bubbleDiv.appendChild(timeDiv);
    }
    
    messageDiv.appendChild(bubbleDiv);
    container.appendChild(messageDiv);
    
    // 스크롤을 최하단으로
    container.scrollTop = container.scrollHeight;
}

// 시스템 메시지 추가
function addSystemMessage(content) {
    const container = document.getElementById('messagesContainer');
    
    const messageDiv = document.createElement('div');
    messageDiv.className = 'message system';
    
    const bubbleDiv = document.createElement('div');
    bubbleDiv.className = 'message-bubble';
    bubbleDiv.textContent = content;
    
    messageDiv.appendChild(bubbleDiv);
    container.appendChild(messageDiv);
    
    // 스크롤을 최하단으로
    container.scrollTop = container.scrollHeight;
}

// 시간 포맷팅
function formatTime(timestamp) {
    const date = new Date(timestamp);
    const hours = date.getHours().toString().padStart(2, '0');
    const minutes = date.getMinutes().toString().padStart(2, '0');
    return `${hours}:${minutes}`;
}

// 채팅방 나가기
function leaveChat() {
    if (ws) {
        ws.close();
    }
    
    // UI 초기화
    document.getElementById('chatScreen').style.display = 'none';
    document.getElementById('loginScreen').style.display = 'block';
    document.getElementById('messagesContainer').innerHTML = '';
    document.getElementById('messageInput').value = '';
    
    currentUsername = '';
    currentRoom = '';
}

// 페이지 이탈 시 WebSocket 정리
window.addEventListener('beforeunload', () => {
    if (ws) {
        ws.close();
    }
});
