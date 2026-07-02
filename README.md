# travel-agent

A Go-based travel planning agent prototype with a Vue frontend.

The project is organized as a small multi-service workspace. It separates travel-planning orchestration from the LLM access layer, and combines LLM reasoning with local tools such as weather lookup, itinerary validation, geocoding, hotel-area recommendation, nearby hotel lookup, and coordinate-based map display.

## What This Project Does

Given a travel request such as destination, dates, budget, traveler count, preferences, and constraints, the system can:

- query destination weather
- generate a structured itinerary draft
- validate whether the draft matches key constraints
- recommend suitable hotel areas
- attach nearby hotel candidates around a recommended stay center
- let a logged-in user confirm and save a generated plan
- list and view saved plans by current user
- render saved plan details in a frontend UI, including hotel images and map points

The current implementation already supports an end-to-end workflow across backend and frontend:

1. register / login
2. generate plan
3. confirm save
4. view plan list
5. view plan detail

## Workspace Layout

This repository uses a Go workspace plus a frontend app:

- `services/plan-orchestrator`
  Main application entrypoint for user auth, travel-planning workflows, plan persistence, and agent APIs.
- `services/llm-gateway`
  Isolated LLM access layer for prompt rendering, provider routing, and model invocation.
- `shared`
  Shared request/response contracts and utilities used across services.
- `frontend`
  Vue 3 + Vite frontend for auth, plan generation, save flow, plan list, and plan detail visualization.
- `prompts`
  Prompt templates used by the LLM gateway and the agent runtime.
- `doc`
  Daily implementation notes and progress summaries.

## Architecture

The system is intentionally split into two backend services plus a separate frontend:

1. `plan-orchestrator`
   Owns the travel-planning business flow, user auth, plan persistence, and HTTP APIs.

2. `llm-gateway`
   Owns prompt loading and model access.

3. `frontend`
   Owns user interaction, plan generation UI, save confirmation flow, plan browsing, and detail rendering.

This keeps orchestration logic independent from provider-specific code and keeps frontend iteration separate from backend service concerns.

## Core Backend Flow

The main planning flow is implemented as a staged agent loop:

1. Receive a `GeneratePlanRequest`
2. Query weather for the destination
3. Build a structured itinerary draft through the LLM gateway
4. Validate the generated itinerary against key constraints
5. Recommend hotel stay areas based on the validated itinerary
6. Return the final answer together with structured plan data and tool traces

The controller enforces this order so the agent does not skip weather, validation, or hotel recommendation stages.

## Plan Lifecycle

The project now uses a two-step plan flow:

1. Generate plan
   `POST /v1/agent/plan/run`
   Returns a generated result but does not persist it.

2. Confirm and save plan
   `POST /v1/plans`
   Persists a user-confirmed plan into MySQL.

This design leaves room for later flows such as:

- generate again
- compare multiple generated plans
- save only the chosen result

## Main Capabilities

### 1. Structured itinerary generation

The planner returns structured data, not only free-form text. Output includes:

- plan title
- destination
- summary
- per-day itinerary
- per-activity metadata such as time slot, type, indoor/outdoor, and optional coordinates

### 2. Weather-aware planning

The orchestrator calls AMap weather APIs and includes a weather summary in the planning prompt so the itinerary can adapt to forecast conditions such as rain.

### 3. Constraint validation

After draft generation, the system validates the result against several practical checks:

- destination consistency
- daily activity count
- budget consistency
- weather adaptation

If validation fails, the runtime can trigger a limited repair pass.

### 4. Hotel area recommendation

Based on the generated itinerary, the system recommends stay areas and explains:

- why each area fits
- rough nightly price range
- pros and cons
- suitable travel styles

It can also compute a recommended stay center coordinate and query nearby hotel candidates from AMap.

### 5. User auth and user-owned plans

The project now supports:

- user registration
- user login
- current-user query
- saving plans under a specific user
- listing the current user’s saved plans
- reading one saved plan in detail

### 6. Frontend visualization

The frontend now supports:

