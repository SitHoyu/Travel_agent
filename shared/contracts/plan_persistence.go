package contracts

import "time"

// SavePlanRequest persists a generated plan after the user confirms it.
// UserID is temporary until auth middleware injects the current user.
type SavePlanRequest struct {
	UserID  int64               `json:"user_id"`
	Request GeneratePlanRequest `json:"request"`
	Result  AgentPlanResponse   `json:"result"`
}

type SavedPlanResponse struct {
	ID                int64                         `json:"id"`
	UserID            int64                         `json:"user_id"`
	RequestID         string                        `json:"request_id"`
	SessionID         string                        `json:"session_id"`
	Status            string                        `json:"status"`
	Title             string                        `json:"title"`
	Destination       string                        `json:"destination"`
	Summary           string                        `json:"summary"`
	FinalAnswer       string                        `json:"final_answer"`
	ValidationSummary string                        `json:"validation_summary,omitempty"`
	Plan              Plan                          `json:"plan"`
	HotelAreas        HotelAreaRecommendationResult `json:"hotel_areas"`
	ExecutedTools     []string                      `json:"executed_tools,omitempty"`
	ToolExecutions    []ToolTrace                   `json:"tool_executions,omitempty"`
	CreatedAt         time.Time                     `json:"created_at"`
	UpdatedAt         time.Time                     `json:"updated_at"`
}

type PlanListItem struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	RequestID   string    `json:"request_id"`
	SessionID   string    `json:"session_id"`
	Status      string    `json:"status"`
	Title       string    `json:"title"`
	Destination string    `json:"destination"`
	Summary     string    `json:"summary"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ListPlansResponse struct {
	Items    []PlanListItem `json:"items"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}
