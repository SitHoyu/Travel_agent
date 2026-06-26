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
	enrichHotelAreaToolCalls(session, &decision)
	enforceStageGuards(session, &decision)
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
	revisionFeedback := latestValidationFailureSummary(session)
	existingPlan := latestStructuredPlanFromSession(session)

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

		if strings.TrimSpace(weatherSummary) != "" {
			req.WeatherSummary = weatherSummary
		}
		if strings.TrimSpace(revisionFeedback) != "" {
			req.RevisionFeedback = revisionFeedback
		}
		if existingPlan != nil {
			req.ExistingPlan = existingPlan
		}
		decision.ToolCalls[i].Arguments["request"] = req
	}
}

func enrichValidationToolCalls(session *domain.Session, decision *domain.AgentDecision) {
	draft := latestSuccessfulToolOutput(session, "build_itinerary_draft")
	if strings.TrimSpace(draft) == "" {
		return
	}

	weatherSummary := latestSuccessfulToolOutput(session, "query_weather")
	plan := latestStructuredPlanFromSession(session)
	if plan == nil {
		return
	}

	for i := range decision.ToolCalls {
		if decision.ToolCalls[i].Name != "validate_constraints" {
			continue
		}

		req, err := requestFromSession(session)
		if err != nil {
			continue
		}

		if decision.ToolCalls[i].Arguments == nil {
			decision.ToolCalls[i].Arguments = map[string]interface{}{}
		}

		decision.ToolCalls[i].Arguments["request"] = req
		decision.ToolCalls[i].Arguments["draft"] = draft
		decision.ToolCalls[i].Arguments["plan"] = *plan
		if strings.TrimSpace(weatherSummary) != "" {
			decision.ToolCalls[i].Arguments["weather_summary"] = weatherSummary
		}
	}
}

