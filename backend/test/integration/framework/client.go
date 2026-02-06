//go:build integration
// +build integration

// APIClient 基于 resty 封装的 HTTP 客户端，直接复用业务结构体
package framework

import (
	"encoding/json"
	"fmt"
	"time"

	domainTeam "github.com/cocursor/backend/internal/domain/team"
	"github.com/go-resty/resty/v2"
)

// APIClient 测试用 HTTP 客户端
type APIClient struct {
	client  *resty.Client
	baseURL string
}

// NewAPIClient 创建测试用 HTTP 客户端
func NewAPIClient(baseURL string) *APIClient {
	client := resty.New().
		SetBaseURL(baseURL).
		SetTimeout(10 * time.Second).
		SetHeader("Content-Type", "application/json")

	return &APIClient{
		client:  client,
		baseURL: baseURL,
	}
}

// --- 通用响应结构 ---

// APIResponse 通用 API 响应（复用 response.Response 的 JSON 结构）
type APIResponse[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data,omitempty"`
}

// --- 各接口 Data 结构（与 handler 返回的 gin.H 对应） ---

// IdentityData POST /team/identity 响应 data
type IdentityData struct {
	Identity *domainTeam.Identity `json:"identity"`
}

// GetIdentityData GET /team/identity 响应 data
type GetIdentityData struct {
	Exists   bool                 `json:"exists"`
	Identity *domainTeam.Identity `json:"identity"`
}

// TeamData 创建/加入团队响应 data
type TeamData struct {
	Team *domainTeam.Team `json:"team"`
}

// TeamListData 团队列表响应 data
type TeamListData struct {
	Teams []domainTeam.Team `json:"teams"`
	Total int               `json:"total"`
}

// TeamMembersData 团队成员响应 data
type TeamMembersData struct {
	Members []domainTeam.TeamMember `json:"members"`
	Total   int                     `json:"total"`
}

// MessageData 通用消息响应 data
type MessageData struct {
	Message string `json:"message"`
}

// DiscoveredTeamInfo 发现的团队信息（与 P2P 层 DiscoveredTeamInfo 对应）
type DiscoveredTeamInfo struct {
	TeamID      string `json:"team_id"`
	Name        string `json:"name"`
	LeaderName  string `json:"leader_name"`
	Endpoint    string `json:"endpoint"`
	MemberCount int    `json:"member_count"`
	Version     string `json:"version"`
}

// DiscoverData 发现团队响应 data
type DiscoverData struct {
	Teams []DiscoveredTeamInfo `json:"teams"`
}

// do 执行请求并统一处理成功/错误响应的 JSON 解析
// resty 的 SetResult 仅在 2xx 时解析，SetError 在 4xx/5xx 时解析
// 由于两者的 code/message 字段一致，用同类型接收即可
func do[T any](r *resty.Request, result *APIResponse[T]) *resty.Request {
	return r.SetResult(result).SetError(result)
}

// --- 健康检查 ---

// HealthCheck 健康检查
func (c *APIClient) HealthCheck() error {
	resp, err := c.client.R().Get("/health")
	if err != nil {
		return err
	}
	if resp.StatusCode() != 200 {
		return fmt.Errorf("health check failed: status %d", resp.StatusCode())
	}
	return nil
}

// --- 身份管理 ---

// CreateIdentity 创建身份
func (c *APIClient) CreateIdentity(name string) (*APIResponse[IdentityData], error) {
	var result APIResponse[IdentityData]
	_, err := do(c.client.R().SetBody(map[string]string{"name": name}), &result).
		Post("/api/v1/team/identity")
	return &result, err
}

// GetIdentity 获取身份
func (c *APIClient) GetIdentity() (*APIResponse[GetIdentityData], error) {
	var result APIResponse[GetIdentityData]
	_, err := do(c.client.R(), &result).
		Get("/api/v1/team/identity")
	return &result, err
}

// --- 团队管理 ---

// CreateTeam 创建团队
func (c *APIClient) CreateTeam(name string) (*APIResponse[TeamData], error) {
	var result APIResponse[TeamData]
	_, err := do(c.client.R().SetBody(map[string]string{"name": name}), &result).
		Post("/api/v1/team/create")
	return &result, err
}

