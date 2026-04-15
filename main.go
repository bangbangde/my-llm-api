package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/my-llm-api/config"
	"github.com/my-llm-api/handlers"
	"github.com/my-llm-api/middleware"
	"github.com/my-llm-api/scheduler"
)

func main() {
	if err := config.LoadConfig(""); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 使用工厂模式创建调度器
	factory := scheduler.DefaultFactory()
	sched, err := factory.BuildScheduler(config.AppConfig)
	if err != nil {
		log.Fatalf("Failed to build scheduler: %v", err)
	}

	sched.LogStats()

	router := setupRouter(sched)

	log.Printf("Starting server on port %s", config.AppConfig.Server.Port)
	if err := router.Run(":" + config.AppConfig.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func setupRouter(sched *scheduler.Scheduler) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())
	router.Use(middleware.CORS(config.AppConfig.Server.AllowedOrigins))
	router.Use(middleware.Auth(config.AppConfig.Server.APIKeys))

	chatHandler := handlers.NewChatHandler(sched)

	router.GET("/health", chatHandler.Health)

	v1 := router.Group("/v1")
	{
		v1.POST("/chat/completions", chatHandler.ChatCompletions)
		v1.GET("/models", chatHandler.ListModels)
	}

	router.POST("/chat/completions", chatHandler.ChatCompletions)
	router.GET("/models", chatHandler.ListModels)

	return router
}
