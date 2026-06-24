package toolkit

import (
	"context"
	"fmt"
	"sort"

	"github.com/travel-agent/services/plan-orchestrator/internal/domain"
)

type Tool interface {
	Name() string
	Description() string
	Execute(context.Context, map[string]interface{}) (domain.ToolExecution, error)
}

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

func (r *Registry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

func (r *Registry) List() []Tool {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)

	tools := make([]Tool, 0, len(names))
	for _, name := range names {
		tools = append(tools, r.tools[name])
	}
	return tools
}

func (r *Registry) Execute(ctx context.Context, call domain.ToolCall) (domain.ToolExecution, error) {
	tool, ok := r.tools[call.Name]
	if !ok {
		return domain.ToolExecution{}, fmt.Errorf("tool %s not found", call.Name)
	}

	execution, err := tool.Execute(ctx, call.Arguments)
	execution.ToolCallID = call.ID
	execution.Name = call.Name
	return execution, err
}
