package common

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"
)

// SSEManager SSE 连接管理器
type SSEManager struct {
	clients map[string]chan string
	mu      sync.RWMutex
}

// NewSSEManager 创建 SSE 管理器
func NewSSEManager() *SSEManager {
	return &SSEManager{
		clients: make(map[string]chan string),
	}
}

// Register 注册 SSE 客户端
func (m *SSEManager) Register(taskID string) chan string {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan string, 100) // 缓冲通道
	m.clients[taskID] = ch
	return ch
}

// Unregister 注销 SSE 客户端
func (m *SSEManager) Unregister(taskID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ch, ok := m.clients[taskID]; ok {
		close(ch)
		delete(m.clients, taskID)
	}
}

// Send 发送消息
func (m *SSEManager) Send(taskID string, data interface{}) {
	m.mu.RLock()
	ch, ok := m.clients[taskID]
	m.mu.RUnlock()

	if !ok {
		return
	}

	// 将数据转为 JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}

	// 非阻塞发送
	select {
	case ch <- string(jsonData):
	case <-time.After(5 * time.Second):
		// 超时则放弃
	}
}

// Complete 完成连接
func (m *SSEManager) Complete(taskID string) {
	m.Unregister(taskID)
}

// Exists 检查连接是否存在
func (m *SSEManager) Exists(taskID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.clients[taskID]
	return ok
}

// WriteSSE 写入 SSE 消息
func WriteSSE(w io.Writer, data string) error {
	_, err := fmt.Fprintf(w, "data: %s\n\n", data)
	return err
}
