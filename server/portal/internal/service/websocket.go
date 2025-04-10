package service

import (
	"sync"

	"github.com/gorilla/websocket"
)

// WebSocketClient 表示一个 WebSocket 客户端连接
type WebSocketClient struct {
	Conn     *websocket.Conn
	WriteMux sync.Mutex
}

// NewWebSocketClient 创建新的 WebSocket 客户端
func NewWebSocketClient(conn *websocket.Conn) *WebSocketClient {
	return &WebSocketClient{
		Conn: conn,
	}
}

// SafeWrite 安全地写入消息
func (c *WebSocketClient) SafeWrite(v interface{}) error {
	c.WriteMux.Lock()
	defer c.WriteMux.Unlock()
	return c.Conn.WriteJSON(v)
}

// WebSocketManager WebSocket 连接管理器
type WebSocketManager struct {
	Clients   map[*WebSocketClient]bool
	ClientMux sync.Mutex
}

// NewWebSocketManager 创建新的 WebSocket 管理器
func NewWebSocketManager() *WebSocketManager {
	return &WebSocketManager{
		Clients: make(map[*WebSocketClient]bool),
	}
}

// AddClient 添加客户端
func (m *WebSocketManager) AddClient(client *WebSocketClient) {
	m.ClientMux.Lock()
	defer m.ClientMux.Unlock()
	m.Clients[client] = true
}

// RemoveClient 移除客户端
func (m *WebSocketManager) RemoveClient(client *WebSocketClient) {
	m.ClientMux.Lock()
	defer m.ClientMux.Unlock()
	delete(m.Clients, client)
}

// BroadcastMessage 广播消息给所有客户端
func (m *WebSocketManager) BroadcastMessage(v interface{}) {
	m.ClientMux.Lock()
	clients := make([]*WebSocketClient, 0, len(m.Clients))
	for client := range m.Clients {
		clients = append(clients, client)
	}
	m.ClientMux.Unlock()

	for _, client := range clients {
		go client.SafeWrite(v)
	}
} 