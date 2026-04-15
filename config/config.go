package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig            `yaml:"server"`
	Providers map[string]ProviderConfig `yaml:"providers"`
}

type ServerConfig struct {
	Port     string `yaml:"port"`
	LogLevel string `yaml:"log_level"`
}

type ProviderConfig struct {
	BaseURL string         `yaml:"base_url"`
	Accounts []AccountConfig `yaml:"accounts"`
	Models  []string       `yaml:"models"`
}

type AccountConfig struct {
	ID      string `yaml:"id"`
	APIKey  string `yaml:"api_key"`
	Weight  int    `yaml:"weight"`
	Enabled bool   `yaml:"enabled"`
}

var AppConfig *Config

func LoadConfig(configPath string) error {
	if configPath == "" {
		configPath = "config.yaml"
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	AppConfig = &Config{}
	if err := yaml.Unmarshal(data, AppConfig); err != nil {
		return err
	}

	if AppConfig.Server.Port == "" {
		AppConfig.Server.Port = "8080"
	}
	if AppConfig.Server.LogLevel == "" {
		AppConfig.Server.LogLevel = "info"
	}

	return nil
}

func GetProviderConfig(providerName string) *ProviderConfig {
	if AppConfig == nil || AppConfig.Providers == nil {
		return nil
	}
	config, exists := AppConfig.Providers[providerName]
	if !exists {
		return nil
	}
	return &config
}

func GetEnabledAccounts(providerName string) []AccountConfig {
	providerConfig := GetProviderConfig(providerName)
	if providerConfig == nil {
		return nil
	}

	var enabled []AccountConfig
	for _, acc := range providerConfig.Accounts {
		if acc.Enabled {
			enabled = append(enabled, acc)
		}
	}
	return enabled
}
