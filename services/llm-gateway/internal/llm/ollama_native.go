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

type OllamaNativeProvider struct {
	name       string
	config     appconfig.ProviderConfig
	httpClient *http.Client
}

type ollamaGenerateRequest struct {
	Model   string `json:"model"`
	Prompt  string `json:"prompt"`
	System  string `json:"system,omitempty"`
	Stream  bool   `json:"stream"`
	Think   bool   `json:"think,omitempty"`
	Format  string `json:"format,omitempty"`
	Options any    `json:"options,omitempty"`
}

type ollamaGenerateResponse struct {
	Model              string `json:"model"`
	Response           string `json:"response"`
	Thinking           string `json:"thinking"`
	Done               bool   `json:"done"`
	DoneReason         string `json:"done_reason"`
	PromptEvalCount    int    `json:"prompt_eval_count"`
	PromptEvalDuration int64  `json:"prompt_eval_duration"`
	EvalCount          int    `json:"eval_count"`
	EvalDuration       int64  `json:"eval_duration"`
}

func NewOllamaNativeProvider(name string, cfg appconfig.ProviderConfig) *OllamaNativeProvider {
	return &OllamaNativeProvider{
		name:   name,
		config: cfg,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.TimeoutSec) * time.Second,
		},
	}
}

func (p *OllamaNativeProvider) Name() string {
	return p.name
}

func (p *OllamaNativeProvider) Generate(ctx context.Context, input GenerateInput) (GenerateOutput, error) {
	model := input.Model
	if model == "" {
		model = p.config.DefaultModel
	}

	bodyReq := ollamaGenerateRequest{
		Model:  model,
		Prompt: input.Prompt,
		System: fallbackSystem(input.System),
		Stream: false,
		Think:  true,
	}

	payload, err := json.Marshal(bodyReq)
	if err != nil {
		return GenerateOutput{}, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := strings.TrimRight(p.config.BaseURL, "/") + "/api/generate"
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
		return GenerateOutput{}, fmt.Errorf("call ollama: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return GenerateOutput{}, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 300 {
		return GenerateOutput{}, fmt.Errorf("ollama status %d: %s", resp.StatusCode, string(rawBody))
	}

	var parsed ollamaGenerateResponse
	if err := json.Unmarshal(rawBody, &parsed); err != nil {
		return GenerateOutput{}, fmt.Errorf("decode response: %w", err)
	}

	return GenerateOutput{
		Provider: p.name,
		Model:    firstNonEmpty(parsed.Model, model),
		Content:  parsed.Response,
		Usage: contracts.LLMUsage{
			PromptTokens:     parsed.PromptEvalCount,
			CompletionTokens: parsed.EvalCount,
			TotalTokens:      parsed.PromptEvalCount + parsed.EvalCount,
		},
		Raw: map[string]any{
			"thinking":             parsed.Thinking,
			"done":                 parsed.Done,
			"done_reason":          parsed.DoneReason,
			"prompt_eval_count":    parsed.PromptEvalCount,
			"prompt_eval_duration": parsed.PromptEvalDuration,
			"eval_count":           parsed.EvalCount,
			"eval_duration":        parsed.EvalDuration,
		},
	}, nil
}
