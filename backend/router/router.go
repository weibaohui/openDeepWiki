package router

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/opendeepwiki/backend/config"
	"github.com/opendeepwiki/backend/handlers"
)

func Setup(cfg *config.Config) *gin.Engine {
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

	repoHandler := handlers.NewRepositoryHandler(cfg)
	taskHandler := handlers.NewTaskHandler(cfg)
	docHandler := handlers.NewDocumentHandler(cfg)
	configHandler := handlers.NewConfigHandler(cfg)

	api := r.Group("/api")
	{
		repos := api.Group("/repositories")
		{
			repos.POST("", repoHandler.Create)
			repos.GET("", repoHandler.List)
			repos.GET("/:id", repoHandler.Get)
			repos.DELETE("/:id", repoHandler.Delete)
			repos.POST("/:id/run-all", repoHandler.RunAllTasks)
			repos.GET("/:id/tasks", taskHandler.GetByRepository)
			repos.GET("/:id/documents", docHandler.GetByRepository)
			repos.GET("/:id/documents/index", docHandler.GetIndex)
			repos.GET("/:id/documents/export", docHandler.Export)
		}

		tasks := api.Group("/tasks")
		{
			tasks.GET("/:id", taskHandler.Get)
			tasks.POST("/:id/run", taskHandler.Run)
			tasks.POST("/:id/reset", taskHandler.Reset)
		}

		docs := api.Group("/documents")
		{
			docs.GET("/:id", docHandler.Get)
			docs.PUT("/:id", docHandler.Update)
		}

		api.GET("/config", configHandler.Get)
		api.PUT("/config", configHandler.Update)
	}

	return r
}
