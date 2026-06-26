package local

import (
	"context"
	"fmt"

	"github.com/travel-agent/services/plan-orchestrator/internal/client/llmgateway"
	"github.com/travel-agent/services/plan-orchestrator/internal/domain"
	"github.com/travel-agent/shared/contracts"
)

type BuildItineraryDraftTool struct {
	client   *llmgateway.Client
	enricher *LocationEnricher
}

func NewBuildItineraryDraftTool(client *llmgateway.Client, enricher *LocationEnricher) *BuildItineraryDraftTool {
	return &BuildItineraryDraftTool{
		client:   client,
		enricher: enricher,
	}
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

	enrichedCount, enrichErr := t.enrichCoordinates(ctx, req, &plan)

	output := plan.Summary
	if enrichErr != nil {
		output = fmt.Sprintf("%s\n[location enrichment warning] %v", output, enrichErr)
	}

	return domain.ToolExecution{
		Success: true,
		Output:  output,
		Meta: map[string]interface{}{
			"provider":            resp.Provider,
			"model":               resp.Model,
			"latency_ms":          resp.LatencyMs,
			"plan":                plan,
			"raw_content":         resp.Content,
			"location_enriched":   enrichedCount,
			"location_enrich_err": enrichErrString(enrichErr),
		},
	}, nil
}

func (t *BuildItineraryDraftTool) enrichCoordinates(ctx context.Context, req contracts.GeneratePlanRequest, plan *contracts.Plan) (int, error) {
	if t.enricher == nil {
		return 0, nil
	}
	return t.enricher.EnrichPlan(ctx, req.Destination, plan)
}

func enrichErrString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
