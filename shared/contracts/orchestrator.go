package contracts

type AgentPlanResponse struct {
	SessionID         string   `json:"session_id"`
	RequestID         string   `json:"request_id"`
	Status            string   `json:"status"`
	FinalAnswer       string   `json:"final_answer"`
	Plan              Plan     `json:"plan"`
	ToolRuns          int      `json:"tool_runs"`
	MessageCount      int      `json:"message_count"`
	ExecutedTools     []string `json:"executed_tools,omitempty"`
	ValidationSummary string   `json:"validation_summary,omitempty"`
}
