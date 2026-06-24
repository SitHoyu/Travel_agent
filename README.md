# travel-agent

First-phase project layout for a simple travel planning agent in Go.

## Services

- `services/plan-orchestrator`: main entrypoint for travel planning workflows
- `services/llm-gateway`: isolated LLM access layer for prompts, provider routing, retries, and response shaping

## Phase 1 Principles

- Keep orchestration and LLM access separated
- Keep storage inside `plan-orchestrator` as an internal repository first
- Keep shared contracts minimal to avoid cross-service coupling
- Favor structured plan data instead of free-form text

## Next Implementation Order

1. Define request/response contracts in `shared/contracts`
2. Implement orchestrator use case flow
3. Implement in-memory or SQLite repository in `plan-orchestrator`
4. Implement prompt loading and provider adapter in `llm-gateway`
5. Add end-to-end local integration with `docker-compose.yml`
