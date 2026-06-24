package agent

import (
	"context"

	"github.com/travel-agent/services/plan-orchestrator/internal/domain"
)

type Agent interface {
	Think(context.Context, *domain.Session) (domain.Thought, error)
}
