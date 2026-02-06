package team

import (
	"encoding/json"
	"fmt"
	"time"
)

// SharedSession 分享的会话
type SharedSession struct {
	ID           string           `json:"id"`                      // 分享记录唯一 ID
	TeamID       string           `json:"team_id"`                 // 所属团队
	SharerID     string           `json:"sharer_id"`               // 分享者成员 ID
	SharerName   string           `json:"sharer_name"`             // 分享者名称
	SessionID    string           `json:"session_id"`              // 原始会话 ID
	Title        string           `json:"title"`                   // 会话标题
	Messages     json.RawMessage  `json:"messages"`                // 消息内容（JSON 存储）
	MessageCount int              `json:"message_count"`           // 消息数量
	Description  string           `json:"description,omitempty"`   // 可选分享说明
	SharedAt     time.Time        `json:"shared_at"`               // 分享时间
	CommentCount int              `json:"comment_count"`           // 评论数
}

// SharedSessionListItem 分享列表项（不含完整消息）
type SharedSessionListItem struct {
	ID           string    `json:"id"`
	TeamID       string    `json:"team_id"`
	SharerID     string    `json:"sharer_id"`
	SharerName   string    `json:"sharer_name"`
	SessionID    string    `json:"session_id"`
	Title        string    `json:"title"`
	MessageCount int       `json:"message_count"`
	Description  string    `json:"description,omitempty"`
	SharedAt     time.Time `json:"shared_at"`
	CommentCount int       `json:"comment_count"`
}

// ToListItem 转换为列表项
func (s *SharedSession) ToListItem() SharedSessionListItem {
	return SharedSessionListItem{
		ID:           s.ID,
		TeamID:       s.TeamID,
		SharerID:     s.SharerID,
		SharerName:   s.SharerName,
		SessionID:    s.SessionID,
		Title:        s.Title,
		MessageCount: s.MessageCount,
		Description:  s.Description,
		SharedAt:     s.SharedAt,
		CommentCount: s.CommentCount,
	}
}

// SessionComment 会话评论
type SessionComment struct {
	ID         string    `json:"id"`                   // 评论唯一 ID
	ShareID    string    `json:"share_id"`             // 关联的分享 ID
	AuthorID   string    `json:"author_id"`            // 评论者成员 ID
	AuthorName string    `json:"author_name"`          // 评论者名称
	Content    string    `json:"content"`              // 评论内容（Markdown）
	Mentions   []string  `json:"mentions,omitempty"`   // @提及的成员 ID 列表
	CreatedAt  time.Time `json:"created_at"`           // 创建时间
}

// ShareSessionRequest 分享会话请求
type ShareSessionRequest struct {
	SessionID   string          `json:"session_id"`            // 原始会话 ID
	Title       string          `json:"title"`                 // 会话标题
	Messages    json.RawMessage `json:"messages"`              // 消息内容
	Description string          `json:"description,omitempty"` // 可选分享说明
}

// Validate 验证分享请求
func (r *ShareSessionRequest) Validate() error {
	if r.SessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	if r.Title == "" {
		return fmt.Errorf("title is required")
	}
	if len(r.Messages) == 0 {
		return fmt.Errorf("messages is required")
	}
	return nil
}

// AddCommentRequest 添加评论请求
type AddCommentRequest struct {
	Content  string   `json:"content"`            // 评论内容
	Mentions []string `json:"mentions,omitempty"` // @提及的成员 ID 列表
}

// Validate 验证评论请求
func (r *AddCommentRequest) Validate() error {
	if r.Content == "" {
		return fmt.Errorf("content is required")
	}
	return nil
}
