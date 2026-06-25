package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/travel-agent/services/llm-gateway/internal/llm"
	"github.com/travel-agent/shared/contracts"
)

type PromptRenderer interface {
	Render(name string, variables map[string]any) (string, error)
}

type ProviderRegistry interface {
	Get(name string) (llm.Provider, error)
}

type Service struct {
	prompts  PromptRenderer
	registry ProviderRegistry
}

func New(prompts PromptRenderer, registry ProviderRegistry) *Service {
	return &Service{
		prompts:  prompts,
		registry: registry,
	}
}

func (s *Service) Generate(ctx context.Context, req contracts.LLMGenerateRequest) (contracts.LLMGenerateResponse, error) {
	if strings.TrimSpace(req.Provider) == "" {
		return contracts.LLMGenerateResponse{}, fmt.Errorf("provider is required")
	}
	if strings.TrimSpace(req.Template) == "" {
		return contracts.LLMGenerateResponse{}, fmt.Errorf("template is required")
	}

	provider, err := s.registry.Get(req.Provider)
	if err != nil {
		return contracts.LLMGenerateResponse{}, err
	}

	prompt, err := s.prompts.Render(req.Template, req.Variables)
	if err != nil {
		return contracts.LLMGenerateResponse{}, err
	}

	startedAt := time.Now()
	output, err := provider.Generate(ctx, llm.GenerateInput{
		Provider:    req.Provider,
		Model:       req.Model,
		System:      req.System,
		Prompt:      prompt,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	})
	if err != nil {
		return contracts.LLMGenerateResponse{}, err
	}

	return contracts.LLMGenerateResponse{
		RequestID: req.RequestID,
		Provider:  output.Provider,
		Model:     output.Model,
		Prompt:    prompt,
		Content:   output.Content,
		Usage:     output.Usage,
		LatencyMs: time.Since(startedAt).Milliseconds(),
		Raw:       output.Raw,
	}, nil
}

func (s *Service) GeneratePlan(ctx context.Context, req contracts.GeneratePlanRequest, providerName, model string) (contracts.LLMGenerateResponse, error) {
	variables := map[string]any{
		"request_id":        req.RequestID,
		"destination":       req.Destination,
		"start_date":        req.StartDate,
		"end_date":          req.EndDate,
		"budget":            req.Budget,
		"travelers":         req.Travelers,
		"preferences":       strings.Join(req.Preferences, ", "),
		"constraints":       strings.Join(req.Constraints, ", "),
		"weather_summary":   req.WeatherSummary,
		"revision_feedback": req.RevisionFeedback,
		"existing_plan":     mustJSON(req.ExistingPlan),
		"payload":           mustJSON(req),
	}

	return s.Generate(ctx, contracts.LLMGenerateRequest{
		RequestID:   req.RequestID,
		Provider:    providerName,
		Model:       model,
		Template:    "planner",
		Variables:   variables,
		System:      "You are a travel planning assistant. Return a practical itinerary draft as valid JSON.",
		Temperature: 0.4,
		MaxTokens:   1200,
	})
}

func (s *Service) RevisePlan(ctx context.Context, req contracts.RevisePlanRequest, providerName, model string) (contracts.LLMGenerateResponse, error) {
	variables := map[string]any{
		"plan_id":    req.PlanID,
		"feedback":   req.Feedback,
		"keep_items": strings.Join(req.KeepItems, ", "),
		"payload":    mustJSON(req),
	}

	return s.Generate(ctx, contracts.LLMGenerateRequest{
		RequestID:   req.PlanID,
		Provider:    providerName,
		Model:       model,
		Template:    "reviser",
		Variables:   variables,
		System:      "You revise travel itineraries while preserving user constraints.",
		Temperature: 0.3,
		MaxTokens:   1200,
	})
}

func mustJSON(v any) string {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(raw)
}
