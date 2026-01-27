package main

import (
	"flag"
	"log"
	"os"

	"github.com/opendeepwiki/backend/config"
	"github.com/opendeepwiki/backend/models"
	"github.com/opendeepwiki/backend/router"
	"k8s.io/klog/v2"
)

func main() {
	// 初始化 klog
	klog.InitFlags(nil)
	// 默认设置日志级别为 6，确保 V(6) 的日志能打印出来
	flag.Set("v", "6")
	flag.Set("logtostderr", "true")
	flag.Parse()
	defer klog.Flush()

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

	r := router.Setup(cfg)

	log.Printf("Server starting on port %s...", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
