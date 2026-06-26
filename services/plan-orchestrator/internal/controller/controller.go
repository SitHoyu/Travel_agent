package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/travel-agent/services/plan-orchestrator/internal/agent"
	"github.com/travel-agent/services/plan-orchestrator/internal/domain"
	"github.com/travel-agent/services/plan-orchestrator/internal/toolkit"
	"github.com/travel-agent/shared/contracts"
	"github.com/travel-agent/shared/utils"
)

var ErrMaxStepsExceeded = errors.New("max controller steps exceeded")

type Controller struct {
	agent    agent.Agent
	toolkit  *toolkit.Registry
	maxSteps int
}

func New(agent agent.Agent, toolkit *toolkit.Registry, maxSteps int) *Controller {
	if maxSteps <= 0 {
		maxSteps = 4
	}
	return &Controller{
		agent:    agent,
		toolkit:  toolkit,
		maxSteps: maxSteps,
	}
}

func (c *Controller) Run(ctx context.Context, req contracts.GeneratePlanRequest) (domain.PlanRunResult, error) {
	requestRaw, err := json.Marshal(req)
	if err != nil {
		return domain.PlanRunResult{}, fmt.Errorf("marshal request: %w", err)
	}

	session := &domain.Session{
		ID:          utils.NewID(),
		RequestID:   req.RequestID,
		Status:      "running",
		RequestText: string(requestRaw),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Messages: []domain.Message{
			{
				Role:    "system",
				Content: "You are the travel planning runtime controller. Analyze the user request, use tools when needed, and return the final answer in Chinese.",
			},
			{
				Role:    "user",
				Content: string(requestRaw),
			},
		},
	}

	for step := 0; step < c.maxSteps; step++ {
		select {
		case <-ctx.Done():
			return domain.PlanRunResult{}, ctx.Err()
		default:
		}

		thought, err := c.agent.Think(ctx, session)
		if err != nil {
			return domain.PlanRunResult{}, fmt.Errorf("agent think: %w", err)
		}

		session.Messages = append(session.Messages, domain.Message{
			Role:      "assistant",
			Content:   thought.Text,
			ToolCalls: thought.ToolCalls,
		})
		session.UpdatedAt = time.Now()

		if thought.Done || len(thought.ToolCalls) == 0 {
			finalAnswer := strings.TrimSpace(thought.Text)
			if finalAnswer == "" {
				finalAnswer = latestDraft(session)
			}
			session.Status = "completed"
			return domain.PlanRunResult{
				SessionID:         session.ID,
				RequestID:         session.RequestID,
				Status:            session.Status,
				FinalAnswer:       finalAnswer,
				Plan:              buildPlanDraft(session, req, finalAnswer),
				HotelAreas:        latestHotelAreaRecommendation(session),
				ToolRuns:          len(session.Executions),
				MessageCount:      len(session.Messages),
				ExecutedTools:     executedToolNames(session),
				ValidationSummary: latestToolOutputByName(session, "validate_constraints"),
				ToolExecutions:    toolExecutionTraces(session),
			}, nil
		}

		for _, call := range thought.ToolCalls {
			execution, err := c.toolkit.Execute(ctx, call)
			if err != nil {
				execution = domain.ToolExecution{
					ToolCallID: call.ID,
					Name:       call.Name,
					Success:    false,
					Output:     fmt.Sprintf("tool %s failed: %v", call.Name, err),
				}
			}

			session.Executions = append(session.Executions, execution)
			session.Messages = append(session.Messages, domain.Message{
				Role:       "tool",
				Content:    execution.Output,
				ToolCallID: call.ID,
				Meta:       execution.Meta,
			})
			session.UpdatedAt = time.Now()
		}
	}

	return domain.PlanRunResult{}, ErrMaxStepsExceeded
}

func latestDraft(session *domain.Session) string {
	return latestToolOutputByName(session, "build_itinerary_draft")
}

func latestToolOutputByName(session *domain.Session, toolName string) string {
	for i := len(session.Executions) - 1; i >= 0; i-- {
		if session.Executions[i].Name == toolName && session.Executions[i].Success {
			return session.Executions[i].Output
		}
	}
	return ""
}

func executedToolNames(session *domain.Session) []string {
	if len(session.Executions) == 0 {
		return nil
	}

	names := make([]string, 0, len(session.Executions))
	for _, execution := range session.Executions {
		names = append(names, execution.Name)
	}
	return names
}

func toolExecutionTraces(session *domain.Session) []domain.ToolTrace {
	if len(session.Executions) == 0 {
		return nil
	}

	traces := make([]domain.ToolTrace, 0, len(session.Executions))
	for _, execution := range session.Executions {
		traces = append(traces, domain.ToolTrace{
			Name:    execution.Name,
			Success: execution.Success,
			Output:  truncateForResponse(execution.Output, 240),
		})
	}
	return traces
}

func truncateForResponse(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

func buildPlanDraft(session *domain.Session, req contracts.GeneratePlanRequest, content string) contracts.Plan {
	if plan := latestStructuredPlan(session); plan != nil {
		plan.ID = utils.NewID()
		plan.Status = "draft"
		if strings.TrimSpace(plan.Summary) == "" {
			plan.Summary = content
		}
		return *plan
	}

	return contracts.Plan{
		ID:          utils.NewID(),
		Status:      "draft",
		Title:       fmt.Sprintf("%s %s-%s itinerary draft", req.Destination, req.StartDate, req.EndDate),
		Destination: req.Destination,
		Summary:     content,
		Days:        []contracts.PlanDay{},
	}
}

func latestStructuredPlan(session *domain.Session) *contracts.Plan {
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

func latestHotelAreaRecommendation(session *domain.Session) contracts.HotelAreaRecommendationResult {
	for i := len(session.Executions) - 1; i >= 0; i-- {
		execution := session.Executions[i]
		if execution.Name != "recommend_hotel_area" || !execution.Success {
			continue
		}

		value, ok := execution.Meta["hotel_areas"]
		if !ok {
			continue
		}

		raw, err := json.Marshal(value)
		if err != nil {
			continue
		}

		var result contracts.HotelAreaRecommendationResult
		if err := json.Unmarshal(raw, &result); err != nil {
			continue
		}
		return result
	}
	return contracts.HotelAreaRecommendationResult{}
}
