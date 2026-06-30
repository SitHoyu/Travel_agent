# travel-agent

A Go-based travel planning agent prototype that generates structured itineraries from user requests.

The project is organized as a small multi-service workspace. It separates the travel planning orchestration logic from the LLM access layer, and combines LLM reasoning with local tools such as weather lookup, itinerary validation, geocoding, and hotel-area recommendation.

## What This Project Does

Given a travel request such as destination, dates, budget, number of travelers, preferences, and constraints, the system can:

- query destination weather
- generate a structured itinerary draft with day-by-day activities
- validate whether the generated draft matches key constraints
- recommend suitable hotel areas based on the itinerary distribution
- attach nearby hotel candidates around the recommended stay center

The current implementation already supports an end-to-end agent-style planning workflow through HTTP APIs.

## Workspace Layout

This repository uses a Go workspace:

- `services/plan-orchestrator`
  Main application entrypoint for travel planning workflows. It runs the controller loop, invokes tools, stores run results, and exposes the agent API.
- `services/llm-gateway`
  Isolated LLM access layer for prompt rendering, provider routing, and model invocation.
- `shared`
  Shared request/response contracts, error definitions, and utility helpers used across services.
- `prompts`
  Prompt templates used by the LLM gateway and the agent runtime.
- `doc`
  Daily implementation notes and progress summaries.

## Architecture

The system is intentionally split into two layers:

1. `plan-orchestrator`
   Owns the travel-planning business flow.
   It drives the runtime session, decides when to call tools, and assembles the final response.

2. `llm-gateway`
   Owns prompt loading and model access.
   It hides provider details from the orchestrator and supports multiple LLM backends.

This separation keeps orchestration logic independent from provider-specific code and makes it easier to evolve prompts and model routing without changing the planning workflow.

## Core Planning Flow

The main planning flow is implemented as a staged agent loop.

1. Receive a `GeneratePlanRequest`
2. Query weather for the destination
3. Build a structured itinerary draft through the LLM gateway
4. Validate the generated itinerary against key constraints
5. Recommend hotel stay areas based on the validated itinerary
6. Return the final answer together with structured plan data and tool traces

The controller currently enforces this order so the agent does not skip important steps such as weather adaptation or validation.

## Main Capabilities

### 1. Structured itinerary generation

The planner does not only return free-form text. It produces structured data including:

- plan title
- destination
- summary
- per-day itinerary
- per-activity fields such as time slot, type, indoor/outdoor, and optional coordinates

This makes the output easier to validate, enrich, store, and render later in a UI.

### 2. Weather-aware planning

The orchestrator can call AMap weather APIs and include a weather summary in the planning prompt.
This allows the itinerary to adapt to forecast conditions such as rain.

### 3. Constraint validation

After draft generation, the system validates the result against several practical checks, including:

- destination consistency
- daily activity count
- budget consistency
- weather adaptation

If validation fails, the runtime can trigger one repair pass to regenerate the draft with revision feedback.

### 4. Hotel area recommendation

Based on the generated itinerary, the system recommends 2-3 stay areas and explains:

- why each area fits
- rough nightly price range
- pros and cons
- suitable travel styles

It can also compute a recommended stay center coordinate and query nearby hotel candidates from AMap.

### 5. Tool execution tracing

The final response includes tool execution metadata such as:

- executed tool names
- validation summary
- summarized tool outputs

This improves debuggability during prompt and workflow iteration.

## Current Local Tools

The orchestrator currently registers these local tools:

- `think`
  A lightweight reasoning helper tool used by the staged runtime.
- `query_weather`
  Queries AMap 3-day weather forecast for the requested city.
- `build_itinerary_draft`
  Calls the LLM gateway to generate a structured itinerary draft and then enriches activity locations when possible.
- `validate_constraints`
  Validates the generated draft and structured plan.
- `recommend_hotel_area`
  Produces hotel-area recommendations and nearby hotel candidates.

## LLM Gateway Responsibilities

The `llm-gateway` service is responsible for:

- loading prompt templates from `prompts/`
- rendering prompt variables
- routing requests to a configured provider
- exposing generic and travel-specific generation APIs

The current provider registry supports:

- `openai-compatible`
- `ollama-native`

## HTTP APIs

### `plan-orchestrator`

- `GET /healthz`
- `POST /v1/agent/plan/run`

`POST /v1/agent/plan/run` accepts a structured travel request and returns:

- final answer text
- structured plan
- hotel area recommendations
- tool execution summary

### `llm-gateway`

- `GET /healthz`
- `POST /v1/generate`
- `POST /v1/travel/plan/generate`
- `POST /v1/travel/plan/revise`

## Example Request Shape

The main planning API accepts a payload shaped like:

```json
{
  "request_id": "req-001",
  "destination": "Hangzhou",
  "start_date": "2026-07-10",
  "end_date": "2026-07-12",
  "budget": "3000 RMB",
  "travelers": 2,
  "preferences": ["food", "slow travel"],
  "constraints": ["at most 2 attractions per day"]
}
```

The returned result includes a structured `plan`, a Chinese `final_answer`, and hotel-area recommendation data.

## Configuration

Each service currently uses a local `config.yaml`.

Important configuration areas include:

- server ports
- LLM gateway base URL
- default LLM provider and model
- AMap API settings
- prompt base directory
- controller max steps

Current defaults in the repository use:

- `plan-orchestrator` on port `8080`
- `llm-gateway` on port `8081`
- AMap APIs for weather, geocoding, and hotel search
- Ollama as the default planning model backend

## Running Locally

### Option 1: with Docker Compose

```bash
docker-compose up --build
```

### Option 2: run services separately

In separate terminals:

```bash
cd services/llm-gateway
go run ./cmd/main.go
```

```bash
cd services/plan-orchestrator
go run ./cmd/main.go
```

Make sure the required provider and API configuration is available before testing end-to-end flows.

## Development Status

This repository is beyond an empty scaffold and already contains a working MVP-style backend flow:

- shared contracts are defined
- the orchestrator/controller flow is implemented
- prompt rendering and provider adapters exist
- the HTTP APIs are wired
- in-memory result storage is available
- weather lookup, plan validation, and hotel recommendation tools are connected end-to-end

Recent progress has also added:

- recommended stay center coordinates
- nearby hotel candidate lookup
- nearby hotel lookup error visibility in API responses

## Current Limitations

The project is still an early-stage prototype, and some areas are intentionally simple:

- storage is currently in-memory only
- there is no frontend yet
- validation rules are useful but still heuristic
- itinerary ordering and route optimization are not yet implemented
- some Chinese text in source files shows encoding issues and should be cleaned up

## Design Principles

- Keep orchestration and LLM access separated
- Keep storage inside `plan-orchestrator` first, then evolve later
- Keep shared contracts minimal to avoid cross-service coupling
- Favor structured plan data instead of free-form text
- Make the workflow observable through tool traces and validation summaries

## Suggested Next Steps

Likely next iterations for the project:

1. replace in-memory storage with SQLite or another persistent store
2. improve hotel-area naming and ranking quality
3. add itinerary ordering and route optimization
4. clean up prompt/tool text encoding issues
5. add a simple frontend or API consumer for interactive testing
