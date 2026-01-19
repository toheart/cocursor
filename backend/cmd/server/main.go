// @title cocursor Daemon API
// @version 1.0
// @description cocursor 守护进程 API 服务
// @host localhost:19960
// @BasePath /api/v1
// @schemes http
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/cocursor/backend/internal/infrastructure/config"
	applog "github.com/cocursor/backend/internal/infrastructure/log"
	"github.com/cocursor/backend/internal/infrastructure/singleton"
	"github.com/cocursor/backend/internal/wire"
)

func main() {
	// 初始化日志系统
	applog.Init(nil)

	// 加载配置获取端口
	cfg := config.NewConfig()
	port := cfg.Server.HTTPPort

	// 单例锁检查：尝试获取端口锁
	listener, err := singleton.CheckAndLock(port)
	if err != nil {
		log.Fatalf("单例锁检查失败: %v", err)
	}
	if listener == nil {
		// 已有实例运行，直接退出
		log.Println("检测到已有实例在运行，当前进程退出")
		os.Exit(0)
	}
	// 关闭临时 listener，实际监听由 HTTP 服务器负责
	_ = listener.Close()

	// Wire 自动生成的初始化函数
	app, err := wire.InitializeAll()
	if err != nil {
		applog.GetLogger().Error("Failed to initialize application",
			"error", err,
		)
		os.Exit(1)
	}

	// 启动所有服务
	if err := app.Start(); err != nil {
		applog.GetLogger().Error("Failed to start application",
			"error", err,
		)
		os.Exit(1)
	}

	// 优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	applog.GetLogger().Info("Shutting down application...")
	if err := app.Stop(); err != nil {
		applog.GetLogger().Error("Error during application shutdown",
			"error", err,
		)
	}
	applog.GetLogger().Info("Application stopped")
}
