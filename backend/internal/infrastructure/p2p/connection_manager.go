package p2p

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/cocursor/backend/internal/domain/p2p"
	"github.com/cocursor/backend/internal/infrastructure/log"
)

// ConnectionManager 连接管理器
// 管理与多个团队 Leader 的 WebSocket 连接，支持自动重连
type ConnectionManager struct {
	mu           sync.RWMutex
	clients      map[string]*WebSocketClient // teamID -> client
	memberID     string
	memberName   string
	localEndpoint string
	listener     p2p.EventListener
	logger       *slog.Logger
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewConnectionManager 创建连接管理器
func NewConnectionManager(memberID, memberName, localEndpoint string, listener p2p.EventListener) *ConnectionManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &ConnectionManager{
		clients:       make(map[string]*WebSocketClient),
		memberID:      memberID,
		memberName:    memberName,
		localEndpoint: localEndpoint,
		listener:      listener,
		logger:        log.NewModuleLogger("p2p", "connection_manager"),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Connect 连接到指定团队
func (m *ConnectionManager) Connect(teamID, endpoint string) error {
	m.mu.Lock()
	
	// 如果已有连接，先关闭
	if existing, ok := m.clients[teamID]; ok {
		existing.Close()
		delete(m.clients, teamID)
	}

	// 创建新客户端
	client := NewWebSocketClient(teamID, endpoint, m.memberID, m.memberName, m.localEndpoint, m.listener)
	m.clients[teamID] = client
	m.mu.Unlock()

	// 连接
	if err := client.Connect(m.ctx); err != nil {
		m.mu.Lock()
		delete(m.clients, teamID)
		m.mu.Unlock()
		return err
	}

	// 启动重连监控
	go m.monitorConnection(teamID)

	return nil
}

// monitorConnection 监控连接状态，断开时自动重连
func (m *ConnectionManager) monitorConnection(teamID string) {
	retryInterval := p2p.ReconnectMinInterval

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-time.After(time.Second):
		}

		m.mu.RLock()
		client, exists := m.clients[teamID]
		m.mu.RUnlock()

		if !exists {
			// 连接已被移除，停止监控
			return
		}

		if client.IsConnected() {
			// 连接正常，重置重试间隔
			retryInterval = p2p.ReconnectMinInterval
			continue
		}

		// 连接断开，尝试重连
		m.logger.Info("attempting to reconnect",
			"team_id", teamID,
			"retry_interval", retryInterval,
		)

		if err := client.Connect(m.ctx); err != nil {
			m.logger.Warn("reconnect failed",
				"team_id", teamID,
				"error", err,
			)

			// 指数退避
			retryInterval *= 2
			if retryInterval > p2p.ReconnectMaxInterval {
				retryInterval = p2p.ReconnectMaxInterval
			}

			// 等待后重试
			select {
			case <-m.ctx.Done():
				return
			case <-time.After(retryInterval):
			}
		} else {
			m.logger.Info("reconnected successfully",
				"team_id", teamID,
			)
			retryInterval = p2p.ReconnectMinInterval
		}
	}
}

// Disconnect 断开与指定团队的连接
func (m *ConnectionManager) Disconnect(teamID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, exists := m.clients[teamID]
	if !exists {
		return nil
	}

	client.Close()
	delete(m.clients, teamID)

	m.logger.Info("disconnected from team",
		"team_id", teamID,
	)

	return nil
}

// Send 发送事件到指定团队
func (m *ConnectionManager) Send(teamID string, event *p2p.Event) error {
	m.mu.RLock()
	client, exists := m.clients[teamID]
	m.mu.RUnlock()

	if !exists {
		return nil
	}

	return client.Send(event)
}

// IsConnected 检查是否已连接到指定团队
func (m *ConnectionManager) IsConnected(teamID string) bool {
	m.mu.RLock()
	client, exists := m.clients[teamID]
	m.mu.RUnlock()

	return exists && client.IsConnected()
}

// GetConnectionInfo 获取指定团队的连接信息
func (m *ConnectionManager) GetConnectionInfo(teamID string) *p2p.ConnectionInfo {
	m.mu.RLock()
	client, exists := m.clients[teamID]
	m.mu.RUnlock()

	if !exists {
		return nil
	}

	info := client.GetConnectionInfo()
	return &info
}

// GetAllConnections 获取所有连接信息
func (m *ConnectionManager) GetAllConnections() []p2p.ConnectionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var infos []p2p.ConnectionInfo
	for _, client := range m.clients {
		infos = append(infos, client.GetConnectionInfo())
	}
	return infos
}

// Close 关闭所有连接
func (m *ConnectionManager) Close() error {
	m.cancel()

	m.mu.Lock()
	defer m.mu.Unlock()

	for teamID, client := range m.clients {
		client.Close()
		m.logger.Info("closed connection",
			"team_id", teamID,
		)
	}
	m.clients = make(map[string]*WebSocketClient)

	return nil
}

// UpdateLocalEndpoint 更新本地端点
func (m *ConnectionManager) UpdateLocalEndpoint(endpoint string) {
	m.mu.Lock()
	m.localEndpoint = endpoint
	m.mu.Unlock()
}

// UpdateIdentity 更新身份信息
func (m *ConnectionManager) UpdateIdentity(memberID, memberName string) {
	m.mu.Lock()
	m.memberID = memberID
	m.memberName = memberName
	m.mu.Unlock()
}