- register / login
- protected routes
- plan generation form
- save confirmation
- plans list page
- plan detail page
- hotel image rendering from returned URLs
- AMap-based point rendering for:
  - plan activities with coordinates
  - recommended stay center
  - nearby hotel locations

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

## Storage

The project no longer uses in-memory storage as the main path for user plans.

Current persistence:

- MySQL `users` table
- MySQL `plans` table

Plan records store:

- ownership by user
- request/session identifiers
- structured plan JSON
- hotel recommendation JSON
- final answer
- validation summary
- tool trace summaries

## HTTP APIs

### `plan-orchestrator`

Public endpoints:

- `GET /healthz`
- `POST /v1/auth/register`
- `POST /v1/auth/login`

Protected endpoints:

- `GET /v1/users/me`
- `POST /v1/agent/plan/run`
- `POST /v1/plans`
- `GET /v1/plans`
- `GET /v1/plans/:id`

### `llm-gateway`

- `GET /healthz`
- `POST /v1/generate`
- `POST /v1/travel/plan/generate`
- `POST /v1/travel/plan/revise`

## Frontend Pages

The current frontend includes these main routes:

- `/login`
- `/register`
- `/planner`
- `/plans`
- `/plans/:id`

Current frontend stack:

- Vue 3
- Vite
- Vue Router
- Axios
- AMap JS API Loader

## Example Request Shape

The main generate API accepts a payload shaped like:

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

The generate response includes:

- `final_answer`
- `plan`
- `hotel_areas`
- `validation_summary`
- tool execution summary

## Configuration

Backend services use `config.yaml`, with environment-variable expansion supported via `.env`.

Important backend configuration areas include:

- server ports
- LLM gateway base URL
- default LLM provider and model
- AMap API settings
- JWT secret
- MySQL DSN
- prompt base directory
- controller max steps

Frontend uses Vite env variables:

- `VITE_API_BASE_URL`
- `VITE_AMAP_KEY`
- `VITE_AMAP_SECRET`

## Running Locally

### Backend

Run in separate terminals:

```bash
cd services/llm-gateway
go run ./cmd/main.go
```

```bash
cd services/plan-orchestrator
go run ./cmd/main.go
```

Default ports:

- `plan-orchestrator`: `8080`
- `llm-gateway`: `8081`

### Frontend

```bash
cd frontend
npm install
npm run dev
```

Default dev port:

- `frontend`: `5173`

### Environment Notes

Typical local setup includes:

- MySQL for `users` and `plans`
- AMap key and secret
- Ollama or another configured LLM provider

For the frontend, Vite is configured to read environment variables from the repository root as well.

## Development Status

This repository has moved beyond the initial backend scaffold and now contains a working MVP flow across backend and frontend:

- shared contracts are defined
- orchestrator/controller flow is implemented
- prompt rendering and provider adapters exist
- JWT-based user auth is implemented
- MySQL plan persistence is connected
- generate/save plan flow is decoupled
- frontend auth flow is implemented
- frontend generate/save/list/detail flows are implemented
- hotel images are rendered in detail view
- map points are rendered in detail view from real backend coordinates

## Current Limitations

The project is still an early-stage prototype, and some areas are intentionally simple:

- map marker styles are not yet differentiated between hotels, activities, and recommended center
- some Chinese text encoding issues still appear in parts of the codebase or terminal output
- itinerary ordering and route optimization are not yet implemented
- Ollama may be network-sensitive when deployed on another machine
- concurrency handling is still the default synchronous HTTP path
- async task queue / background worker model is not yet implemented

## Design Principles

- Keep orchestration and LLM access separated
- Keep user-owned plan persistence inside `plan-orchestrator`
- Keep shared contracts minimal to avoid cross-service coupling
- Favor structured plan data instead of free-form text
- Make the workflow observable through tool traces and validation summaries
- Keep the plan lifecycle explicitly split into generate first, save second

## Suggested Next Steps

Likely next iterations for the project:

1. optimize map marker styles and marker interaction
2. improve hotel-area naming and ranking quality
3. add itinerary ordering and route optimization
4. experiment with simple concurrency and worker-pool style task execution
5. add lightweight async logging / tracing improvements
6. consider a lightweight job/task model before introducing a full MQ
