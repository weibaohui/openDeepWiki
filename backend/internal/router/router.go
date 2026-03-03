package router

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/embed"
	"github.com/weibaohui/opendeepwiki/backend/internal/handler"
)

func Setup(
	cfg *config.Config,
	repoHandler *handler.RepositoryHandler,
	taskHandler *handler.TaskHandler,
	docHandler *handler.DocumentHandler,
	apiKeyHandler *handler.APIKeyHandler,
	syncHandler *handler.SyncHandler,
	userRequestHandler *handler.UserRequestHandler,
	openAPIHandler *handler.OpenAPIHandler,
	activityHandler *handler.ActivityHandler,
	agentHandler *handler.AgentHandler,
	embeddingKeyHandler *handler.EmbeddingKeyHandler,
	vectorHandler *handler.VectorHandler,
	chatHandler *handler.ChatHandler,
) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	api := r.Group("/api")
	{
		api.GET("/doc/:id/redirect", docHandler.Redirect)

		repos := api.Group("/repositories")
		{
			repos.POST("", repoHandler.Create)
			repos.GET("", repoHandler.List)
			repos.GET("/:id", repoHandler.Get)
			repos.DELETE("/:id", repoHandler.Delete)
			repos.POST("/:id/run-all", repoHandler.RunAllTasks)
			repos.POST("/:id/clone", repoHandler.Clone)
			repos.POST("/:id/purge-local", repoHandler.PurgeLocal)
			repos.POST("/:id/directory-analyze", repoHandler.AnalyzeDirectory)
			repos.POST("/:id/db-model-analyze", repoHandler.AnalyzeDatabaseModel)
			repos.POST("/:id/api-analyze", repoHandler.AnalyzeAPI)
			repos.POST("/:id/incremental-analysis", repoHandler.IncrementalAnalysis)
			repos.POST("/:id/user-requests", userRequestHandler.CreateUserRequest)
			repos.GET("/:id/user-requests", userRequestHandler.ListUserRequests)
			repos.POST("/:id/set-ready", repoHandler.SetReady)
			repos.GET("/:id/incremental-history", repoHandler.GetIncrementalHistory)
			repos.GET("/:id/tasks", taskHandler.GetByRepository)
			repos.GET("/:id/tasks/stats", taskHandler.GetStats) // 新增：任务统计
			repos.GET("/:id/documents", docHandler.GetByRepository)
			repos.GET("/:id/documents/index", docHandler.GetIndex)
			repos.GET("/:id/documents/export", docHandler.Export)
			repos.GET("/:id/export-pdf", docHandler.ExportPDF)
		}

		tasks := api.Group("/tasks")
		{
			tasks.GET("/status", taskHandler.GetOrchestratorStatus) // 获取编排器状态（新增）
			tasks.GET("/monitor", taskHandler.Monitor)              // 获取全局监控数据（新增）
			tasks.GET("/stuck", taskHandler.GetStuck)               // 获取卡住的任务
			tasks.POST("/cleanup", taskHandler.CleanupStuck)        // 清理卡住的任务
			tasks.GET("/:id", taskHandler.Get)
			tasks.POST("/:id/run", taskHandler.Run)
			tasks.POST("/:id/enqueue", taskHandler.Enqueue)      // 新增：提交任务到队列
			tasks.POST("/:id/retry", taskHandler.Retry)          // 新增：重试任务
			tasks.POST("/:id/regen", taskHandler.ReGenByNewTask) // 新增：重新生成任务
			tasks.POST("/:id/cancel", taskHandler.Cancel)        // 新增：取消任务
			tasks.POST("/:id/reset", taskHandler.Reset)
			tasks.POST("/:id/force-reset", taskHandler.ForceReset) // 强制重置
			tasks.DELETE("/:id", taskHandler.Delete)               // 删除任务（新增）
		}

		docs := api.Group("/documents")
		{
			docs.GET("/:id", docHandler.Get)
			docs.GET("/:id/versions", docHandler.GetVersions)
			docs.PUT("/:id", docHandler.Update)
			docs.POST("/:id/ratings", docHandler.SubmitRating)
			docs.GET("/:id/ratings/stats", docHandler.GetRatingStats)
			docs.GET("/:id/token-usage", docHandler.GetTokenUsage)
		}

		// API Key 管理
		apiKeyHandler.RegisterRoutes(api)

		// 数据同步
		syncHandler.RegisterRoutes(api)

		// 活跃度配置
		activityHandler.RegisterRoutes(api)

		// Agent 管理
		agentHandler.RegisterRoutes(api)

		// Embedding Key 管理
		embeddingKeyHandler.RegisterRoutes(api)

		// 向量管理
		if vectorHandler != nil {
			vectorHandler.RegisterRoutes(api)
		}

		// 对话管理
		if chatHandler != nil {
			chatHandler.RegisterRoutes(api)
		}

		// 用户需求管理
		userRequests := api.Group("/user-requests")
		{
			userRequests.GET("/:id", userRequestHandler.GetUserRequest)
			userRequests.DELETE("/:id", userRequestHandler.DeleteUserRequest)
			userRequests.PATCH("/:id/status", userRequestHandler.UpdateUserRequestStatus)
		}
	}

	// OpenAPI 文档端点（AI 友好 API 端点）
	// 符合 RFC 8615 规范的 .well-known URI
	// 提供 OpenAPI 3.0 规范文档，供 AI 工具使用
	if openAPIHandler != nil {
		r.GET("/.well-known/openapi.yaml", openAPIHandler.ServeOpenAPI)
	}

	// 设置前端静态文件路由（嵌入式）
	// 必须在API路由之后设置，确保API请求优先匹配
	embed.SetupRouter(r)

	return r
}
