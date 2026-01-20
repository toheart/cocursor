package team

import "errors"

// 身份相关错误
var (
	// ErrIdentityNotFound 身份未设置
	ErrIdentityNotFound = errors.New("identity not found, please set your identity first")
	// ErrIdentityNameRequired 身份名称必填
	ErrIdentityNameRequired = errors.New("identity name is required")
)

// 团队相关错误
var (
	// ErrTeamNotFound 团队不存在
	ErrTeamNotFound = errors.New("team not found")
	// ErrTeamAlreadyExists 已经创建了团队
	ErrTeamAlreadyExists = errors.New("already created a team as leader")
	// ErrTeamNameRequired 团队名称必填
	ErrTeamNameRequired = errors.New("team name is required")
	// ErrNotTeamLeader 不是团队 Leader
	ErrNotTeamLeader = errors.New("only team leader can perform this action")
	// ErrIsTeamLeader Leader 不能离开团队
	ErrIsTeamLeader = errors.New("leader cannot leave team, use dissolve instead")
	// ErrNotTeamMember 不是团队成员
	ErrNotTeamMember = errors.New("not a member of this team")
	// ErrAlreadyTeamMember 已经是团队成员
	ErrAlreadyTeamMember = errors.New("already a member of this team")
)

// 网络相关错误
var (
	// ErrLeaderOffline Leader 离线
	ErrLeaderOffline = errors.New("team leader is offline")
	// ErrMemberOffline 成员离线
	ErrMemberOffline = errors.New("team member is offline")
	// ErrConnectionFailed 连接失败
	ErrConnectionFailed = errors.New("failed to connect to team")
	// ErrNoValidInterface 没有有效的网络接口
	ErrNoValidInterface = errors.New("no valid network interface found")
	// ErrPortInUse 端口被占用
	ErrPortInUse = errors.New("port is already in use")
)

// 技能相关错误
var (
	// ErrSkillNotFound 技能不存在
	ErrSkillNotFound = errors.New("skill not found")
	// ErrSkillAlreadyExists 技能已存在
	ErrSkillAlreadyExists = errors.New("skill already exists")
	// ErrInvalidSkillDirectory 无效的技能目录
	ErrInvalidSkillDirectory = errors.New("invalid skill directory")
	// ErrSkillMDNotFound 缺少 SKILL.md
	ErrSkillMDNotFound = errors.New("SKILL.md not found in directory")
	// ErrInvalidSkillFrontmatter 无效的 frontmatter
	ErrInvalidSkillFrontmatter = errors.New("invalid SKILL.md frontmatter")
	// ErrAuthorOffline 作者离线无法下载
	ErrAuthorOffline = errors.New("skill author is offline, cannot download")
	// ErrChecksumMismatch 校验和不匹配
	ErrChecksumMismatch = errors.New("checksum mismatch, file may be corrupted")
)

// WebSocket 相关错误
var (
	// ErrWebSocketNotConnected WebSocket 未连接
	ErrWebSocketNotConnected = errors.New("websocket not connected")
	// ErrWebSocketAuthFailed WebSocket 认证失败
	ErrWebSocketAuthFailed = errors.New("websocket authentication failed")
)