// ListTeams 获取团队列表
func (c *APIClient) ListTeams() (*APIResponse[TeamListData], error) {
	var result APIResponse[TeamListData]
	_, err := do(c.client.R(), &result).
		Get("/api/v1/team/list")
	return &result, err
}

// JoinTeam 加入团队（通过 leader endpoint）
func (c *APIClient) JoinTeam(leaderEndpoint string) (*APIResponse[TeamData], error) {
	var result APIResponse[TeamData]
	_, err := do(c.client.R().SetBody(map[string]string{"endpoint": leaderEndpoint}), &result).
		Post("/api/v1/team/join")
	return &result, err
}

// LeaveTeam 离开团队
func (c *APIClient) LeaveTeam(teamID string) (*APIResponse[MessageData], error) {
	var result APIResponse[MessageData]
	_, err := do(c.client.R(), &result).
		Post(fmt.Sprintf("/api/v1/team/%s/leave", teamID))
	return &result, err
}

// DissolveTeam 解散团队
func (c *APIClient) DissolveTeam(teamID string) (*APIResponse[MessageData], error) {
	var result APIResponse[MessageData]
	_, err := do(c.client.R(), &result).
		Post(fmt.Sprintf("/api/v1/team/%s/dissolve", teamID))
	return &result, err
}

// GetTeamMembers 获取团队成员
func (c *APIClient) GetTeamMembers(teamID string) (*APIResponse[TeamMembersData], error) {
	var result APIResponse[TeamMembersData]
	_, err := do(c.client.R(), &result).
		Get(fmt.Sprintf("/api/v1/team/%s/members", teamID))
	return &result, err
}

// DiscoverTeams 发现局域网团队
func (c *APIClient) DiscoverTeams(timeoutSec int) (*APIResponse[DiscoverData], error) {
	var result APIResponse[DiscoverData]
	_, err := do(c.client.R().SetQueryParam("timeout", fmt.Sprintf("%d", timeoutSec)), &result).
		Get("/api/v1/team/discover")
	return &result, err
}

// --- 协作功能 ---

// SuccessData 通用 success 响应 data
type SuccessData struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// UpdateWorkStatus 更新工作状态
func (c *APIClient) UpdateWorkStatus(teamID string, projectName, currentFile string, visible bool) (*APIResponse[SuccessData], error) {
	var result APIResponse[SuccessData]
	_, err := do(c.client.R().SetBody(map[string]interface{}{
		"project_name":   projectName,
		"current_file":   currentFile,
		"status_visible": visible,
	}), &result).
		Post(fmt.Sprintf("/api/v1/team/%s/status", teamID))
	return &result, err
}

// DailySummariesData 日报列表响应 data
type DailySummariesData struct {
	Summaries interface{} `json:"summaries"`
	Date      string      `json:"date"`
}

// ShareDailySummary 分享日报
func (c *APIClient) ShareDailySummary(teamID, date string) (*APIResponse[SuccessData], error) {
	var result APIResponse[SuccessData]
	_, err := do(c.client.R().SetBody(map[string]string{
		"date": date,
	}), &result).
		Post(fmt.Sprintf("/api/v1/team/%s/daily-summaries/share", teamID))
	return &result, err
}

// GetDailySummaries 获取团队日报列表
func (c *APIClient) GetDailySummaries(teamID, date string) (*APIResponse[DailySummariesData], error) {
	var result APIResponse[DailySummariesData]
	_, err := do(c.client.R().SetQueryParam("date", date), &result).
		Get(fmt.Sprintf("/api/v1/team/%s/daily-summaries", teamID))
	return &result, err
}

// --- 会话分享 ---

// ShareSessionData 分享会话响应 data
type ShareSessionData struct {
	ShareID string `json:"share_id"`
}

// SharedSessionsData 会话列表响应 data
type SharedSessionsData struct {
	Sessions []domainTeam.SharedSessionListItem `json:"sessions"`
	Total    int                                `json:"total"`
}

// SharedSessionDetailData 会话详情响应 data
type SharedSessionDetailData struct {
	Session  *domainTeam.SharedSession   `json:"session"`
	Comments []domainTeam.SessionComment `json:"comments"`
}

// CommentData 评论响应 data
type CommentData struct {
	CommentID string `json:"comment_id"`
}

