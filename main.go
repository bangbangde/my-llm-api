package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/my-llm-api/config"
	"github.com/my-llm-api/handlers"
	"github.com/my-llm-api/middleware"
	"github.com/my-llm-api/providers"
	"github.com/my-llm-api/scheduler"
)

func main() {
	if err := config.LoadConfig(""); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	sched := scheduler.NewScheduler()

	for providerName, providerConfig := range config.AppConfig.Providers {
		var providerType providers.ProviderType
		switch providerName {
		case "siliconflow":
			providerType = providers.ProviderSiliconFlow
		default:
			log.Printf("Unknown provider: %s, skipping", providerName)
			continue
		}

		provider := providers.NewSiliconFlowProvider(providerConfig.BaseURL)
		accountPool := scheduler.NewAccountPool(providerConfig.Accounts)

		sched.RegisterProvider(providerType, provider, accountPool)

		for _, modelName := range providerConfig.Models {
			sched.RegisterModel(modelName, &scheduler.ModelConfig{
				ProviderType: providerType,
				ModelName:    modelName,
				Priority:     1,
				Weight:       1,
				Enabled:      true,
			})
		}

		log.Printf("Registered provider: %s with %d accounts, %d models",
			providerName, accountPool.HealthyCount(), len(providerConfig.Models))
	}

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
	router.Use(middleware.CORS())

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
