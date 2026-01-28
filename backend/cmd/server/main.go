package main

import (
	"flag"
	"log"
	"os"
	"time"

	"k8s.io/klog/v2"

	"github.com/opendeepwiki/backend/config"
	"github.com/opendeepwiki/backend/internal/handler"
	"github.com/opendeepwiki/backend/internal/pkg/database"
	"github.com/opendeepwiki/backend/internal/repository"
	"github.com/opendeepwiki/backend/internal/router"
	"github.com/opendeepwiki/backend/internal/service"
	"github.com/opendeepwiki/backend/internal/service/orchestrator"
)

func main() {
	// 初始化 klog
	klog.InitFlags(nil)
	flag.Parse()
	defer klog.Flush()

	klog.V(6).Info("服务启动中...")

	cfg := config.GetConfig()

	if err := os.MkdirAll(cfg.Data.Dir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}
	if err := os.MkdirAll(cfg.Data.RepoDir, 0755); err != nil {
		log.Fatalf("Failed to create repo directory: %v", err)
	}

	// 初始化数据库
	db, err := database.InitDB(cfg.Database.Type, cfg.Database.DSN)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 初始化 Repository
	repoRepo := repository.NewRepoRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	docRepo := repository.NewDocumentRepository(db)

	// 初始化 Service
	docService := service.NewDocumentService(cfg, docRepo, repoRepo)
	taskService := service.NewTaskService(cfg, taskRepo, repoRepo, docService)
	repoService := service.NewRepositoryService(cfg, repoRepo, taskRepo, docRepo, taskService)

	// 初始化全局任务编排器
	// maxWorkers=2，避免并发过多打爆CPU/LLM配额
	taskExecutor := &taskExecutorAdapter{taskService: taskService}
	orchestrator.InitGlobalOrchestrator(2, taskExecutor)
	taskService.SetOrchestrator(orchestrator.GetGlobalOrchestrator())
	defer orchestrator.ShutdownGlobalOrchestrator()

	// 初始化 Handler
	repoHandler := handler.NewRepositoryHandler(repoService)
	taskHandler := handler.NewTaskHandler(taskService)
	docHandler := handler.NewDocumentHandler(docService)
	configHandler := handler.NewConfigHandler(cfg)

	// 启动时清理卡住的任务（超过 10 分钟的运行中任务）
	cleanupStuckTasks(taskService)

	// 设置路由
	r := router.Setup(cfg, repoHandler, taskHandler, docHandler, configHandler)

	log.Printf("Server starting on port %s...", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// cleanupStuckTasks 清理启动前卡住的任务
func cleanupStuckTasks(taskService *service.TaskService) {
	timeout := 10 * time.Minute

	affected, err := taskService.CleanupStuckTasks(timeout)
	if err != nil {
		klog.V(6).Infof("清理卡住任务失败: %v", err)
		return
	}

	if affected > 0 {
		klog.V(6).Infof("启动时清理了 %d 个卡住的任务", affected)
	}
}