// ShareSession 分享会话
func (c *APIClient) ShareSession(teamID string, req *domainTeam.ShareSessionRequest) (*APIResponse[ShareSessionData], error) {
	var result APIResponse[ShareSessionData]
	_, err := do(c.client.R().SetBody(req), &result).
		Post(fmt.Sprintf("/api/v1/team/%s/sessions/share", teamID))
	return &result, err
}

// GetSharedSessions 获取分享会话列表
func (c *APIClient) GetSharedSessions(teamID string, limit, offset int) (*APIResponse[SharedSessionsData], error) {
	var result APIResponse[SharedSessionsData]
	_, err := do(c.client.R().
		SetQueryParam("limit", fmt.Sprintf("%d", limit)).
		SetQueryParam("offset", fmt.Sprintf("%d", offset)), &result).
		Get(fmt.Sprintf("/api/v1/team/%s/sessions", teamID))
	return &result, err
}

// GetSharedSessionDetail 获取分享会话详情
func (c *APIClient) GetSharedSessionDetail(teamID, shareID string) (*APIResponse[SharedSessionDetailData], error) {
	var result APIResponse[SharedSessionDetailData]
	_, err := do(c.client.R(), &result).
		Get(fmt.Sprintf("/api/v1/team/%s/sessions/%s", teamID, shareID))
	return &result, err
}

// AddComment 添加评论
func (c *APIClient) AddComment(teamID, shareID string, content string, mentions []string) (*APIResponse[CommentData], error) {
	var result APIResponse[CommentData]
	_, err := do(c.client.R().SetBody(domainTeam.AddCommentRequest{
		Content:  content,
		Mentions: mentions,
	}), &result).
		Post(fmt.Sprintf("/api/v1/team/%s/sessions/%s/comments", teamID, shareID))
	return &result, err
}

// --- 网络管理 ---

// NetworkInterfacesData 网络接口响应 data
type NetworkInterfacesData struct {
	Interfaces      interface{} `json:"interfaces"`
	Config          interface{} `json:"config"`
	CurrentEndpoint interface{} `json:"current_endpoint"`
}

// GetNetworkInterfaces 获取网络接口列表
func (c *APIClient) GetNetworkInterfaces() (*APIResponse[NetworkInterfacesData], error) {
	var result APIResponse[NetworkInterfacesData]
	_, err := do(c.client.R(), &result).
		Get("/api/v1/team/network/interfaces")
	return &result, err
}

// --- 辅助方法 ---

// MustCreateIdentityAndTeam 创建身份并建团的快捷方法（测试辅助）
func (c *APIClient) MustCreateIdentityAndTeam(name, teamName string) (identityID, teamID string, err error) {
	idResp, err := c.CreateIdentity(name)
	if err != nil {
		return "", "", fmt.Errorf("create identity: %w", err)
	}
	if idResp.Code != 0 {
		return "", "", fmt.Errorf("create identity failed: %s", idResp.Message)
	}

	teamResp, err := c.CreateTeam(teamName)
	if err != nil {
		return "", "", fmt.Errorf("create team: %w", err)
	}
	if teamResp.Code != 0 {
		return "", "", fmt.Errorf("create team failed: %s", teamResp.Message)
	}

	return idResp.Data.Identity.ID, teamResp.Data.Team.ID, nil
}

// MustJoinTeam 创建身份并加入团队的快捷方法（测试辅助）
func (c *APIClient) MustJoinTeam(name, leaderEndpoint string) (identityID string, err error) {
	idResp, err := c.CreateIdentity(name)
	if err != nil {
		return "", fmt.Errorf("create identity: %w", err)
	}
	if idResp.Code != 0 {
		return "", fmt.Errorf("create identity failed: %s", idResp.Message)
	}

	joinResp, err := c.JoinTeam(leaderEndpoint)
	if err != nil {
		return "", fmt.Errorf("join team: %w", err)
	}
	if joinResp.Code != 0 {
		return "", fmt.Errorf("join team failed: %s", joinResp.Message)
	}

	return idResp.Data.Identity.ID, nil
}

// MakeMessages 构造 mock 消息 JSON 用于 ShareSession
func MakeMessages(messages []map[string]string) json.RawMessage {
	data, _ := json.Marshal(messages)
	return data
}
