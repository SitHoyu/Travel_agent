package local

import (
	"context"

	"github.com/travel-agent/services/plan-orchestrator/internal/domain"
)

type ThinkTool struct{}

func NewThinkTool() *ThinkTool {
	return &ThinkTool{}
}

func (t *ThinkTool) Name() string {
	return "think"
}

func (t *ThinkTool) Description() string {
	return "Record a short planning thought in Chinese for the current step."
}

func (t *ThinkTool) Execute(_ context.Context, args map[string]interface{}) (domain.ToolExecution, error) {
	thought, _ := args["thought"].(string)
	return domain.ToolExecution{
		Success: true,
		Output:  thought,
	}, nil
}
