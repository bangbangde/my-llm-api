package providers

import (
	"context"

	"github.com/my-llm-api/models"
)

type ProviderType string

const (
	ProviderSiliconFlow ProviderType = "siliconflow"
)

type Provider interface {
	Name() string
	ChatCompletion(ctx context.Context, req *models.ChatCompletionRequest, apiKey string) (*models.ChatCompletionResponse, error)
	ChatCompletionStream(ctx context.Context, req *models.ChatCompletionRequest, apiKey string) (<-chan *models.ChatCompletionStreamResponse, error)
}

type ProviderFactory struct {
	providers map[ProviderType]Provider
}

func NewProviderFactory() *ProviderFactory {
	return &ProviderFactory{
		providers: make(map[ProviderType]Provider),
	}
}

func (f *ProviderFactory) Register(providerType ProviderType, provider Provider) {
	f.providers[providerType] = provider
}

func (f *ProviderFactory) Get(providerType ProviderType) Provider {
	return f.providers[providerType]
}

func (f *ProviderFactory) GetDefault() Provider {
	for _, provider := range f.providers {
		return provider
	}
	return nil
}
