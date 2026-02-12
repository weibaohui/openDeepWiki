package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"k8s.io/klog/v2"

	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/domain/writers"
	"github.com/weibaohui/opendeepwiki/backend/internal/eventbus"
	"github.com/weibaohui/opendeepwiki/backend/internal/handler"
	"github.com/weibaohui/opendeepwiki/backend/internal/pkg/adkagents"
	"github.com/weibaohui/opendeepwiki/backend/internal/pkg/database"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"github.com/weibaohui/opendeepwiki/backend/internal/router"
	"github.com/weibaohui/opendeepwiki/backend/internal/service"
	"github.com/weibaohui/opendeepwiki/backend/internal/service/orchestrator"
	syncservice "github.com/weibaohui/opendeepwiki/backend/internal/service/sync"
	"github.com/weibaohui/opendeepwiki/backend/internal/subscriber"
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
	db, err := database.InitDB(cfg)

	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 初始化 Repository
	repoRepo := repository.NewRepoRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	docRepo := repository.NewDocumentRepository(db)
	ratingRepo := repository.NewDocumentRatingRepository(db)
	apiKeyRepo := repository.NewAPIKeyRepository(db)
	hintRepo := repository.NewHintRepository(db)
	taskUsageRepo := repository.NewTaskUsageRepository(db)

	// 初始化 Service
	docService := service.NewDocumentService(cfg, docRepo, repoRepo, ratingRepo)
	apiKeyService := service.NewAPIKeyService(apiKeyRepo)
	taskUsageService := service.NewTaskUsageService(taskUsageRepo)

	//初始化系列Writer
	titleRewriter, err := writers.NewTitleRewriter(cfg, docRepo, taskRepo)
	if err != nil {
		log.Fatalf("Failed to initialize title rewriter service: %v", err)
	}

	userRequestWriter, err := writers.NewUserRequestWriter(cfg, hintRepo)
	if err != nil {
		log.Fatalf("Failed to initialize user request writer service: %v", err)
	}
	defaultWriter, err := writers.NewDefaultWriter(cfg, hintRepo)
	if err != nil {
		log.Fatalf("Failed to initialize document generator service: %v", err)
	}
	dbModelWriter, err := writers.NewDBModelWriter(cfg, hintRepo, taskRepo)
	if err != nil {
		log.Fatalf("Failed to initialize db model writer service: %v", err)
	}
	apiWriter, err := writers.NewAPIWriter(cfg, hintRepo, taskRepo)
	if err != nil {
		log.Fatalf("Failed to initialize api analyzer service: %v", err)
	}

	// 初始化目录分析服务
	tocWriter, err := writers.NewTocWriter(cfg, docRepo, repoRepo, taskRepo, hintRepo)
	if err != nil {
		log.Fatalf("Failed to initialize directory analyzer service: %v", err)
	}
	//初始化系列Writer结束

	taskService := service.NewTaskService(cfg, taskRepo, repoRepo, docService)
	taskService.AddWriters(userRequestWriter)
	taskService.AddWriters(defaultWriter)
	taskService.AddWriters(dbModelWriter)
	taskService.AddWriters(apiWriter)
	taskService.AddWriters(titleRewriter)
	taskService.AddWriters(tocWriter)
	tocWriter.SetTaskService(taskService)

	// 初始化全局任务编排器
	// maxWorkers=2，避免并发过多打爆CPU/LLM配额
	taskExecutor := &taskExecutorAdapter{taskService: taskService}
	orchestrator.InitGlobalOrchestrator(1, taskExecutor)
	taskService.SetOrchestrator(orchestrator.GetGlobalOrchestrator())
	defer orchestrator.ShutdownGlobalOrchestrator()

	//注册TaskEventBus
	taskEventBus := eventbus.NewTaskEventBus()
	subscriber.NewTaskEventSubscriber(taskService).Register(taskEventBus)

	// 初始化 RepositoryService (依赖全局编排器，需要在 orchestrator 初始化之后)
	repoService := service.NewRepositoryService(cfg, repoRepo, taskRepo, docRepo, hintRepo)
	//注册RepoEventBus
	repoEventBus := eventbus.NewRepositoryEventBus()
	subscriber.NewRepositoryEventSubscriber(taskService, repoService).Register(repoEventBus)

	// 初始化 Handler
	repoHandler := handler.NewRepositoryHandler(repoEventBus, taskEventBus, repoService, taskService)
	taskHandler := handler.NewTaskHandler(taskService)
	docHandler := handler.NewDocumentHandler(docService)
	apiKeyHandler := handler.NewAPIKeyHandler(apiKeyService)
	syncService := syncservice.New(repoRepo, taskRepo, docRepo, taskUsageRepo)
	syncHandler := handler.NewSyncHandler(syncService)

	// 初始化 EnhancedModelProvider 并设置到 Manager
	manager, err := adkagents.GetOrCreateInstance(cfg)
	if err != nil {
		log.Fatalf("Failed to get manager: %v", err)
	}
	enhancedModelProvider, err := adkagents.NewEnhancedModelProvider(cfg, apiKeyRepo, apiKeyService, taskUsageService)
	if err != nil {
		log.Fatalf("Failed to create enhanced model provider: %v", err)
	}
	manager.SetEnhancedModelProvider(enhancedModelProvider)

	// 启动时清理卡住的任务（超过 10 分钟的运行中任务）
	cleanupStuckTasks(taskService)
	taskService.StartPendingTaskScheduler(context.Background(), 10*time.Second)

	// 设置路由
	r := router.Setup(cfg, repoHandler, taskHandler, docHandler, apiKeyHandler, syncHandler)

	//eino callbacks注册
	callbacks := adkagents.NewEinoCallbacks(true, 8)
	callbacks.AppendGlobalHandlers(callbacks.Handler())

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

	queuedAffected, err := taskService.CleanupQueuedTasksOnStartup()
	if err != nil {
		klog.V(6).Infof("清理启动遗留排队任务失败: %v", err)
		return
	}

	if queuedAffected > 0 {
		klog.V(6).Infof("启动时清理了 %d 个遗留排队任务", queuedAffected)
	}
}
