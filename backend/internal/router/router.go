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
			repos.POST("/:id/set-ready", repoHandler.SetReady)
			repos.GET("/:id/tasks", taskHandler.GetByRepository)
			repos.GET("/:id/tasks/stats", taskHandler.GetStats) // 新增：任务统计
			repos.GET("/:id/documents", docHandler.GetByRepository)
			repos.GET("/:id/documents/index", docHandler.GetIndex)
			repos.GET("/:id/documents/export", docHandler.Export)
		}

		tasks := api.Group("/tasks")
		{
			tasks.GET("/status", taskHandler.GetOrchestratorStatus) // 获取编排器状态（新增）
			tasks.GET("/monitor", taskHandler.Monitor)              // 获取全局监控数据（新增）
			tasks.GET("/stuck", taskHandler.GetStuck)               // 获取卡住的任务
			tasks.POST("/cleanup", taskHandler.CleanupStuck)        // 清理卡住的任务
			tasks.GET("/:id", taskHandler.Get)
			tasks.POST("/:id/run", taskHandler.Run)
			tasks.POST("/:id/enqueue", taskHandler.Enqueue) // 新增：提交任务到队列
			tasks.POST("/:id/retry", taskHandler.Retry)     // 新增：重试任务
			tasks.POST("/:id/cancel", taskHandler.Cancel)   // 新增：取消任务
			tasks.POST("/:id/reset", taskHandler.Reset)
			tasks.POST("/:id/force-reset", taskHandler.ForceReset) // 强制重置
			tasks.DELETE("/:id", taskHandler.Delete)               // 删除任务（新增）
		}

		docs := api.Group("/documents")
		{
			docs.GET("/:id", docHandler.Get)
			docs.GET("/:id/versions", docHandler.GetVersions)
			docs.PUT("/:id", docHandler.Update)
		}

		// API Key 管理
		apiKeyHandler.RegisterRoutes(api)
	}

	// 设置前端静态文件路由（嵌入式）
	// 必须在API路由之后设置，确保API请求优先匹配
	embed.SetupRouter(r)

	return r
}
