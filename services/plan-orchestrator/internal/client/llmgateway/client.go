package llmgateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/travel-agent/shared/contracts"
)

type Client struct {
	baseURL         string
	defaultProvider string
	defaultModel    string
	httpClient      *http.Client
}

type generatePlanRequest struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	contracts.GeneratePlanRequest
}

type generateRequest struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	contracts.LLMGenerateRequest
}

func NewClient(baseURL, provider, model string) *Client {
	return &Client{
		baseURL:         strings.TrimRight(baseURL, "/"),
		defaultProvider: provider,
		defaultModel:    model,
		httpClient: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

func (c *Client) GeneratePlan(ctx context.Context, req contracts.GeneratePlanRequest) (contracts.LLMGenerateResponse, error) {
	payload, err := json.Marshal(generatePlanRequest{
		Provider:            c.defaultProvider,
		Model:               c.defaultModel,
		GeneratePlanRequest: req,
	})
	if err != nil {
		return contracts.LLMGenerateResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := c.baseURL + "/v1/travel/plan/generate"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return contracts.LLMGenerateResponse{}, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return contracts.LLMGenerateResponse{}, fmt.Errorf("call llm-gateway: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return contracts.LLMGenerateResponse{}, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return contracts.LLMGenerateResponse{}, fmt.Errorf("llm-gateway status %d: %s", resp.StatusCode, string(body))
	}

	var parsed contracts.LLMGenerateResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return contracts.LLMGenerateResponse{}, fmt.Errorf("decode response: %w", err)
	}
	return parsed, nil
}

func (c *Client) Generate(ctx context.Context, req contracts.LLMGenerateRequest) (contracts.LLMGenerateResponse, error) {
	payload, err := json.Marshal(generateRequest{
		Provider: c.defaultProvider,
		Model:    c.defaultModel,
		LLMGenerateRequest: contracts.LLMGenerateRequest{
			RequestID:   req.RequestID,
			Provider:    firstNonEmpty(req.Provider, c.defaultProvider),
			Model:       firstNonEmpty(req.Model, c.defaultModel),
			Template:    req.Template,
			Variables:   req.Variables,
			System:      req.System,
			Temperature: req.Temperature,
			MaxTokens:   req.MaxTokens,
		},
	})
	if err != nil {
		return contracts.LLMGenerateResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := c.baseURL + "/v1/generate"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return contracts.LLMGenerateResponse{}, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return contracts.LLMGenerateResponse{}, fmt.Errorf("call llm-gateway: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return contracts.LLMGenerateResponse{}, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return contracts.LLMGenerateResponse{}, fmt.Errorf("llm-gateway status %d: %s", resp.StatusCode, string(body))
	}

	var parsed contracts.LLMGenerateResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return contracts.LLMGenerateResponse{}, fmt.Errorf("decode response: %w", err)
	}
	return parsed, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
