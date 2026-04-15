package scheduler

import (
	"context"
	"sync"

	"github.com/my-llm-api/models"
	"github.com/my-llm-api/providers"
)

type ModelConfig struct {
	ProviderType providers.ProviderType
	ModelName    string
	Priority     int
	Weight       int
	Enabled      bool
}

type Scheduler struct {
	providers    map[providers.ProviderType]providers.Provider
	accountPools map[providers.ProviderType]*AccountPool
	modelConfigs map[string][]*ModelConfig
	mu           sync.RWMutex
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		providers:    make(map[providers.ProviderType]providers.Provider),
		accountPools: make(map[providers.ProviderType]*AccountPool),
		modelConfigs: make(map[string][]*ModelConfig),
	}
}

func (s *Scheduler) RegisterProvider(providerType providers.ProviderType, provider providers.Provider, accountPool *AccountPool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.providers[providerType] = provider
	s.accountPools[providerType] = accountPool
}

func (s *Scheduler) RegisterModel(modelName string, config *ModelConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.modelConfigs[modelName] = append(s.modelConfigs[modelName], config)
}

func (s *Scheduler) GetProvider(providerType providers.ProviderType) providers.Provider {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.providers[providerType]
}

func (s *Scheduler) GetAccountPool(providerType providers.ProviderType) *AccountPool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.accountPools[providerType]
}

func (s *Scheduler) SelectProviderAndAccount(modelName string) (providers.Provider, *Account) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	configs, exists := s.modelConfigs[modelName]
	if !exists || len(configs) == 0 {
		for providerType, provider := range s.providers {
			if pool, ok := s.accountPools[providerType]; ok {
				if account := pool.Select(); account != nil {
					return provider, account
				}
			}
		}
		return nil, nil
	}

	enabledConfigs := make([]*ModelConfig, 0)
	for _, config := range configs {
		if config.Enabled {
			enabledConfigs = append(enabledConfigs, config)
		}
	}

	if len(enabledConfigs) == 0 {
		for providerType, provider := range s.providers {
			if pool, ok := s.accountPools[providerType]; ok {
				if account := pool.Select(); account != nil {
					return provider, account
				}
			}
		}
		return nil, nil
	}

	for _, config := range enabledConfigs {
		provider := s.providers[config.ProviderType]
		pool := s.accountPools[config.ProviderType]
		if provider != nil && pool != nil {
			if account := pool.Select(); account != nil {
				return provider, account
			}
		}
	}

	return nil, nil
}

func (s *Scheduler) markAccountResult(modelName, accountID string, success bool) {
	s.mu.RLock()
	configs, exists := s.modelConfigs[modelName]
	if !exists || len(configs) == 0 {
		for providerType := range s.providers {
			if pool := s.GetAccountPool(providerType); pool != nil {
				if success {
					pool.MarkSuccess(accountID)
				} else {
					pool.MarkFailed(accountID)
				}
			}
		}
		s.mu.RUnlock()
		return
	}
	s.mu.RUnlock()

	for _, config := range configs {
		if pool := s.GetAccountPool(config.ProviderType); pool != nil {
			if success {
				pool.MarkSuccess(accountID)
			} else {
				pool.MarkFailed(accountID)
			}
		}
	}
}

func (s *Scheduler) ChatCompletion(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
	provider, account := s.SelectProviderAndAccount(req.Model)
	if provider == nil || account == nil {
		return nil, &models.ErrorDetail{
			Message: "no available provider or account for model: " + req.Model,
			Type:    "invalid_request_error",
			Code:    "provider_not_found",
		}
	}

	resp, err := provider.ChatCompletion(ctx, req, account.APIKey)
	if err != nil {
		s.markAccountResult(req.Model, account.ID, false)
		return nil, err
	}

	s.markAccountResult(req.Model, account.ID, true)
	return resp, nil
}

func (s *Scheduler) ChatCompletionStream(ctx context.Context, req *models.ChatCompletionRequest) (<-chan *models.ChatCompletionStreamResponse, error) {
	provider, account := s.SelectProviderAndAccount(req.Model)
	if provider == nil || account == nil {
		return nil, &models.ErrorDetail{
			Message: "no available provider or account for model: " + req.Model,
			Type:    "invalid_request_error",
			Code:    "provider_not_found",
		}
	}

	streamChan, err := provider.ChatCompletionStream(ctx, req, account.APIKey)
	if err != nil {
		s.markAccountResult(req.Model, account.ID, false)
		return nil, err
	}

	return streamChan, nil
}
