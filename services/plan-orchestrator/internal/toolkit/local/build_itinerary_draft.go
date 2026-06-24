package local

import (
	"context"
	"fmt"

	"github.com/travel-agent/services/plan-orchestrator/internal/client/llmgateway"
	"github.com/travel-agent/services/plan-orchestrator/internal/domain"
)

type BuildItineraryDraftTool struct {
	client *llmgateway.Client
}

func NewBuildItineraryDraftTool(client *llmgateway.Client) *BuildItineraryDraftTool {
	return &BuildItineraryDraftTool{client: client}
}

func (t *BuildItineraryDraftTool) Name() string {
	return "build_itinerary_draft"
}

func (t *BuildItineraryDraftTool) Description() string {
	return "Generate a structured travel itinerary draft from the original travel request by calling the llm-gateway."
}

func (t *BuildItineraryDraftTool) Execute(ctx context.Context, args map[string]interface{}) (domain.ToolExecution, error) {
	requestValue, ok := args["request"]
	if !ok {
		return domain.ToolExecution{}, fmt.Errorf("missing request argument")
	}

	req, err := decodeGeneratePlanRequest(requestValue)
	if err != nil {
		return domain.ToolExecution{}, err
	}

	resp, err := t.client.GeneratePlan(ctx, req)
	if err != nil {
		return domain.ToolExecution{}, err
	}

	plan, err := parseGeneratedPlan(resp.Content)
	if err != nil {
		return domain.ToolExecution{}, fmt.Errorf("parse structured itinerary: %w", err)
	}

	return domain.ToolExecution{
		Success: true,
		Output:  plan.Summary,
		Meta: map[string]interface{}{
			"provider":    resp.Provider,
			"model":       resp.Model,
			"latency_ms":  resp.LatencyMs,
			"plan":        plan,
			"raw_content": resp.Content,
		},
	}, nil
}
