package team

import (
	"log/slog"

	"github.com/cocursor/backend/internal/infrastructure/log"
	"github.com/cocursor/backend/internal/infrastructure/marketplace"
)

// TeamComponents 包含团队功能所需的所有组件
type TeamComponents struct {
	TeamService     *TeamService
	SyncService     *SyncService
	SkillPublisher  *marketplace.TeamSkillPublisher
	SkillDownloader *marketplace.TeamSkillDownloader
	SkillLoader     *marketplace.TeamSkillLoader
}

// TeamFactory 团队服务工厂
type TeamFactory struct {
	logger *slog.Logger
}

// NewTeamFactory 创建团队服务工厂
func NewTeamFactory() *TeamFactory {
	return &TeamFactory{
		logger: log.NewModuleLogger("team", "factory"),
	}
}

// Initialize 初始化所有团队相关组件
// port: P2P 服务端口（默认 19960）
// version: 应用版本号
func (f *TeamFactory) Initialize(port int, version string) (*TeamComponents, error) {
	f.logger.Info("initializing team components",
		"port", port,
		"version", version,
	)

	// 创建 TeamService（它会自动初始化存储、P2P 组件等）
	teamService, err := NewTeamService(port, version)
	if err != nil {
		f.logger.Error("failed to create team service", "error", err)
		return nil, err
	}

	// 创建技能相关组件
	skillPublisher := marketplace.NewTeamSkillPublisher()
	skillDownloader := marketplace.NewTeamSkillDownloader()
	skillLoader := marketplace.NewTeamSkillLoader()

	// 创建同步服务
	// 注意：同步服务需要团队存储，但这些已经在 TeamService 中创建
	// 这里创建一个简化版本，实际使用时需要从 TeamService 获取存储引用
	syncService := NewSyncService(nil, nil)

	f.logger.Info("team components initialized successfully")

	return &TeamComponents{
		TeamService:     teamService,
		SyncService:     syncService,
		SkillPublisher:  skillPublisher,
		SkillDownloader: skillDownloader,
		SkillLoader:     skillLoader,
	}, nil
}
