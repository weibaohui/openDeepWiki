package main

import (
	"flag"
	"log"
	"os"
	"time"

	"k8s.io/klog/v2"

	"github.com/opendeepwiki/backend/config"
	"github.com/opendeepwiki/backend/models"
	"github.com/opendeepwiki/backend/router"
	"github.com/opendeepwiki/backend/services"
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

	if err := models.InitDB(cfg.Database.Type, cfg.Database.DSN); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 启动时清理卡住的任务（超过 10 分钟的运行中任务）
	cleanupStuckTasks(cfg)

	r := router.Setup(cfg)

	log.Printf("Server starting on port %s...", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// cleanupStuckTasks 清理启动前卡住的任务
func cleanupStuckTasks(cfg *config.Config) {
	taskService := services.NewTaskService(cfg)
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
