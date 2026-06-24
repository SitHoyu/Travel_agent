package llm

import (
	"context"
	"errors"
	"fmt"

	appconfig "github.com/travel-agent/services/llm-gateway/internal/config"
	"github.com/travel-agent/shared/contracts"
)

var ErrProviderNotFound = errors.New("provider not found")

type GenerateInput struct {
	Provider    string
	Model       string
	System      string
	Prompt      string
	Temperature float64
	MaxTokens   int
}

type GenerateOutput struct {
	Provider string
	Model    string
	Content  string
	Usage    contracts.LLMUsage
	Raw      map[string]any
}

type Provider interface {
	Name() string
	Generate(context.Context, GenerateInput) (GenerateOutput, error)
}

type Registry struct {
	providers map[string]Provider
}

func NewRegistry(configs map[string]appconfig.ProviderConfig) (*Registry, error) {
	providers := make(map[string]Provider, len(configs))
	for name, cfg := range configs {
		if !cfg.Enabled {
			continue
		}

		var provider Provider
		switch cfg.Kind {
		case "openai-compatible":
			provider = NewOpenAICompatibleProvider(name, cfg)
		case "ollama-native":
			provider = NewOllamaNativeProvider(name, cfg)
		default:
			return nil, fmt.Errorf("unsupported provider kind %q for %s", cfg.Kind, name)
		}
		providers[name] = provider
	}

	return &Registry{providers: providers}, nil
}

func (r *Registry) Get(name string) (Provider, error) {
	provider, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, name)
	}
	return provider, nil
}
