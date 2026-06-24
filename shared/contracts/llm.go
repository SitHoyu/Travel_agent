package contracts

type LLMGenerateRequest struct {
	RequestID  string                 `json:"request_id"`
	Provider   string                 `json:"provider"`
	Model      string                 `json:"model"`
	Template   string                 `json:"template"`
	Variables  map[string]any         `json:"variables"`
	System     string                 `json:"system"`
	Temperature float64               `json:"temperature"`
	MaxTokens  int                    `json:"max_tokens"`
}

type LLMGenerateResponse struct {
	RequestID string         `json:"request_id"`
	Provider  string         `json:"provider"`
	Model     string         `json:"model"`
	Prompt    string         `json:"prompt"`
	Content   string         `json:"content"`
	Usage     LLMUsage       `json:"usage"`
	LatencyMs int64          `json:"latency_ms"`
	Raw       map[string]any `json:"raw,omitempty"`
}

type LLMUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
