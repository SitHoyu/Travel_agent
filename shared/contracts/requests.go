package contracts

type GeneratePlanRequest struct {
	RequestID   string   `json:"request_id"`
	Destination string   `json:"destination"`
	StartDate   string   `json:"start_date"`
	EndDate     string   `json:"end_date"`
	Budget      string   `json:"budget"`
	Travelers   int      `json:"travelers"`
	Preferences []string `json:"preferences"`
	Constraints []string `json:"constraints"`
	WeatherSummary string `json:"weather_summary,omitempty"`
}

type GeneratePlanResponse struct {
	Plan       Plan   `json:"plan"`
	Model      string `json:"model"`
	RequestID  string `json:"request_id"`
	LatencyMs  int64  `json:"latency_ms"`
}

type RevisePlanRequest struct {
	PlanID    string   `json:"plan_id"`
	Feedback  string   `json:"feedback"`
	KeepItems []string `json:"keep_items"`
}
