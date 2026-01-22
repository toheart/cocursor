package config

// Config 应用配置
type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	WebSocket  WebSocketConfig
	Cursor     CursorConfig
}

// ServerConfig 服务器配置
type ServerConfig struct {
	HTTPPort string // 固定端口，用于单例锁
	MCPPort  string
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Path string
}

// WebSocketConfig WebSocket 配置
type WebSocketConfig struct {
	ReadBufferSize  int
	WriteBufferSize int
}

// CursorConfig Cursor 路径配置
// 用于用户自定义 Cursor 数据目录路径（主要用于 WSL 等特殊环境）
type CursorConfig struct {
	// UserDataDir Cursor 用户数据目录
	// 例如: /mnt/c/Users/xxx/AppData/Roaming/Cursor/User
	// 留空表示自动检测
	UserDataDir string

	// ProjectsDir Cursor 项目目录（存放 agent-transcripts）
	// 例如: /mnt/c/Users/xxx/.cursor/projects
	// 留空表示自动检测
	ProjectsDir string
}

// NewConfig 创建配置（默认值）
func NewConfig() *Config {
	return &Config{
		Server: ServerConfig{
			HTTPPort: ":19960",
			MCPPort:  ":19961",
		},
		Database: DatabaseConfig{
			Path: "",
		},
		WebSocket: WebSocketConfig{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		Cursor: CursorConfig{
			UserDataDir: "", // 空表示自动检测
			ProjectsDir: "", // 空表示自动检测
		},
	}
}

// NewDatabaseConfig 创建数据库配置
func NewDatabaseConfig(cfg *Config) *DatabaseConfig {
	return &cfg.Database
}

// NewServerConfig 创建服务器配置
func NewServerConfig(cfg *Config) *ServerConfig {
	return &cfg.Server
}

// NewCursorConfig 创建 Cursor 配置
func NewCursorConfig(cfg *Config) *CursorConfig {
	return &cfg.Cursor
}
