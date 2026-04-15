package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/my-llm-api/config"
	"github.com/my-llm-api/handlers"
	"github.com/my-llm-api/models"
	"github.com/my-llm-api/providers"
	"github.com/my-llm-api/scheduler"
)

func init() {
	config.AppConfig = &config.Config{
		Server: config.ServerConfig{
			Port:     "8080",
			LogLevel: "info",
		},
		Providers: map[string]config.ProviderConfig{
			"siliconflow": {
				BaseURL: "https://api.siliconflow.cn/v1",
				Accounts: []config.AccountConfig{
					{ID: "test", APIKey: "test-key", Weight: 1, Enabled: true},
				},
				Models: []string{"Qwen/Qwen2.5-7B-Instruct"},
			},
		},
	}
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	sched := scheduler.NewScheduler()
	provider := providers.NewSiliconFlowProvider("https://api.siliconflow.cn/v1")
	accountPool := scheduler.NewAccountPool([]config.AccountConfig{
		{ID: "test", APIKey: "test-key", Weight: 1, Enabled: true},
	})
	sched.RegisterProvider(providers.ProviderSiliconFlow, provider, accountPool)

	sched.RegisterModel("Qwen/Qwen2.5-7B-Instruct", &scheduler.ModelConfig{
		ProviderType: providers.ProviderSiliconFlow,
		ModelName:    "Qwen/Qwen2.5-7B-Instruct",
		Priority:     1,
		Weight:       1,
		Enabled:      true,
	})

	router := gin.New()
	chatHandler := handlers.NewChatHandler(sched)

	router.GET("/health", chatHandler.Health)

	v1 := router.Group("/v1")
	{
		v1.POST("/chat/completions", chatHandler.ChatCompletions)
		v1.GET("/models", chatHandler.ListModels)
	}

	return router
}

func TestHealthEndpoint(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("Expected status ok, got %v", response["status"])
	}
}

func TestListModelsEndpoint(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest("GET", "/v1/models", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["object"] != "list" {
		t.Errorf("Expected object list, got %v", response["object"])
	}
}

func TestChatCompletionsValidation(t *testing.T) {
	router := setupTestRouter()

	req := models.ChatCompletionRequest{
		Model: "Qwen/Qwen2.5-7B-Instruct",
	}

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for empty messages, got %d", w.Code)
	}
}
