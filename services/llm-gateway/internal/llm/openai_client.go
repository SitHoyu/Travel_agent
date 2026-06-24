package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	appconfig "github.com/travel-agent/services/llm-gateway/internal/config"
	"github.com/travel-agent/shared/contracts"
)

type OpenAICompatibleProvider struct {
	name       string
	config     appconfig.ProviderConfig
	httpClient *http.Client
}

type chatCompletionsRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionsResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type errorResponse struct {
	Error any `json:"error"`
}

func NewOpenAICompatibleProvider(name string, cfg appconfig.ProviderConfig) *OpenAICompatibleProvider {
	return &OpenAICompatibleProvider{
		name:   name,
		config: cfg,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.TimeoutSec) * time.Second,
		},
	}
}

func (p *OpenAICompatibleProvider) Name() string {
	return p.name
}

func (p *OpenAICompatibleProvider) Generate(ctx context.Context, input GenerateInput) (GenerateOutput, error) {
	if p.config.APIKey == "" && !strings.Contains(strings.ToLower(p.name), "ollama") {
		return GenerateOutput{}, fmt.Errorf("provider %s api key is empty", p.name)
	}

	model := input.Model
	if model == "" {
		model = p.config.DefaultModel
	}

	requestBody := chatCompletionsRequest{
		Model: model,
		Messages: []chatMessage{
			{Role: "system", Content: fallbackSystem(input.System)},
			{Role: "user", Content: input.Prompt},
		},
		Temperature: input.Temperature,
		MaxTokens:   input.MaxTokens,
	}

	payload, err := json.Marshal(requestBody)
	if err != nil {
		return GenerateOutput{}, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := strings.TrimRight(p.config.BaseURL, "/") + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return GenerateOutput{}, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return GenerateOutput{}, fmt.Errorf("call provider: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return GenerateOutput{}, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 300 {
		var providerErr errorResponse
		_ = json.Unmarshal(body, &providerErr)
		return GenerateOutput{}, fmt.Errorf("provider status %d: %v", resp.StatusCode, providerErr.Error)
	}

	var parsed chatCompletionsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return GenerateOutput{}, fmt.Errorf("decode response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return GenerateOutput{}, fmt.Errorf("provider %s returned no choices", p.name)
	}

	return GenerateOutput{
		Provider: p.name,
		Model:    firstNonEmpty(parsed.Model, model),
		Content:  parsed.Choices[0].Message.Content,
		Usage: contracts.LLMUsage{
			PromptTokens:     parsed.Usage.PromptTokens,
			CompletionTokens: parsed.Usage.CompletionTokens,
			TotalTokens:      parsed.Usage.TotalTokens,
		},
		Raw: map[string]any{
			"choices_count": len(parsed.Choices),
		},
	}, nil
}

func fallbackSystem(system string) string {
	if strings.TrimSpace(system) == "" {
		return "You are a reliable travel planning assistant. Return concise and structured outputs."
	}
	return system
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
