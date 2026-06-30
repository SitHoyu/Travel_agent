package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

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

// GeneratePlan runs the planning workflow without persisting the result.
func (s *Service) GeneratePlan(ctx context.Context, req contracts.GeneratePlanRequest) (contracts.AgentPlanResponse, error) {
	result, err := s.controller.Run(ctx, req)
	if err != nil {
		return contracts.AgentPlanResponse{}, err
	}

	return toResponse(result), nil
}

// SavePlan persists a user-confirmed plan. User identity is temporarily supplied
// in the request body and will later be replaced by auth middleware context.
func (s *Service) SavePlan(ctx context.Context, req contracts.SavePlanRequest) (contracts.SavedPlanResponse, error) {
	if req.UserID <= 0 {
		return contracts.SavedPlanResponse{}, fmt.Errorf("user_id is required")
	}
	if strings.TrimSpace(req.Result.SessionID) == "" {
		return contracts.SavedPlanResponse{}, fmt.Errorf("result.session_id is required")
	}
	if strings.TrimSpace(req.Result.Plan.Title) == "" {
		return contracts.SavedPlanResponse{}, fmt.Errorf("result.plan.title is required")
	}

	record, err := toPlanRecord(req)
	if err != nil {
		return contracts.SavedPlanResponse{}, err
	}

	saved, err := s.repository.Create(ctx, record)
	if err != nil {
		return contracts.SavedPlanResponse{}, err
	}

	return toSavedPlanResponse(saved), nil
}

func (s *Service) ListPlans(ctx context.Context, userID int64, page, pageSize int) (contracts.ListPlansResponse, error) {
	if userID <= 0 {
		return contracts.ListPlansResponse{}, fmt.Errorf("user_id is required")
	}

	records, total, err := s.repository.ListByUserID(ctx, userID, page, pageSize)
	if err != nil {
		return contracts.ListPlansResponse{}, err
	}

	items := make([]contracts.PlanListItem, 0, len(records))
	for _, record := range records {
		items = append(items, contracts.PlanListItem{
			ID:          record.ID,
			UserID:      record.UserID,
			RequestID:   record.RequestID,
			SessionID:   record.SessionID,
			Status:      record.Status,
			Title:       record.Title,
			Destination: record.Destination,
			Summary:     record.Summary,
			CreatedAt:   record.CreatedAt,
			UpdatedAt:   record.UpdatedAt,
		})
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	return contracts.ListPlansResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *Service) GetPlan(ctx context.Context, userID, planID int64) (contracts.SavedPlanResponse, bool, error) {
	if userID <= 0 {
		return contracts.SavedPlanResponse{}, false, fmt.Errorf("user_id is required")
	}
	if planID <= 0 {
		return contracts.SavedPlanResponse{}, false, fmt.Errorf("plan_id is required")
	}

	record, ok, err := s.repository.GetByIDAndUserID(ctx, planID, userID)
	if err != nil {
		return contracts.SavedPlanResponse{}, false, err
	}
	if !ok {
		return contracts.SavedPlanResponse{}, false, nil
	}
	return toSavedPlanResponse(record), true, nil
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

func toPlanRecord(req contracts.SavePlanRequest) (domain.PlanRecord, error) {
	requestJSON, err := marshalJSONString(req.Request)
	if err != nil {
		return domain.PlanRecord{}, fmt.Errorf("marshal request payload: %w", err)
	}

	planJSON, err := marshalJSONString(req.Result.Plan)
	if err != nil {
		return domain.PlanRecord{}, fmt.Errorf("marshal plan: %w", err)
	}

	hotelAreasJSON, err := marshalJSONString(req.Result.HotelAreas)
	if err != nil {
		return domain.PlanRecord{}, fmt.Errorf("marshal hotel areas: %w", err)
	}

	executedToolsJSON, err := marshalJSONString(req.Result.ExecutedTools)
	if err != nil {
		return domain.PlanRecord{}, fmt.Errorf("marshal executed tools: %w", err)
	}

	toolExecutionsJSON, err := marshalJSONString(req.Result.ToolExecutions)
	if err != nil {
		return domain.PlanRecord{}, fmt.Errorf("marshal tool executions: %w", err)
	}

	now := time.Now()
	return domain.PlanRecord{
		UserID:             req.UserID,
		Request:            req.Request,
		RequestID:          req.Result.RequestID,
		SessionID:          req.Result.SessionID,
		Status:             "saved",
		Title:              req.Result.Plan.Title,
		Destination:        req.Result.Plan.Destination,
		Summary:            req.Result.Plan.Summary,
		FinalAnswer:        req.Result.FinalAnswer,
		ValidationSummary:  req.Result.ValidationSummary,
		Plan:               req.Result.Plan,
		HotelAreas:         req.Result.HotelAreas,
		ExecutedTools:      append([]string(nil), req.Result.ExecutedTools...),
		ToolExecutions:     toDomainToolTraces(req.Result.ToolExecutions),
		RequestPayloadJSON: requestJSON,
		PlanJSON:           planJSON,
		HotelAreasJSON:     hotelAreasJSON,
		ExecutedToolsJSON:  executedToolsJSON,
		ToolExecutionsJSON: toolExecutionsJSON,
		CreatedAt:          now,
		UpdatedAt:          now,
	}, nil
}

func toSavedPlanResponse(record domain.PlanRecord) contracts.SavedPlanResponse {
	return contracts.SavedPlanResponse{
		ID:                record.ID,
		UserID:            record.UserID,
		RequestID:         record.RequestID,
		SessionID:         record.SessionID,
		Status:            record.Status,
		Title:             record.Title,
		Destination:       record.Destination,
		Summary:           record.Summary,
		FinalAnswer:       record.FinalAnswer,
		ValidationSummary: record.ValidationSummary,
		Plan:              record.Plan,
		HotelAreas:        record.HotelAreas,
		ExecutedTools:     append([]string(nil), record.ExecutedTools...),
		ToolExecutions:    toToolTraces(record.ToolExecutions),
		CreatedAt:         record.CreatedAt,
		UpdatedAt:         record.UpdatedAt,
	}
}

func toDomainToolTraces(traces []contracts.ToolTrace) []domain.ToolTrace {
	if len(traces) == 0 {
		return nil
	}

	result := make([]domain.ToolTrace, 0, len(traces))
	for _, trace := range traces {
		result = append(result, domain.ToolTrace{
			Name:    trace.Name,
			Success: trace.Success,
			Output:  trace.Output,
		})
	}
	return result
}

func marshalJSONString(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func ParsePositiveInt64(value string, fieldName string) (int64, error) {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", fieldName)
	}
	return parsed, nil
}
