package p2p

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/cocursor/backend/internal/domain/p2p"
	"github.com/cocursor/backend/internal/infrastructure/log"
)

// WebSocketServer WebSocket 服务端（Leader 使用）
type WebSocketServer struct {
	mu           sync.RWMutex
	connections  map[string]*WebSocketConnection // memberID -> connection
	eventHandler p2p.EventHandler
	logger       *slog.Logger
	upgrader     websocket.Upgrader
}

// WebSocketConnection 单个 WebSocket 连接
type WebSocketConnection struct {
	mu            sync.Mutex
	conn          *websocket.Conn
	memberID      string
	memberName    string
	endpoint      string
	authenticated bool
	lastPing      time.Time
	sendChan      chan []byte
	done          chan struct{}
}

// NewWebSocketServer 创建 WebSocket 服务端
func NewWebSocketServer(eventHandler p2p.EventHandler) *WebSocketServer {
	return &WebSocketServer{
		connections:  make(map[string]*WebSocketConnection),
		eventHandler: eventHandler,
		logger:       log.NewModuleLogger("p2p", "websocket_server"),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // 局域网内允许所有来源
			},
		},
	}
}

// HandleConnection 处理新的 WebSocket 连接
func (s *WebSocketServer) HandleConnection(w http.ResponseWriter, r *http.Request, teamID string) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("failed to upgrade connection",
			"error", err,
		)
		return
	}

	// 创建连接对象
	wsConn := &WebSocketConnection{
		conn:     conn,
		lastPing: time.Now(),
		sendChan: make(chan []byte, 256),
		done:     make(chan struct{}),
	}

	// 等待认证
	go s.handleAuth(wsConn, teamID)
}

// handleAuth 处理认证
func (s *WebSocketServer) handleAuth(wsConn *WebSocketConnection, teamID string) {
	// 设置读取超时（10 秒内必须完成认证）
	_ = wsConn.conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// 读取认证消息
	_, message, err := wsConn.conn.ReadMessage()
	if err != nil {
		s.logger.Warn("failed to read auth message",
			"error", err,
		)
		_ = wsConn.conn.Close()
		return
	}

	// 解析事件
	var event p2p.Event
	if err := json.Unmarshal(message, &event); err != nil {
		s.logger.Warn("failed to parse auth message",
			"error", err,
		)
		s.sendAuthResult(wsConn, false, "invalid message format")
		_ = wsConn.conn.Close()
		return
	}

	if event.Type != p2p.EventAuth {
		s.sendAuthResult(wsConn, false, "expected auth event")
		_ = wsConn.conn.Close()
		return
	}

	// 解析认证数据
	var authPayload p2p.AuthPayload
	if err := event.ParsePayload(&authPayload); err != nil {
		s.sendAuthResult(wsConn, false, "invalid auth payload")
		_ = wsConn.conn.Close()
		return
	}

	// 验证成员（这里简化处理，实际应检查成员列表）
	if authPayload.MemberID == "" {
		s.sendAuthResult(wsConn, false, "member_id is required")
		_ = wsConn.conn.Close()
		return
	}

	// 认证成功
	wsConn.memberID = authPayload.MemberID
	wsConn.memberName = authPayload.MemberName
	wsConn.endpoint = authPayload.Endpoint
	wsConn.authenticated = true

	// 清除读取超时
	_ = wsConn.conn.SetReadDeadline(time.Time{})

	// 发送认证成功
	s.sendAuthResult(wsConn, true, "")

	// 添加到连接列表
	s.mu.Lock()
	// 如果已有连接，关闭旧连接
	if oldConn, exists := s.connections[wsConn.memberID]; exists {
		close(oldConn.done)
		_ = oldConn.conn.Close()
	}
	s.connections[wsConn.memberID] = wsConn
	s.mu.Unlock()

	s.logger.Info("member connected",
		"member_id", wsConn.memberID,
		"member_name", wsConn.memberName,
		"endpoint", wsConn.endpoint,
	)

	// 触发成员上线事件
	if s.eventHandler != nil {
		onlineEvent, _ := p2p.NewEvent(p2p.EventMemberOnline, teamID, p2p.MemberStatusPayload{
			MemberID:   wsConn.memberID,
			MemberName: wsConn.memberName,
			Endpoint:   wsConn.endpoint,
			IsOnline:   true,
		})
		_ = s.eventHandler.HandleEvent(onlineEvent)
	}

	// 启动读写协程
	go s.readPump(wsConn, teamID)
	go s.writePump(wsConn)
}

// sendAuthResult 发送认证结果
func (s *WebSocketServer) sendAuthResult(wsConn *WebSocketConnection, success bool, errMsg string) {
	event, _ := p2p.NewEvent(p2p.EventAuthResult, "", p2p.AuthResultPayload{
		Success: success,
		Error:   errMsg,
	})
	data, _ := json.Marshal(event)
	_ = wsConn.conn.WriteMessage(websocket.TextMessage, data)
}

