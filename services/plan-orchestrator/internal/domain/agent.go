package domain

import "time"

type Message struct {
	Role       string                 `json:"role"`
	Content    string                 `json:"content"`
	ToolCalls  []ToolCall             `json:"tool_calls,omitempty"`
	ToolCallID string                 `json:"tool_call_id,omitempty"`
	Meta       map[string]interface{} `json:"meta,omitempty"`
}

type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type ToolExecution struct {
	ToolCallID string                 `json:"tool_call_id"`
	Name       string                 `json:"name"`
	Success    bool                   `json:"success"`
	Output     string                 `json:"output"`
	Meta       map[string]interface{} `json:"meta,omitempty"`
}

type Thought struct {
	Text      string     `json:"text"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Done      bool       `json:"done"`
}

type Session struct {
	ID          string           `json:"id"`
	RequestID   string           `json:"request_id"`
	Status      string           `json:"status"`
	RequestText string           `json:"request_text"`
	Messages    []Message        `json:"messages"`
	Executions  []ToolExecution  `json:"executions"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}
