package domain

import (
	"time"

	"github.com/travel-agent/shared/contracts"
)

// PlanRecord is the persisted form of a user-confirmed plan.
type PlanRecord struct {
	ID                 int64
	UserID             int64
	Request            contracts.GeneratePlanRequest
	RequestID          string
	SessionID          string
	Status             string
	Title              string
	Destination        string
	Summary            string
	FinalAnswer        string
	ValidationSummary  string
	Plan               contracts.Plan
	HotelAreas         contracts.HotelAreaRecommendationResult
	ExecutedTools      []string
	ToolExecutions     []ToolTrace
	RequestPayloadJSON string
	PlanJSON           string
	HotelAreasJSON     string
	ExecutedToolsJSON  string
	ToolExecutionsJSON string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