// readPump 读取消息
func (s *WebSocketServer) readPump(wsConn *WebSocketConnection, teamID string) {
	defer func() {
		s.removeConnection(wsConn, teamID)
	}()

	wsConn.conn.SetReadLimit(512 * 1024) // 512KB

	// 设置初始读取超时，超过 HeartbeatTimeout 未收到任何消息则断开
	_ = wsConn.conn.SetReadDeadline(time.Now().Add(p2p.HeartbeatTimeout))

	wsConn.conn.SetPongHandler(func(string) error {
		wsConn.mu.Lock()
		wsConn.lastPing = time.Now()
		wsConn.mu.Unlock()
		// 收到 Pong 说明对方存活，续期读取超时
		_ = wsConn.conn.SetReadDeadline(time.Now().Add(p2p.HeartbeatTimeout))
		return nil
	})

	for {
		select {
		case <-wsConn.done:
			return
		default:
		}

		_, message, err := wsConn.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Warn("connection read error",
					"member_id", wsConn.memberID,
					"error", err,
				)
			}
			return
		}

		// 收到任何消息都续期读取超时
		_ = wsConn.conn.SetReadDeadline(time.Now().Add(p2p.HeartbeatTimeout))

		// 解析并处理事件
		var event p2p.Event
		if err := json.Unmarshal(message, &event); err != nil {
			s.logger.Warn("failed to parse message",
				"member_id", wsConn.memberID,
				"error", err,
			)
			continue
		}

		// 处理心跳
		if event.Type == p2p.EventPing {
			pongEvent, _ := p2p.NewEvent(p2p.EventPong, teamID, nil)
			data, _ := json.Marshal(pongEvent)
			wsConn.sendChan <- data
			continue
		}

		// 交给事件处理器
		if s.eventHandler != nil {
			_ = s.eventHandler.HandleEvent(&event)
		}
	}
}

// writePump 写入消息
func (s *WebSocketServer) writePump(wsConn *WebSocketConnection) {
	ticker := time.NewTicker(p2p.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-wsConn.done:
			return
		case message := <-wsConn.sendChan:
			wsConn.mu.Lock()
			err := wsConn.conn.WriteMessage(websocket.TextMessage, message)
			wsConn.mu.Unlock()
			if err != nil {
				s.logger.Warn("failed to write message",
					"member_id", wsConn.memberID,
					"error", err,
				)
				return
			}
		case <-ticker.C:
			// 发送 Ping
			wsConn.mu.Lock()
			err := wsConn.conn.WriteMessage(websocket.PingMessage, nil)
			wsConn.mu.Unlock()
			if err != nil {
				return
			}
		}
	}
}

// removeConnection 移除连接
func (s *WebSocketServer) removeConnection(wsConn *WebSocketConnection, teamID string) {
	s.mu.Lock()
	if existing, ok := s.connections[wsConn.memberID]; ok && existing == wsConn {
		delete(s.connections, wsConn.memberID)
	}
	s.mu.Unlock()

	close(wsConn.done)
	wsConn.conn.Close()

	s.logger.Info("member disconnected",
		"member_id", wsConn.memberID,
		"member_name", wsConn.memberName,
	)

	// 触发成员离线事件
	if s.eventHandler != nil && wsConn.memberID != "" {
		offlineEvent, _ := p2p.NewEvent(p2p.EventMemberOffline, teamID, p2p.MemberStatusPayload{
			MemberID:   wsConn.memberID,
			MemberName: wsConn.memberName,
			IsOnline:   false,
		})
		s.eventHandler.HandleEvent(offlineEvent)
	}
}

// Broadcast 广播消息给所有连接
func (s *WebSocketServer) Broadcast(event *p2p.Event) {
	data, err := json.Marshal(event)
	if err != nil {
		s.logger.Error("failed to marshal event",
			"error", err,
		)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, conn := range s.connections {
		if conn.authenticated {
			select {
			case conn.sendChan <- data:
			default:
				// 发送缓冲区满，跳过
				s.logger.Warn("send buffer full, dropping message",
					"member_id", conn.memberID,
				)
			}
		}
	}
}

// SendToMember 发送消息给指定成员
func (s *WebSocketServer) SendToMember(memberID string, event *p2p.Event) error {
	s.mu.RLock()
	conn, exists := s.connections[memberID]
	s.mu.RUnlock()

	if !exists || !conn.authenticated {
		return nil
	}

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	select {
	case conn.sendChan <- data:
		return nil
	default:
		return nil // 缓冲区满，静默失败
	}
}

// GetConnectedMembers 获取已连接的成员列表
func (s *WebSocketServer) GetConnectedMembers() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var members []string
	for memberID, conn := range s.connections {
		if conn.authenticated {
			members = append(members, memberID)
		}
	}
	return members
}

// IsConnected 检查成员是否已连接
func (s *WebSocketServer) IsConnected(memberID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	conn, exists := s.connections[memberID]
	return exists && conn.authenticated
}

// SetEventHandler 设置事件处理器
func (s *WebSocketServer) SetEventHandler(handler p2p.EventHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.eventHandler = handler
}

// Close 关闭所有连接
func (s *WebSocketServer) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, conn := range s.connections {
		close(conn.done)
		conn.conn.Close()
	}
	s.connections = make(map[string]*WebSocketConnection)
}
