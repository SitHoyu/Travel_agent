package domain

type AgentDecision struct {
	Thought     string             `json:"thought"`
	ToolCalls   []ToolCallDecision `json:"tool_calls"`
	FinalAnswer string             `json:"final_answer"`
	Done        bool               `json:"done"`
}

type ToolCallDecision struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}
