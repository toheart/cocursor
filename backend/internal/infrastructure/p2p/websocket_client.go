package p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/cocursor/backend/internal/domain/p2p"
	"github.com/cocursor/backend/internal/infrastructure/log"
)

// WebSocketClient WebSocket 客户端（成员使用）
type WebSocketClient struct {
	mu           sync.RWMutex
	conn         *websocket.Conn
	teamID       string
	endpoint     string
	memberID     string
	memberName   string
	localEndpoint string
	state        p2p.ConnectionState
	lastPing     time.Time
	retryCount   int
	sendChan     chan []byte
	done         chan struct{}
	listener     p2p.EventListener
	logger       *slog.Logger
}

// NewWebSocketClient 创建 WebSocket 客户端
func NewWebSocketClient(teamID, endpoint, memberID, memberName, localEndpoint string, listener p2p.EventListener) *WebSocketClient {
	return &WebSocketClient{
		teamID:        teamID,
		endpoint:      endpoint,
		memberID:      memberID,
		memberName:    memberName,
		localEndpoint: localEndpoint,
		state:         p2p.StateDisconnected,
		sendChan:      make(chan []byte, 256),
		done:          make(chan struct{}),
		listener:      listener,
		logger:        log.NewModuleLogger("p2p", "websocket_client"),
	}
}

// Connect 连接到服务端
func (c *WebSocketClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	if c.state == p2p.StateConnected {
		c.mu.Unlock()
		return nil
	}
	c.state = p2p.StateConnecting
	c.mu.Unlock()

	// 构建 WebSocket URL
	url := fmt.Sprintf("ws://%s/team/%s/ws", c.endpoint, c.teamID)

	c.logger.Info("connecting to leader",
		"url", url,
		"team_id", c.teamID,
	)

	// 建立连接
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, url, nil)
	if err != nil {
		c.mu.Lock()
		c.state = p2p.StateDisconnected
		c.mu.Unlock()
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	// 发送认证
	if err := c.authenticate(); err != nil {
		conn.Close()
		c.mu.Lock()
		c.state = p2p.StateDisconnected
		c.conn = nil
		c.mu.Unlock()
		return fmt.Errorf("authentication failed: %w", err)
	}

	c.mu.Lock()
	c.state = p2p.StateConnected
	c.lastPing = time.Now()
	c.retryCount = 0
	c.mu.Unlock()

	c.logger.Info("connected to leader",
		"team_id", c.teamID,
		"endpoint", c.endpoint,
	)

	// 通知监听器
	if c.listener != nil {
		c.listener.OnConnect(c.teamID)
	}

	// 启动读写协程
	go c.readPump()
	go c.writePump()

	return nil
}

// authenticate 发送认证
func (c *WebSocketClient) authenticate() error {
	// 发送认证事件
	authEvent, err := p2p.NewEvent(p2p.EventAuth, c.teamID, p2p.AuthPayload{
		MemberID:   c.memberID,
		MemberName: c.memberName,
		Endpoint:   c.localEndpoint,
	})
	if err != nil {
		return err
	}

	data, err := json.Marshal(authEvent)
	if err != nil {
		return err
	}

	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return err
	}

	// 等待认证结果
	c.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, message, err := c.conn.ReadMessage()
	if err != nil {
		return err
	}
	c.conn.SetReadDeadline(time.Time{})

	var event p2p.Event
	if err := json.Unmarshal(message, &event); err != nil {
		return err
	}

	if event.Type != p2p.EventAuthResult {
		return fmt.Errorf("unexpected event type: %s", event.Type)
	}

	var result p2p.AuthResultPayload
	if err := event.ParsePayload(&result); err != nil {
		return err
	}

	if !result.Success {
		return fmt.Errorf("auth failed: %s", result.Error)
	}

	return nil
}

// readPump 读取消息
func (c *WebSocketClient) readPump() {
	defer c.handleDisconnect()

	c.conn.SetReadLimit(512 * 1024)
	c.conn.SetPongHandler(func(string) error {
		c.mu.Lock()
		c.lastPing = time.Now()
		c.mu.Unlock()
		return nil
	})

	for {
		select {
		case <-c.done:
			return
		default:
		}

		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Warn("connection read error",
					"team_id", c.teamID,
					"error", err,
				)
			}
			return
		}

		// 解析事件
		var event p2p.Event
		if err := json.Unmarshal(message, &event); err != nil {
			c.logger.Warn("failed to parse message",
				"error", err,
			)
			continue
		}

		// 处理 Pong
		if event.Type == p2p.EventPong {
			c.mu.Lock()
			c.lastPing = time.Now()
			c.mu.Unlock()
			continue
		}

		// 通知监听器
		if c.listener != nil {
			c.listener.OnEvent(&event)
		}
	}
}

// writePump 写入消息
func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(p2p.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case message := <-c.sendChan:
			c.mu.RLock()
			conn := c.conn
			c.mu.RUnlock()

			if conn == nil {
				continue
			}

			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				c.logger.Warn("failed to write message",
					"team_id", c.teamID,
					"error", err,
				)
				return
			}
		case <-ticker.C:
			// 发送 Ping
			c.mu.RLock()
			conn := c.conn
			c.mu.RUnlock()

			if conn == nil {
				continue
			}

			// 发送应用层 Ping 事件
			pingEvent, _ := p2p.NewEvent(p2p.EventPing, c.teamID, nil)
			data, _ := json.Marshal(pingEvent)
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		}
	}
}

// handleDisconnect 处理断开连接
func (c *WebSocketClient) handleDisconnect() {
	c.mu.Lock()
	wasConnected := c.state == p2p.StateConnected
	c.state = p2p.StateDisconnected
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.mu.Unlock()

	if wasConnected {
		c.logger.Info("disconnected from leader",
			"team_id", c.teamID,
		)

		// 通知监听器
		if c.listener != nil {
			c.listener.OnDisconnect(c.teamID, nil)
		}
	}
}

// Send 发送事件
func (c *WebSocketClient) Send(event *p2p.Event) error {
	c.mu.RLock()
	state := c.state
	c.mu.RUnlock()

	if state != p2p.StateConnected {
		return fmt.Errorf("not connected")
	}

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	select {
	case c.sendChan <- data:
		return nil
	default:
		return fmt.Errorf("send buffer full")
	}
}

// Close 关闭连接
func (c *WebSocketClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	select {
	case <-c.done:
		// 已关闭
	default:
		close(c.done)
	}

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.state = p2p.StateDisconnected

	return nil
}

// GetState 获取连接状态
func (c *WebSocketClient) GetState() p2p.ConnectionState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// IsConnected 是否已连接
func (c *WebSocketClient) IsConnected() bool {
	return c.GetState() == p2p.StateConnected
}

// GetConnectionInfo 获取连接信息
func (c *WebSocketClient) GetConnectionInfo() p2p.ConnectionInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return p2p.ConnectionInfo{
		TeamID:     c.teamID,
		Endpoint:   c.endpoint,
		State:      c.state,
		LastPing:   c.lastPing,
		RetryCount: c.retryCount,
	}
}
