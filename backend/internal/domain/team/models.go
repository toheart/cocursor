package team

import (
	"fmt"
	"time"
)

// Team 团队
type Team struct {
	ID             string    `json:"id"`              // 团队 ID (UUID)
	Name           string    `json:"name"`            // 团队名称
	LeaderID       string    `json:"leader_id"`       // Leader 成员 ID
	LeaderName     string    `json:"leader_name"`     // Leader 名称
	LeaderEndpoint string    `json:"leader_endpoint"` // Leader 端点 (IP:Port)
	MemberCount    int       `json:"member_count"`    // 成员数量
	SkillCount     int       `json:"skill_count"`     // 技能数量
	CreatedAt      time.Time `json:"created_at"`      // 创建时间
	JoinedAt       time.Time `json:"joined_at"`       // 本机加入时间
	IsLeader       bool      `json:"is_leader"`       // 当前用户是否是 Leader
	LeaderOnline   bool      `json:"leader_online"`   // Leader 是否在线
	LastSyncAt     time.Time `json:"last_sync_at"`    // 最后同步时间
}

// TeamMember 团队成员
type TeamMember struct {
	ID         string            `json:"id"`                    // 成员 ID
	Name       string            `json:"name"`                  // 成员名称
	Endpoint   string            `json:"endpoint"`              // 成员端点 (IP:Port)
	IsLeader   bool              `json:"is_leader"`             // 是否是 Leader
	IsOnline   bool              `json:"is_online"`             // 是否在线
	JoinedAt   time.Time         `json:"joined_at"`             // 加入时间
	WorkStatus *MemberWorkStatus `json:"work_status,omitempty"` // 工作状态
}

// MemberWorkStatus 成员工作状态
type MemberWorkStatus struct {
	ProjectName   string    `json:"project_name"`   // 当前项目名
	CurrentFile   string    `json:"current_file"`   // 当前文件（相对路径）
	LastActiveAt  time.Time `json:"last_active_at"` // 最后活跃时间
	StatusVisible bool      `json:"status_visible"` // 是否公开状态
}

// TeamDailySummary 团队日报
type TeamDailySummary struct {
	MemberID      string    `json:"member_id"`      // 成员 ID
	MemberName    string    `json:"member_name"`    // 成员名称
	Date          string    `json:"date"`           // 日期 YYYY-MM-DD
	Summary       string    `json:"summary"`        // Markdown 内容
	Language      string    `json:"language"`       // 语言 zh/en
	SharedAt      time.Time `json:"shared_at"`      // 分享时间
	TotalSessions int       `json:"total_sessions"` // 会话总数
	ProjectCount  int       `json:"project_count"`  // 项目数量
}

// TeamDailySummaryIndex 团队日报索引（Leader 维护）
type TeamDailySummaryIndex struct {
	TeamID    string             `json:"team_id"`    // 团队 ID
	UpdatedAt time.Time          `json:"updated_at"` // 更新时间
	Summaries []TeamDailySummary `json:"summaries"`  // 日报列表（不含完整 Summary 内容）
}

// FindSummary 查找日报
func (idx *TeamDailySummaryIndex) FindSummary(date, memberID string) *TeamDailySummary {
	for i := range idx.Summaries {
		if idx.Summaries[i].Date == date && idx.Summaries[i].MemberID == memberID {
			return &idx.Summaries[i]
		}
	}
	return nil
}

// AddOrUpdateSummary 添加或更新日报
func (idx *TeamDailySummaryIndex) AddOrUpdateSummary(entry TeamDailySummary) {
	// 索引中不存储完整的 Summary 内容
	entry.Summary = ""
	for i := range idx.Summaries {
		if idx.Summaries[i].Date == entry.Date && idx.Summaries[i].MemberID == entry.MemberID {
			idx.Summaries[i] = entry
			idx.UpdatedAt = time.Now()
			return
		}
	}
	idx.Summaries = append(idx.Summaries, entry)
	idx.UpdatedAt = time.Now()
}

