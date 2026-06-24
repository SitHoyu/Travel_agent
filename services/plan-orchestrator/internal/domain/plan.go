package domain

import "github.com/travel-agent/shared/contracts"

type PlanRunResult struct {
	SessionID         string         `json:"session_id"`
	RequestID         string         `json:"request_id"`
	Status            string         `json:"status"`
	FinalAnswer       string         `json:"final_answer"`
	Plan              contracts.Plan `json:"plan"`
	ToolRuns          int            `json:"tool_runs"`
	MessageCount      int            `json:"message_count"`
	ExecutedTools     []string       `json:"executed_tools,omitempty"`
	ValidationSummary string         `json:"validation_summary,omitempty"`
}
