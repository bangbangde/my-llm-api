package scheduler

import (
	"context"
	"log"
	"sync"

	"github.com/my-llm-api/models"
	"github.com/my-llm-api/providers"
)

// SchedulerDeps 依赖注入结构，便于测试和扩展
type SchedulerDeps struct {
	Providers    map[providers.ProviderType]providers.Provider
	AccountPools map[providers.ProviderType]*AccountPool
}

type ModelConfig struct {
	ProviderType providers.ProviderType
	ModelName    string
	Priority     int
	Weight       int
	Enabled      bool
}

type Scheduler struct {
	deps SchedulerDeps
	// modelConfigs 需要读写锁保护
	modelConfigs map[string][]*ModelConfig
	mu           sync.RWMutex
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		deps: SchedulerDeps{
			Providers:    make(map[providers.ProviderType]providers.Provider),
			AccountPools: make(map[providers.ProviderType]*AccountPool),
		},
		modelConfigs: make(map[string][]*ModelConfig),
	}
}

// NewSchedulerWithDeps 使用依赖创建调度器（用于测试）
func NewSchedulerWithDeps(deps SchedulerDeps) *Scheduler {
	return &Scheduler{
		deps:         deps,
		modelConfigs: make(map[string][]*ModelConfig),
	}
}

func (s *Scheduler) RegisterProvider(providerType providers.ProviderType, provider providers.Provider, accountPool *AccountPool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deps.Providers[providerType] = provider
	s.deps.AccountPools[providerType] = accountPool
}

func (s *Scheduler) RegisterModel(modelName string, config *ModelConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.modelConfigs[modelName] = append(s.modelConfigs[modelName], config)
}

func (s *Scheduler) GetProvider(providerType providers.ProviderType) providers.Provider {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.deps.Providers[providerType]
}

func (s *Scheduler) GetAccountPool(providerType providers.ProviderType) *AccountPool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.deps.AccountPools[providerType]
}

// SelectProviderAndAccount 优化后的选择逻辑，减少锁持有时间
// 1. 短暂加锁获取配置
// 2. 在锁外选择账号（AccountPool 内部有独立锁）
func (s *Scheduler) SelectProviderAndAccount(modelName string) (providers.Provider, *Account) {
	// 1. 短暂加锁获取配置
	s.mu.RLock()
	configs, exists := s.modelConfigs[modelName]

	if !exists || len(configs) == 0 {
		s.mu.RUnlock()
		return s.selectFromAllProviders()
	}

	// 筛选启用的配置
	var targetConfig *ModelConfig
	for _, config := range configs {
		if config.Enabled {
			targetConfig = config
			break
		}
	}

	if targetConfig == nil {
		s.mu.RUnlock()
		return s.selectFromAllProviders()
	}

	// 提取需要的引用，释放锁
	provider := s.deps.Providers[targetConfig.ProviderType]
	pool := s.deps.AccountPools[targetConfig.ProviderType]
	s.mu.RUnlock()

	// 2. 在锁外选择账号（AccountPool.Select() 有自己的锁）
	if provider != nil && pool != nil {
		if account := pool.Select(); account != nil {
			return provider, account
		}
	}

	// 如果当前 provider 没有可用账号，尝试其他 provider
	return s.selectFromAllProviders()
}

// selectFromAllProviders 从所有 provider 中选择任意可用账号
// 锁已在外层释放，此处需要重新加锁获取快照
func (s *Scheduler) selectFromAllProviders() (providers.Provider, *Account) {
	// 获取当前快照
	s.mu.RLock()
	defer s.mu.RUnlock()

	for providerType, provider := range s.deps.Providers {
		if pool, ok := s.deps.AccountPools[providerType]; ok {
			if account := pool.Select(); account != nil {
				return provider, account
			}
		}
	}
	return nil, nil
}

func (s *Scheduler) ListModels() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	models := make([]string, 0, len(s.modelConfigs))
	for name := range s.modelConfigs {
		models = append(models, name)
	}
	return models
}

// GetModelOwner returns the provider name that owns the given model.
// If the model is registered under multiple providers, the first one is returned.
// Returns empty string if the model is not found.
func (s *Scheduler) GetModelOwner(modelName string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	configs, exists := s.modelConfigs[modelName]
	if !exists || len(configs) == 0 {
		return ""
	}
	provider := s.deps.Providers[configs[0].ProviderType]
	if provider == nil {
		return ""
	}
	return provider.Name()
}

func (s *Scheduler) markAccountResult(modelName, accountID string, success bool) {
	// 获取配置（短暂锁）
	s.mu.RLock()
	configs, exists := s.modelConfigs[modelName]
	if !exists || len(configs) == 0 {
		s.mu.RUnlock()
		// Model not registered — nothing to mark.
		return
	}
	s.mu.RUnlock()

	// 更新账号状态
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

	// Wrap the upstream channel so we can mark the account as successful
	// once the stream completes normally.
	wrappedChan := make(chan *models.ChatCompletionStreamResponse, cap(streamChan))
	go func() {
		defer close(wrappedChan)

		var streamErr error
		defer func() {
			if r := recover(); r != nil {
				streamErr = &models.ErrorDetail{
					Message: "panic in stream processing",
					Type:    "internal_error",
					Code:    "panic",
				}
			}
			// 根据是否有错误来标记账号状态
			s.markAccountResult(req.Model, account.ID, streamErr == nil)
		}()

		for chunk := range streamChan {
			select {
			case wrappedChan <- chunk:
			case <-ctx.Done():
				streamErr = ctx.Err()
				return
			}
		}
		// Stream finished without panic
	}()

	return wrappedChan, nil
}

// GetStats 返回调度器统计信息（用于监控）
func (s *Scheduler) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["provider_count"] = len(s.deps.Providers)
	stats["model_count"] = len(s.modelConfigs)

	accountStats := make(map[string]int)
	for providerType, pool := range s.deps.AccountPools {
		accountStats[string(providerType)] = pool.HealthyCount()
	}
	stats["healthy_accounts"] = accountStats

	return stats
}

// LogStats 打印当前调度器状态
func (s *Scheduler) LogStats() {
	stats := s.GetStats()
	log.Printf("[Scheduler] Providers: %d, Models: %d, HealthyAccounts: %v",
		stats["provider_count"],
		stats["model_count"],
		stats["healthy_accounts"],
	)
}