// GetSummariesByDate 按日期获取日报列表
func (idx *TeamDailySummaryIndex) GetSummariesByDate(date string) []TeamDailySummary {
	var result []TeamDailySummary
	for _, s := range idx.Summaries {
		if s.Date == date {
			result = append(result, s)
		}
	}
	return result
}

// Identity 本机身份
type Identity struct {
	ID        string    `json:"id"`         // 成员 ID (UUID)
	Name      string    `json:"name"`       // 显示名称
	CreatedAt time.Time `json:"created_at"` // 创建时间
	UpdatedAt time.Time `json:"updated_at"` // 更新时间
}

// Validate 验证身份信息
func (i *Identity) Validate() error {
	if i.ID == "" {
		return fmt.Errorf("identity id is required")
	}
	if i.Name == "" {
		return fmt.Errorf("identity name is required")
	}
	return nil
}

// MemberEndpoint 成员端点（支持多地址）
type MemberEndpoint struct {
	PrimaryIP   string   `json:"primary_ip"`             // 主要 IP
	AllIPs      []string `json:"all_ips"`                // 所有可用 IP
	Port        int      `json:"port"`                   // 端口
	PreferredIF string   `json:"preferred_if,omitempty"` // 首选网卡名称
}

// GetAddress 获取主地址
func (e *MemberEndpoint) GetAddress() string {
	return fmt.Sprintf("%s:%d", e.PrimaryIP, e.Port)
}

// GetAllAddresses 获取所有可能的地址（用于连接尝试）
func (e *MemberEndpoint) GetAllAddresses() []string {
	addrs := make([]string, 0, len(e.AllIPs))
	for _, ip := range e.AllIPs {
		addrs = append(addrs, fmt.Sprintf("%s:%d", ip, e.Port))
	}
	return addrs
}

// NetworkInterface 网络接口信息
type NetworkInterface struct {
	Name       string   `json:"name"`        // 接口名称，如 "en0", "eth0"
	Addresses  []string `json:"addresses"`   // IPv4 地址列表
	IsUp       bool     `json:"is_up"`       // 是否启用
	IsLoopback bool     `json:"is_loopback"` // 是否回环
	IsVirtual  bool     `json:"is_virtual"`  // 是否虚拟网卡（VMware, VirtualBox, Docker等）
}

// NetworkConfig 网卡偏好配置
type NetworkConfig struct {
	PreferredInterface string    `json:"preferred_interface,omitempty"` // 首选网卡名称
	PreferredIP        string    `json:"preferred_ip,omitempty"`        // 首选 IP
	LastUpdated        time.Time `json:"last_updated"`
}

// TeamSkillEntry 团队技能目录条目
type TeamSkillEntry struct {
	PluginID       string    `json:"plugin_id"`                   // 插件 ID
	Name           string    `json:"name"`                        // 技能名称（英文）
	NameZhCN       string    `json:"name_zh_cn,omitempty"`        // 技能名称（中文）
	Description    string    `json:"description"`                 // 技能描述（英文）
	DescZhCN       string    `json:"description_zh_cn,omitempty"` // 技能描述（中文）
	Version        string    `json:"version"`                     // 版本号
	Category       string    `json:"category,omitempty"`          // 分类（productivity/creative/design/tools/other）
	Scope          string    `json:"scope"`                       // 范围：global | project
	AuthorID       string    `json:"author_id"`                   // 作者成员 ID
	AuthorName     string    `json:"author_name"`                 // 作者名称
	AuthorEndpoint string    `json:"author_endpoint"`             // 作者端点（用于下载）
	PublishedAt    time.Time `json:"published_at"`                // 发布时间
	FileCount      int       `json:"file_count"`                  // 文件数量
	TotalSize      int64     `json:"total_size"`                  // 总大小（字节）
	Checksum       string    `json:"checksum"`                    // 校验和
}

