package providers

import (
	"context"
	"net/http"
	"time"

	"github.com/my-llm-api/config"
	"github.com/my-llm-api/models"
)

type ProviderType string

const (
	ProviderSiliconFlow ProviderType = "siliconflow"
)

// HTTPDoer 接口，允许注入自定义 HTTP Client（用于测试和自定义配置）
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// DefaultHTTPClient 共享的 HTTP Client，所有 Provider 实例复用连接池
var DefaultHTTPClient = &http.Client{
	Timeout: 120 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost:  10,
		IdleConnTimeout:      90 * time.Second,
		DisableKeepAlives:    false,
		MaxConnsPerHost:      0, // 不限制每个 host 的连接数
	},
}

// Provider is the interface that all backend LLM providers must implement.
type Provider interface {
	Name() string
	ChatCompletion(ctx context.Context, req *models.ChatCompletionRequest, apiKey string) (*models.ChatCompletionResponse, error)
	ChatCompletionStream(ctx context.Context, req *models.ChatCompletionRequest, apiKey string) (<-chan *models.ChatCompletionStreamResponse, error)
}

// ProviderBuilder Provider 构建器类型
type ProviderBuilder func(cfg config.ProviderConfig) Provider

// ProviderRegistry Provider 注册表
type ProviderRegistry struct {
	builders map[string]ProviderBuilder
}

// NewProviderRegistry 创建新的 Provider 注册表
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		builders: make(map[string]ProviderBuilder),
	}
}

// Register 注册 Provider 构建器
func (r *ProviderRegistry) Register(name string, builder ProviderBuilder) {
	r.builders[name] = builder
}

// Build 构建 Provider 实例
func (r *ProviderRegistry) Build(name string, cfg config.ProviderConfig) Provider {
	builder, ok := r.builders[name]
	if !ok {
		return nil
	}

	return builder(cfg)
}

// GetGlobalRegistry 获取全局注册表
var globalRegistry *ProviderRegistry

func init() {
	globalRegistry = NewProviderRegistry()

	// 注册内置 Provider
	globalRegistry.Register("siliconflow", func(cfg config.ProviderConfig) Provider {
		return NewSiliconFlowProvider(cfg.BaseURL)
	})
}

func GetGlobalRegistry() *ProviderRegistry {
	return globalRegistry
}
