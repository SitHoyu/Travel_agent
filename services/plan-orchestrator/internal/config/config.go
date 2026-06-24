package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server     ServerConfig     `yaml:"server"`
	LLMGateway LLMGatewayConfig `yaml:"llm_gateway"`
	AMap       AMapConfig       `yaml:"amap"`
	Controller ControllerConfig `yaml:"controller"`
	Storage    StorageConfig    `yaml:"storage"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
}

type LLMGatewayConfig struct {
	BaseURL  string `yaml:"base_url"`
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
}

type AMapConfig struct {
	BaseURL    string `yaml:"base_url"`
	APIKey     string `yaml:"api_key"`
	APIKeyEnv  string `yaml:"api_key_env"`
	AdcodeFile string `yaml:"adcode_file"`
}

type ControllerConfig struct {
	MaxSteps int `yaml:"max_steps"`
}

type StorageConfig struct {
	Driver string `yaml:"driver"`
	DSN    string `yaml:"dsn"`
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

	if cfg.Server.Port == "" {
		cfg.Server.Port = "8080"
	}
	if cfg.Controller.MaxSteps <= 0 {
		cfg.Controller.MaxSteps = 4
	}
	if cfg.LLMGateway.BaseURL == "" {
		cfg.LLMGateway.BaseURL = "http://localhost:8081"
	}
	if cfg.AMap.BaseURL == "" {
		cfg.AMap.BaseURL = "https://restapi.amap.com"
	}
	if cfg.AMap.APIKey == "" && cfg.AMap.APIKeyEnv != "" {
		cfg.AMap.APIKey = os.Getenv(cfg.AMap.APIKeyEnv)
	}
	if cfg.AMap.AdcodeFile == "" {
		cfg.AMap.AdcodeFile = "../../AMap_adcode_citycode.xlsx"
	}
	if cfg.Storage.Driver == "" {
		cfg.Storage.Driver = "memory"
	}

	return cfg, nil
}