func enrichHotelAreaToolCalls(session *domain.Session, decision *domain.AgentDecision) {
	plan := latestStructuredPlanFromSession(session)
	if plan == nil {
		return
	}

	for i := range decision.ToolCalls {
		if decision.ToolCalls[i].Name != "recommend_hotel_area" {
			continue
		}

		req, err := requestFromSession(session)
		if err != nil {
			continue
		}

		if decision.ToolCalls[i].Arguments == nil {
			decision.ToolCalls[i].Arguments = map[string]interface{}{}
		}

		decision.ToolCalls[i].Arguments["request"] = req
		decision.ToolCalls[i].Arguments["plan"] = *plan
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

func hasSuccessfulToolExecution(session *domain.Session, toolName string) bool {
	return strings.TrimSpace(latestSuccessfulToolOutput(session, toolName)) != ""
}

func enforceStageGuards(session *domain.Session, decision *domain.AgentDecision) {
	hasDraft := hasSuccessfulToolExecution(session, "build_itinerary_draft")
	hasValidation := hasSuccessfulToolExecution(session, "validate_constraints")
	hasHotelAreas := hasSuccessfulToolExecution(session, "recommend_hotel_area")
	validationPassed, validationFailed := latestValidationState(session)

	if !hasDraft {
		return
	}

	if validationFailed && shouldTriggerRepair(session) {
		req, err := requestFromSession(session)
		if err == nil {
			if weatherSummary := latestSuccessfulToolOutput(session, "query_weather"); strings.TrimSpace(weatherSummary) != "" {
				req.WeatherSummary = weatherSummary
			}
			if feedback := latestValidationFailureSummary(session); strings.TrimSpace(feedback) != "" {
				req.RevisionFeedback = feedback
			}
			if plan := latestStructuredPlanFromSession(session); plan != nil {
				req.ExistingPlan = plan
			}

			decision.ToolCalls = []domain.ToolCallDecision{
				{
					Name: "build_itinerary_draft",
					Arguments: map[string]interface{}{
						"request": req,
					},
				},
			}
			decision.Done = false
			decision.FinalAnswer = ""
			decision.Thought = "约束校验未通过，先根据失败原因修正行程草案。"
			return
		}
	}

	filtered := make([]domain.ToolCallDecision, 0, len(decision.ToolCalls))
	removedDraftCall := false
	for _, call := range decision.ToolCalls {
		if call.Name == "build_itinerary_draft" {
			removedDraftCall = true
			continue
		}
		filtered = append(filtered, call)
	}
	decision.ToolCalls = filtered

	if !hasValidation || (validationFailed && !shouldTriggerRepair(session)) {
		if !containsTool(decision.ToolCalls, "validate_constraints") {
			req, err := requestFromSession(session)
			if err == nil {
				plan := latestStructuredPlanFromSession(session)
				if plan == nil {
					return
				}

				callArgs := map[string]interface{}{
					"request": req,
					"draft":   latestSuccessfulToolOutput(session, "build_itinerary_draft"),
					"plan":    *plan,
				}
				if weatherSummary := latestSuccessfulToolOutput(session, "query_weather"); strings.TrimSpace(weatherSummary) != "" {
					callArgs["weather_summary"] = weatherSummary
				}

				decision.ToolCalls = append([]domain.ToolCallDecision{
					{
						Name:      "validate_constraints",
						Arguments: callArgs,
					},
				}, decision.ToolCalls...)
			}
		}

		decision.Done = false
		decision.FinalAnswer = ""
		if removedDraftCall && strings.TrimSpace(decision.Thought) == "" {
			decision.Thought = "Draft already exists. Proceeding to constraint validation."
		}
		return
	}

	if validationPassed && !hasHotelAreas {
		req, err := requestFromSession(session)
		if err == nil {
			plan := latestStructuredPlanFromSession(session)
			if plan == nil {
				return
			}

			decision.ToolCalls = []domain.ToolCallDecision{
				{
					Name: "recommend_hotel_area",
					Arguments: map[string]interface{}{
						"request": req,
						"plan":    *plan,
					},
				},
			}
			decision.Done = false
			decision.FinalAnswer = ""
			if strings.TrimSpace(decision.Thought) == "" {
				decision.Thought = "行程与约束校验已完成，补充住宿区域建议。"
			}
			return
		}
	}

	if validationPassed && hasHotelAreas {
		decision.ToolCalls = nil
	}
}

func containsTool(calls []domain.ToolCallDecision, name string) bool {
	for _, call := range calls {
		if call.Name == name {
			return true
		}
	}
	return false
}

func requestFromSession(session *domain.Session) (contracts.GeneratePlanRequest, error) {
	var req contracts.GeneratePlanRequest
	if err := json.Unmarshal([]byte(session.RequestText), &req); err != nil {
		return contracts.GeneratePlanRequest{}, err
	}
	return req, nil
}

func latestStructuredPlanFromSession(session *domain.Session) *contracts.Plan {
	for i := len(session.Executions) - 1; i >= 0; i-- {
		execution := session.Executions[i]
		if execution.Name != "build_itinerary_draft" || !execution.Success {
			continue
		}

		planValue, ok := execution.Meta["plan"]
		if !ok {
			continue
		}

		raw, err := json.Marshal(planValue)
		if err != nil {
			continue
		}

		var plan contracts.Plan
		if err := json.Unmarshal(raw, &plan); err != nil {
			continue
		}
		return &plan
	}
	return nil
}

func latestValidationState(session *domain.Session) (passed bool, failed bool) {
	for i := len(session.Executions) - 1; i >= 0; i-- {
		execution := session.Executions[i]
		if execution.Name != "validate_constraints" || !execution.Success {
			continue
		}

		value, ok := execution.Meta["passed"]
		if !ok {
			break
		}

		passedValue, ok := value.(bool)
		if !ok {
			break
		}

		if passedValue {
			return true, false
		}
		return false, true
	}
	return false, false
}

func latestValidationFailureSummary(session *domain.Session) string {
	passed, failed := latestValidationState(session)
	if passed || !failed {
		return ""
	}
	return latestSuccessfulToolOutput(session, "validate_constraints")
}

func shouldTriggerRepair(session *domain.Session) bool {
	lastValidationIndex := latestSuccessfulExecutionIndex(session, "validate_constraints")
	if lastValidationIndex < 0 {
		return false
	}

	lastDraftIndex := latestSuccessfulExecutionIndex(session, "build_itinerary_draft")
	if lastDraftIndex < 0 {
		return false
	}

	if countSuccessfulToolExecutions(session, "build_itinerary_draft") >= 2 {
		return false
	}

	return lastDraftIndex < lastValidationIndex
}

func latestSuccessfulExecutionIndex(session *domain.Session, toolName string) int {
	for i := len(session.Executions) - 1; i >= 0; i-- {
		execution := session.Executions[i]
		if execution.Name == toolName && execution.Success {
			return i
		}
	}
	return -1
}

func countSuccessfulToolExecutions(session *domain.Session, toolName string) int {
	count := 0
	for _, execution := range session.Executions {
		if execution.Name == toolName && execution.Success {
			count++
		}
	}
	return count
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
