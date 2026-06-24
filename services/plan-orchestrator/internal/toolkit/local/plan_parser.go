package local

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/travel-agent/shared/contracts"
)

type generatedPlanPayload struct {
	Title       string              `json:"title"`
	Destination string              `json:"destination"`
	Summary     string              `json:"summary"`
	Days        []contracts.PlanDay `json:"days"`
}

func parseGeneratedPlan(raw string) (contracts.Plan, error) {
	cleaned := extractJSONObject(raw)

	var payload generatedPlanPayload
	if err := json.Unmarshal([]byte(cleaned), &payload); err != nil {
		return contracts.Plan{}, fmt.Errorf("decode generated plan: %w", err)
	}

	if strings.TrimSpace(payload.Destination) == "" {
		return contracts.Plan{}, fmt.Errorf("generated plan destination is empty")
	}
	if len(payload.Days) == 0 {
		return contracts.Plan{}, fmt.Errorf("generated plan days are empty")
	}

	return contracts.Plan{
		Title:       payload.Title,
		Destination: payload.Destination,
		Summary:     payload.Summary,
		Days:        payload.Days,
	}, nil
}

func extractJSONObject(raw string) string {
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
