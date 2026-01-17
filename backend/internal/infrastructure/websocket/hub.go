package websocket

import (
	"encoding/json"
	"sync"
)

// Hub WebSocket 连接管理中心
type Hub struct {
	// 按团队码分组的连接
	teams map[string]map[*Connection]bool
	// 注册连接
	register chan *Connection
	// 注销连接
	unregister chan *Connection
	// 广播消息
	broadcast chan *Message
	mu        sync.RWMutex
}

// Connection WebSocket 连接
type Connection struct {
	TeamCode string
	Send     chan []byte
}

// Message 消息
type Message struct {
	TeamCode string
	Data     []byte
}

// NewHub 创建 Hub
func NewHub() *Hub {
	return &Hub{
		teams:     make(map[string]map[*Connection]bool),
		register:  make(chan *Connection),
		unregister: make(chan *Connection),
		broadcast:  make(chan *Message),
	}
}

// Run 运行 Hub（需要在 goroutine 中运行）
func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			if h.teams[conn.TeamCode] == nil {
				h.teams[conn.TeamCode] = make(map[*Connection]bool)
			}
			h.teams[conn.TeamCode][conn] = true
			h.mu.Unlock()

		case conn := <-h.unregister:
			h.mu.Lock()
			if team, ok := h.teams[conn.TeamCode]; ok {
				if _, ok := team[conn]; ok {
					delete(team, conn)
					close(conn.Send)
					if len(team) == 0 {
						delete(h.teams, conn.TeamCode)
					}
				}
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.RLock()
			if team, ok := h.teams[msg.TeamCode]; ok {
				for conn := range team {
					select {
					case conn.Send <- msg.Data:
					default:
						close(conn.Send)
						delete(team, conn)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Start 启动 Hub（启动后台 goroutine）
func (h *Hub) Start() {
	go h.Run()
}

// Register 注册连接
func (h *Hub) Register(conn *Connection) {
	h.register <- conn
}

// Unregister 注销连接
func (h *Hub) Unregister(conn *Connection) {
	h.unregister <- conn
}

// BroadcastToTeam 向指定团队广播消息
func (h *Hub) BroadcastToTeam(teamCode string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	h.broadcast <- &Message{
		TeamCode: teamCode,
		Data:     jsonData,
	}
	return nil
}
