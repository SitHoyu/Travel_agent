package local

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/travel-agent/services/plan-orchestrator/internal/client/amapgeo"
	"github.com/travel-agent/shared/contracts"
)

type LocationEnricher struct {
	client *amapgeo.Client
}

func NewLocationEnricher(client *amapgeo.Client) *LocationEnricher {
	return &LocationEnricher{client: client}
}

func (e *LocationEnricher) EnrichPlan(ctx context.Context, destination string, plan *contracts.Plan) (int, error) {
	if e == nil || e.client == nil || plan == nil {
		return 0, nil
	}

	enriched := 0
	for dayIndex := range plan.Days {
		for activityIndex := range plan.Days[dayIndex].Activities {
			ok, err := e.enrichActivity(ctx, destination, &plan.Days[dayIndex].Activities[activityIndex])
			if err != nil {
				return enriched, err
			}
			if ok {
				enriched++
			}
		}
	}

	return enriched, nil
}

func (e *LocationEnricher) enrichActivity(ctx context.Context, destination string, activity *contracts.Activity) (bool, error) {
	if activity == nil {
		return false, nil
	}

	address := firstNonEmpty(activity.Location, activity.Name)
	if strings.TrimSpace(address) == "" {
		return false, nil
	}

	resp, err := e.client.Geocode(ctx, destination, address)
	if err != nil {
		return false, nil
	}
	if len(resp.Geocodes) == 0 {
		return false, nil
	}

	best := resp.Geocodes[0]
	lng, lat, err := parseLocation(best.Location)
	if err != nil {
		return false, fmt.Errorf("parse geocode location for %s: %w", address, err)
	}

	activity.ResolvedAddress = best.FormattedAddress
	activity.Longitude = lng
	activity.Latitude = lat
	activity.Adcode = best.Adcode
	activity.GeoLevel = best.Level
	return true, nil
}

func parseLocation(location string) (float64, float64, error) {
	parts := strings.Split(strings.TrimSpace(location), ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid location %q", location)
	}

	lng, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parse longitude: %w", err)
	}
	lat, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parse latitude: %w", err)
	}
	return lng, lat, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
