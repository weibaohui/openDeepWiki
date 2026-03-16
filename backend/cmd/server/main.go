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
	"github.com/weibaohui/opendeepwiki/backend/internal/mcp"
	"github.com/weibaohui/opendeepwiki/backend/internal/pkg/adkagents"
	"github.com/mark3labs/mcp-go/server"
	"github.com/gin-gonic/gin"
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
	syncTargetRepo := repository.NewSyncTargetRepository(db)
	syncEventRepo := repository.NewSyncEventRepository(db)
	incrementalHistoryRepo := repository.NewIncrementalUpdateHistoryRepository(db)
	userRequestRepo := repository.NewUserRequestRepository(db)
	agentVersionRepo := repository.NewAgentVersionRepository(db)
	chatSessionRepo := repository.NewChatSessionRepository(db)
	chatMessageRepo := repository.NewChatMessageRepository(db)
	chatToolCallRepo := repository.NewChatToolCallRepository(db)

	// 初始化 Service
	docService := service.NewDocumentService(cfg, docRepo, repoRepo, ratingRepo, nil)
	apiKeyService := service.NewAPIKeyService(apiKeyRepo)
	taskUsageService := service.NewTaskUsageService(taskUsageRepo)
	userRequestService := service.NewUserRequestService(userRequestRepo, repoRepo)
	agentService := service.NewAgentService(agentVersionRepo, cfg.Agent.Dir)
	chatService := service.NewChatService(chatSessionRepo, chatMessageRepo, chatToolCallRepo)

	//初始化系列Writer
	titleRewriter, err := writers.NewTitleRewriter(cfg, docRepo, taskRepo)
	if err != nil {
		log.Fatalf("Failed to initialize title rewriter service: %v", err)
	}
	docRewriter, err := writers.NewDocRewriter(cfg, docRepo, taskRepo)
	if err != nil {
		log.Fatalf("Failed to initialize doc rewriter service: %v", err)
	}

	userRequestWriter, err := writers.NewUserRequestWriter(cfg, hintRepo)
	if err != nil {
		log.Fatalf("Failed to initialize user request writer service: %v", err)
	}
	defaultWriter, err := writers.NewDefaultWriter(cfg, hintRepo, taskRepo)
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

	incrementalWriter, err := writers.NewIncrementalWriter(cfg, repoRepo, taskRepo, hintRepo, docRepo, incrementalHistoryRepo)
	if err != nil {
		log.Fatalf("Failed to initialize incremental writer service: %v", err)
	}
	//初始化系列Writer结束

	taskService := service.NewTaskService(cfg, taskRepo, repoRepo, docService)
	taskService.AddWriters(userRequestWriter)
	taskService.AddWriters(defaultWriter)
	taskService.AddWriters(dbModelWriter)
	taskService.AddWriters(apiWriter)
	taskService.AddWriters(titleRewriter)
	taskService.AddWriters(docRewriter)
	taskService.AddWriters(tocWriter)
	taskService.AddWriters(incrementalWriter)
	tocWriter.SetTaskService(taskService)
	incrementalWriter.SetTaskService(taskService)

	// 初始化全局任务编排器
	// maxWorkers=2，避免并发过多打爆CPU/LLM配额
	taskExecutor := &taskExecutorAdapter{taskService: taskService}
	orchestrator.InitGlobalOrchestrator(1, taskExecutor)
	taskService.SetOrchestrator(orchestrator.GetGlobalOrchestrator())
	defer orchestrator.ShutdownGlobalOrchestrator()

	// 初始化任务事件总线
	taskEventBus := eventbus.NewTaskEventBus()
	subscriber.NewTaskEventSubscriber(taskService).Register(taskEventBus)
	taskService.SetEventBus(taskEventBus)

	// 初始化活跃度事件总线
	activityEventBus := eventbus.NewActivityEventBus()
	subscriber.NewActivityEventSubscriber(repoRepo, cfg).Register(activityEventBus)

	// 初始化活跃度调度器
	activityScheduler := service.NewActivityScheduler(cfg, repoRepo, taskEventBus)
	activityScheduler.Start(context.Background())
	defer activityScheduler.Stop()

	// 初始化 ActivityHandler
	activityHandler := handler.NewActivityHandler(cfg)

	// 初始化 RepositoryService (依赖全局编排器，需要在 orchestrator 初始化之后)
	repoService := service.NewRepositoryService(cfg, repoRepo, taskRepo, docRepo, hintRepo, incrementalHistoryRepo)
	//注册RepoEventBus
	repoEventBus := eventbus.NewRepositoryEventBus()
	subscriber.NewRepositoryEventSubscriber(taskEventBus, taskService, repoService).Register(repoEventBus)
	incrementalWriter.SetRepositoryEventBus(repoEventBus)

	// 初始化文档事件总线
	docEventBus := eventbus.NewDocEventBus()
	subscriber.NewDocEventSubscriber(taskEventBus, syncEventRepo).Register(docEventBus)

	// 初始化 Handler
	repoHandler := handler.NewRepositoryHandler(repoEventBus, taskEventBus, repoService, taskService)
	taskHandler := handler.NewTaskHandler(taskService)
	docHandler := handler.NewDocumentHandler(docEventBus, docService)
	apiKeyHandler := handler.NewAPIKeyHandler(apiKeyService)
	syncService := syncservice.New(repoRepo, taskRepo, docRepo, taskUsageRepo, syncTargetRepo, syncEventRepo)
	syncService.SetDocEventBus(docEventBus)
	syncHandler := handler.NewSyncHandler(syncService)
	userRequestHandler := handler.NewUserRequestHandler(userRequestService, taskEventBus, taskService)

	agentHandler := handler.NewAgentHandler(agentService)

	// 初始化 OpenAPIHandler（AI 友好 API 端点）
	// 提供 /.well-known/openapi.yaml 端点，供 AI 工具使用
	openAPIHandler := handler.NewOpenAPIHandler(".well-known/openapi.yaml")

	// 初始化 EnhancedModelProvider 并设置到 Manager
	manager, err := adkagents.GetOrCreateInstanceWithDocRepo(cfg, docRepo)
	if err != nil {
		log.Fatalf("Failed to get manager: %v", err)
	}
	enhancedModelProvider, err := adkagents.NewEnhancedModelProvider(cfg, apiKeyRepo, apiKeyService, taskUsageService)
	if err != nil {
		log.Fatalf("Failed to create enhanced model provider: %v", err)
	}
	manager.SetEnhancedModelProvider(enhancedModelProvider)

	// 创建 AgentFactory（必须在 Manager 设置 EnhancedModelProvider 之后）
	agentFactory, err := adkagents.NewAgentFactory(cfg)
	if err != nil {
		log.Fatalf("Failed to create agent factory: %v", err)
	}

	// 创建 ChatHandler，传入 AgentFactory、RepositoryService 和 DocumentService
	chatHandler := handler.NewChatHandler(chatService, repoService, docService, agentFactory)
	// 启动ChatHub
	go chatHandler.GetHub().Run()

	// 创建 MCP Server，为 AI 编程工具提供文档查询接口
	mcpServer := mcp.NewMCPServer(repoService, docService)
	klog.V(6).Info("MCP Server 已初始化")

	// 启动时清理卡住的任务（超过 10 分钟的运行中任务）
	cleanupStuckTasks(taskService)
	taskService.StartPendingTaskScheduler(context.Background(), 10*time.Second)

	// 设置路由
	r := router.Setup(cfg, repoHandler, taskHandler, docHandler, apiKeyHandler, syncHandler, userRequestHandler, openAPIHandler, activityHandler, agentHandler, chatHandler)

	// 添加 MCP SSE 端点
	// 提供 /mcp/sse 端点，供 Cursor、Claude Code 等 AI 编程工具使用
	r.GET("/mcp/sse", func(c *gin.Context) {
		// 使用 mcp-go 的 SSE handler
		handler := server.NewSSEServer(mcpServer.GetServer())
		handler.ServeHTTP(c.Writer, c.Request)
	})
	klog.V(6).Info("MCP SSE 端点已注册: /mcp/sse")

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