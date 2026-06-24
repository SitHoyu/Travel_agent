package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/travel-agent/services/plan-orchestrator/internal/client/llmgateway"
	"github.com/travel-agent/services/plan-orchestrator/internal/domain"
	"github.com/travel-agent/services/plan-orchestrator/internal/toolkit"
	"github.com/travel-agent/shared/contracts"
	"github.com/travel-agent/shared/utils"
)

type LLMAgent struct {
	client *llmgateway.Client
	tools  *toolkit.Registry
}

func NewLLMAgent(client *llmgateway.Client, tools *toolkit.Registry) *LLMAgent {
	return &LLMAgent{
		client: client,
		tools:  tools,
	}
}

func (a *LLMAgent) Think(ctx context.Context, session *domain.Session) (domain.Thought, error) {
	toolSpecs, err := json.MarshalIndent(toToolSpecs(a.tools.List()), "", "  ")
	if err != nil {
		return domain.Thought{}, fmt.Errorf("marshal tools: %w", err)
	}
	messageJSON, err := json.MarshalIndent(session.Messages, "", "  ")
	if err != nil {
		return domain.Thought{}, fmt.Errorf("marshal messages: %w", err)
	}
	executionJSON, err := json.MarshalIndent(session.Executions, "", "  ")
	if err != nil {
		return domain.Thought{}, fmt.Errorf("marshal executions: %w", err)
	}

	resp, err := a.client.Generate(ctx, buildDecisionRequest(session, string(toolSpecs), string(messageJSON), string(executionJSON)))
	if err != nil {
		return domain.Thought{}, err
	}

	decision, err := parseDecision(resp.Content)
	if err != nil {
		return domain.Thought{}, fmt.Errorf("parse agent decision: %w", err)
	}

	enrichDraftToolCalls(session, &decision)
	enrichValidationToolCalls(session, &decision)
	return toThought(decision), nil
}

type toolSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func toToolSpecs(tools []toolkit.Tool) []toolSpec {
	specs := make([]toolSpec, 0, len(tools))
	for _, tool := range tools {
		specs = append(specs, toolSpec{
			Name:        tool.Name(),
			Description: tool.Description(),
		})
	}
	return specs
}

func buildDecisionRequest(session *domain.Session, tools, messages, executions string) contracts.LLMGenerateRequest {
	return contracts.LLMGenerateRequest{
		RequestID:   session.RequestID,
		Template:    "agent_decision",
		System:      "You are a structured agent runtime. Return valid JSON only.",
		Temperature: 0.1,
		MaxTokens:   1200,
		Variables: map[string]any{
			"tools":      tools,
			"messages":   messages,
			"executions": executions,
			"request":    session.RequestText,
		},
	}
}

func parseDecision(raw string) (domain.AgentDecision, error) {
	cleaned := extractJSON(raw)

	var decision domain.AgentDecision
	if err := json.Unmarshal([]byte(cleaned), &decision); err != nil {
		return domain.AgentDecision{}, err
	}
	return decision, nil
}

func toThought(decision domain.AgentDecision) domain.Thought {
	toolCalls := make([]domain.ToolCall, 0, len(decision.ToolCalls))
	for _, call := range decision.ToolCalls {
		toolCalls = append(toolCalls, domain.ToolCall{
			ID:        call.Name + "-" + utils.NewID(),
			Name:      call.Name,
			Arguments: call.Arguments,
		})
	}

	text := strings.TrimSpace(decision.Thought)
	if decision.Done && strings.TrimSpace(decision.FinalAnswer) != "" {
		text = decision.FinalAnswer
	}

	return domain.Thought{
		Text:      text,
		ToolCalls: toolCalls,
		Done:      decision.Done,
	}
}

func enrichDraftToolCalls(session *domain.Session, decision *domain.AgentDecision) {
	weatherSummary := latestSuccessfulToolOutput(session, "query_weather")
	if strings.TrimSpace(weatherSummary) == "" {
		return
	}

	for i := range decision.ToolCalls {
		if decision.ToolCalls[i].Name != "build_itinerary_draft" {
			continue
		}

		requestValue, ok := decision.ToolCalls[i].Arguments["request"]
		if !ok {
			continue
		}

		raw, err := json.Marshal(requestValue)
		if err != nil {
			continue
		}

		var req contracts.GeneratePlanRequest
		if err := json.Unmarshal(raw, &req); err != nil {
			continue
		}

		req.WeatherSummary = weatherSummary
		decision.ToolCalls[i].Arguments["request"] = req
	}
}

func latestSuccessfulToolOutput(session *domain.Session, toolName string) string {
	for i := len(session.Executions) - 1; i >= 0; i-- {
		execution := session.Executions[i]
		if execution.Name == toolName && execution.Success {
			return execution.Output
		}
	}
	return ""
}

func enrichValidationToolCalls(session *domain.Session, decision *domain.AgentDecision) {
	draft := latestSuccessfulToolOutput(session, "build_itinerary_draft")
	if strings.TrimSpace(draft) == "" {
		return
	}

	weatherSummary := latestSuccessfulToolOutput(session, "query_weather")

	for i := range decision.ToolCalls {
		if decision.ToolCalls[i].Name != "validate_constraints" {
			continue
		}

		requestValue, ok := decision.ToolCalls[i].Arguments["request"]
		if !ok {
			continue
		}

		raw, err := json.Marshal(requestValue)
		if err != nil {
			continue
		}

		var req contracts.GeneratePlanRequest
		if err := json.Unmarshal(raw, &req); err != nil {
			continue
		}

		decision.ToolCalls[i].Arguments["request"] = req
		decision.ToolCalls[i].Arguments["draft"] = draft
		if strings.TrimSpace(weatherSummary) != "" {
			decision.ToolCalls[i].Arguments["weather_summary"] = weatherSummary
		}
	}
}

func extractJSON(raw string) string {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	start := strings.Index(cleaned, "{")
	end := strings.LastIndex(cleaned, "}")
	if start >= 0 && end > start {
		return cleaned[start : end+1]
	}
	return cleaned
}
