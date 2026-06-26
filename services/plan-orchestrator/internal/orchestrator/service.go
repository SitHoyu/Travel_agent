package orchestrator

import (
	"context"

	"github.com/travel-agent/services/plan-orchestrator/internal/controller"
	"github.com/travel-agent/services/plan-orchestrator/internal/domain"
	planrepo "github.com/travel-agent/services/plan-orchestrator/internal/repository/plan"
	"github.com/travel-agent/shared/contracts"
)

type Service struct {
	controller *controller.Controller
	repository planrepo.Repository
}

func NewService(controller *controller.Controller, repository planrepo.Repository) *Service {
	return &Service{
		controller: controller,
		repository: repository,
	}
}

func (s *Service) RunPlan(ctx context.Context, req contracts.GeneratePlanRequest) (contracts.AgentPlanResponse, error) {
	result, err := s.controller.Run(ctx, req)
	if err != nil {
		return contracts.AgentPlanResponse{}, err
	}

	if err := s.repository.Save(ctx, result); err != nil {
		return contracts.AgentPlanResponse{}, err
	}

	return toResponse(result), nil
}

func toResponse(result domain.PlanRunResult) contracts.AgentPlanResponse {
	return contracts.AgentPlanResponse{
		SessionID:         result.SessionID,
		RequestID:         result.RequestID,
		Status:            result.Status,
		FinalAnswer:       result.FinalAnswer,
		Plan:              result.Plan,
		HotelAreas:        result.HotelAreas,
		ToolRuns:          result.ToolRuns,
		MessageCount:      result.MessageCount,
		ExecutedTools:     result.ExecutedTools,
		ValidationSummary: result.ValidationSummary,
		ToolExecutions:    toToolTraces(result.ToolExecutions),
	}
}

func toToolTraces(traces []domain.ToolTrace) []contracts.ToolTrace {
	if len(traces) == 0 {
		return nil
	}

	result := make([]contracts.ToolTrace, 0, len(traces))
	for _, trace := range traces {
		result = append(result, contracts.ToolTrace{
			Name:    trace.Name,
			Success: trace.Success,
			Output:  trace.Output,
		})
	}
	return result
}
