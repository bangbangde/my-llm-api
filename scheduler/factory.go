package scheduler

import (
	"fmt"
	"log"

	"github.com/my-llm-api/config"
	"github.com/my-llm-api/providers"
)

// ProviderBuilder 创建 Provider 实例的函数类型
type ProviderBuilder func(cfg config.ProviderConfig) providers.Provider

// Factory 负责根据配置创建和组装组件
type Factory struct {
	providerBuilders map[string]ProviderBuilder
}

// NewFactory 创建工厂实例
func NewFactory() *Factory {
	return &Factory{
		providerBuilders: make(map[string]ProviderBuilder),
	}
}

// RegisterProviderBuilder 注册 provider 构建器
func (f *Factory) RegisterProviderBuilder(name string, builder ProviderBuilder) {
	f.providerBuilders[name] = builder
}

// BuildScheduler 根据配置构建调度器
func (f *Factory) BuildScheduler(appConfig *config.Config) (*Scheduler, error) {
	sched := NewScheduler()
	registeredCount := 0

	for providerName, providerConfig := range appConfig.Providers {
		builder, ok := f.providerBuilders[providerName]
		if !ok {
			log.Printf("[Factory] No builder registered for provider: %s, skipping", providerName)
			continue
		}

		provider := builder(providerConfig)
		accountPool := NewAccountPool(providerConfig.Accounts)

		var providerType providers.ProviderType
		switch providerName {
		case "siliconflow":
			providerType = providers.ProviderSiliconFlow
		default:
			providerType = providers.ProviderType(providerName)
		}

		sched.RegisterProvider(providerType, provider, accountPool)

		for _, modelName := range providerConfig.Models {
			sched.RegisterModel(modelName, &ModelConfig{
				ProviderType: providerType,
				ModelName:    modelName,
				Priority:     1,
				Weight:       1,
				Enabled:      true,
			})
		}

		registeredCount++
		log.Printf("[Factory] Registered provider: %s with %d accounts, %d models",
			providerName, accountPool.HealthyCount(), len(providerConfig.Models))
	}

	if registeredCount == 0 {
		return nil, fmt.Errorf("no providers registered")
	}

	return sched, nil
}

// DefaultFactory 创建带有默认 provider 构建器的工厂
func DefaultFactory() *Factory {
	f := NewFactory()

	// 注册硅基流动构建器
	f.RegisterProviderBuilder("siliconflow", func(cfg config.ProviderConfig) providers.Provider {
		return providers.NewSiliconFlowProvider(cfg.BaseURL)
	})

	return f
}
