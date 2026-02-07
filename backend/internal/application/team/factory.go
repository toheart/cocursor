package team

import (
	"database/sql"
	"log/slog"

	"github.com/cocursor/backend/internal/infrastructure/log"
	"github.com/cocursor/backend/internal/infrastructure/marketplace"
	"github.com/cocursor/backend/internal/infrastructure/storage"
)

// TeamComponents 包含团队功能所需的所有组件
type TeamComponents struct {
	// 核心服务
	TeamService           *TeamService
	IdentityService       *IdentityService
	NetworkConfigService  *NetworkConfigService
	SyncService           *SyncService
	CollaborationService  *CollaborationService
	WeeklyReportService   *WeeklyReportService
	SessionSharingService *SessionSharingService

	// 技能相关
	SkillPublisher  *marketplace.TeamSkillPublisher
	SkillDownloader *marketplace.TeamSkillDownloader
	SkillLoader     *marketplace.TeamSkillLoader

	// Config 共享配置
	Config *TeamServiceConfig
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
// dailySummaryRepo: 日报仓储（用于分享本地日报）
// db: 数据库连接（用于会话分享存储）
func (f *TeamFactory) Initialize(port int, version string, dailySummaryRepo storage.DailySummaryRepository, db *sql.DB) (*TeamComponents, error) {
	f.logger.Info("initializing team components",
		"port", port,
		"version", version,
	)

	// 创建共享配置
	config := NewTeamServiceConfig(port, version)

	// 创建身份服务
	identityService, err := NewIdentityService()
	if err != nil {
		f.logger.Error("failed to create identity service", "error", err)
		return nil, err
	}

	// 创建网络配置服务
	networkConfigService, err := NewNetworkConfigService(port)
	if err != nil {
		f.logger.Error("failed to create network config service", "error", err)
		return nil, err
	}

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

	// 创建同步服务（通过接口依赖 TeamService）
	syncService := NewSyncService(teamService, config)

	// 创建协作服务
	collaborationService := NewCollaborationService(teamService, dailySummaryRepo)

	// 创建周报服务（使用共享配置）
	weeklyReportService := NewWeeklyReportServiceWithConfig(teamService, config)

	// 创建会话分享服务（使用共享配置）
	var sessionSharingService *SessionSharingService
	if db != nil {
		var err error
		sessionSharingService, err = NewSessionSharingServiceWithConfig(teamService, db, config)
		if err != nil {
			f.logger.Warn("failed to create session sharing service", "error", err)
			// 不阻塞其他服务初始化
		}
	}

	f.logger.Info("team components initialized successfully")

	return &TeamComponents{
		TeamService:           teamService,
		IdentityService:       identityService,
		NetworkConfigService:  networkConfigService,
		SyncService:           syncService,
		CollaborationService:  collaborationService,
		WeeklyReportService:   weeklyReportService,
		SessionSharingService: sessionSharingService,
		SkillPublisher:        skillPublisher,
		SkillDownloader:       skillDownloader,
		SkillLoader:           skillLoader,
		Config:                config,
	}, nil
}
