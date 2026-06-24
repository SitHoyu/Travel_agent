package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig              `yaml:"server"`
	Prompts   PromptConfig              `yaml:"prompts"`
	Providers map[string]ProviderConfig `yaml:"providers"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
}

type PromptConfig struct {
	BaseDir string `yaml:"base_dir"`
}

type ProviderConfig struct {
	Kind        string `yaml:"kind"`
	BaseURL     string `yaml:"base_url"`
	APIKey      string `yaml:"api_key"`
	APIKeyEnv   string `yaml:"api_key_env"`
	DefaultModel string `yaml:"default_model"`
	Enabled     bool   `yaml:"enabled"`
	TimeoutSec  int    `yaml:"timeout_sec"`
}

func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	for name, provider := range cfg.Providers {
		if provider.APIKey == "" && provider.APIKeyEnv != "" {
			provider.APIKey = os.Getenv(provider.APIKeyEnv)
		}
		if provider.TimeoutSec <= 0 {
			provider.TimeoutSec = 60
		}
		cfg.Providers[name] = provider
	}

	if cfg.Server.Port == "" {
		cfg.Server.Port = "8081"
	}

	return cfg, nil
}
