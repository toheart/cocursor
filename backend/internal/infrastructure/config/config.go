package config

// Config 应用配置
type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	WebSocket  WebSocketConfig
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