// SkillMetadata 用户提交的技能元数据（发布时使用）
type SkillMetadata struct {
	PluginID    string `json:"plugin_id"`                   // 技能 ID（必填）
	Name        string `json:"name"`                        // 英文名称（必填）
	NameZhCN    string `json:"name_zh_cn,omitempty"`        // 中文名称（可选）
	Description string `json:"description"`                 // 英文描述（必填）
	DescZhCN    string `json:"description_zh_cn,omitempty"` // 中文描述（可选）
	Version     string `json:"version"`                     // 版本号（必填）
	Category    string `json:"category"`                    // 分类（必填）
	Author      string `json:"author"`                      // 作者（必填）
}

// Validate 验证技能元数据
func (m *SkillMetadata) Validate() error {
	if m.PluginID == "" {
		return fmt.Errorf("plugin_id is required")
	}
	if m.Name == "" {
		return fmt.Errorf("name is required")
	}
	if m.Description == "" {
		return fmt.Errorf("description is required")
	}
	if m.Version == "" {
		return fmt.Errorf("version is required")
	}
	if m.Category == "" {
		return fmt.Errorf("category is required")
	}
	if m.Author == "" {
		return fmt.Errorf("author is required")
	}
	// 验证分类有效性
	validCategories := map[string]bool{
		"productivity": true,
		"creative":     true,
		"design":       true,
		"tools":        true,
		"other":        true,
	}
	if !validCategories[m.Category] {
		return fmt.Errorf("invalid category: %s", m.Category)
	}
	return nil
}

// TeamSkillIndex 团队技能目录
type TeamSkillIndex struct {
	TeamID    string           `json:"team_id"`    // 团队 ID
	UpdatedAt time.Time        `json:"updated_at"` // 更新时间
	Skills    []TeamSkillEntry `json:"skills"`     // 技能列表
}

// FindSkill 查找技能
func (idx *TeamSkillIndex) FindSkill(pluginID string) *TeamSkillEntry {
	for i := range idx.Skills {
		if idx.Skills[i].PluginID == pluginID {
			return &idx.Skills[i]
		}
	}
	return nil
}

// AddOrUpdateSkill 添加或更新技能
func (idx *TeamSkillIndex) AddOrUpdateSkill(entry TeamSkillEntry) {
	for i := range idx.Skills {
		if idx.Skills[i].PluginID == entry.PluginID {
			idx.Skills[i] = entry
			idx.UpdatedAt = time.Now()
			return
		}
	}
	idx.Skills = append(idx.Skills, entry)
	idx.UpdatedAt = time.Now()
}

// RemoveSkill 移除技能
func (idx *TeamSkillIndex) RemoveSkill(pluginID string) bool {
	for i := range idx.Skills {
		if idx.Skills[i].PluginID == pluginID {
			idx.Skills = append(idx.Skills[:i], idx.Skills[i+1:]...)
			idx.UpdatedAt = time.Now()
			return true
		}
	}
	return false
}

// DiscoveredTeam 发现的团队（mDNS 发现结果）
type DiscoveredTeam struct {
	TeamID      string `json:"team_id"`      // 团队 ID
	Name        string `json:"name"`         // 团队名称
	LeaderName  string `json:"leader_name"`  // Leader 名称
	Endpoint    string `json:"endpoint"`     // 端点 (IP:Port)
	MemberCount int    `json:"member_count"` // 成员数量
	Version     string `json:"version"`      // cocursor 版本
}

// JoinRequest 加入团队请求
type JoinRequest struct {
	MemberID   string `json:"member_id"`   // 成员 ID
	MemberName string `json:"member_name"` // 成员名称
	Endpoint   string `json:"endpoint"`    // 成员端点
}

// JoinResponse 加入团队响应
type JoinResponse struct {
	Success    bool            `json:"success"`
	Team       *Team           `json:"team,omitempty"`
	Members    []TeamMember    `json:"members,omitempty"`
	SkillIndex *TeamSkillIndex `json:"skill_index,omitempty"`
	Error      string          `json:"error,omitempty"`
}
