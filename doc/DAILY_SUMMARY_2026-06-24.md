# Daily Summary - 2026-06-24

## Today's Progress

### 1. Agent runtime multi-step chain is basically working

The current `plan-orchestrator` can now complete a real multi-step workflow instead of returning a single direct model answer:

- receive user travel request
- call `query_weather`
- call `build_itinerary_draft`
- attempt `validate_constraints`
- return final answer and plan response

### 2. AMap weather tool has been integrated

Implemented a real weather tool based on AMap:

- reads city `adcode` from `AMap_adcode_citycode.xlsx`
- maps city names such as `杭州` / `杭州市`
- calls AMap 3-day forecast API
- returns a summarized weather result for the agent/tool chain

### 3. Weather context is now fed into itinerary generation

The weather result is no longer isolated.
It is now injected into the itinerary draft generation request so the model can adjust:

- rainy day indoor activities
- outdoor activity timing
- pacing suggestions

### 4. Structured itinerary output has been introduced

The planner prompt was upgraded from free-form text to structured JSON output.

The generated plan now targets a schema with:

- `title`
- `destination`
- `summary`
- `days[]`
- `activities[]`

Each activity now supports fields such as:

- `name`
- `location`
- `time_slot`
- `type`
- `indoor`
- `description`

### 5. Constraint validation tool was added

Implemented a first version of `validate_constraints` with several practical rules:

- destination consistency
- budget limit check
- weather adaptation check

### 6. Better runtime observability

The final response now exposes more execution details:

- `executed_tools`
- `validation_summary`
- `tool_executions`

This made it much easier to understand which tool actually ran and where the chain failed or looped.

### 7. Basic stage guardrail logic was added

Added a lightweight guardrail in `LLMAgent` to reduce unstable tool loops:

- prevent repeated `build_itinerary_draft` after a draft already exists
- push the flow toward `validate_constraints`
- restrict later turns once validation exists


## Main Problems Found Today

### 1. Tool argument format from the model is unstable

The model sometimes returned:

- `request` as an object
- `request` as a JSON string

This caused decoding errors in local tools.

### 2. Structured plan generation is not fully stable yet

Although structured output has started to work, it is not consistently stable:

- sometimes the draft tool succeeds
- sometimes the model output does not match the expected JSON shape
- sometimes the runtime falls back to text-style summary behavior

### 3. Validation step still loops or misfires

`validate_constraints` was called repeatedly in some runs.
In some cases it failed because of incomplete arguments such as:

- missing `request`
- missing `draft`

### 4. Validation currently relies too much on summary text

Some validation failures are technically false positives because the validator is often checking summarized text instead of the full structured itinerary.

This is the main remaining weakness in the current chain.


## Improvements Made

### 1. Added robust request decoding for tools

Introduced a shared decoder so tools can handle both:

- object-form request arguments
- string-form JSON request arguments

This reduced a class of runtime failures caused by model output variation.

### 2. Improved final response debugging capability

Added execution trace information so debugging no longer depends only on `tool_runs`.

### 3. Tightened the agent decision prompt

The prompt was changed from soft suggestions to stage-based guidance:

- weather first
- then draft
- then validation
- then final answer

### 4. Connected weather-aware planning to structured output

The planner now receives weather summary explicitly and is instructed to adapt indoor/outdoor activities accordingly.


## Current Status

The project is now beyond the "can it run" phase.
It already demonstrates:

- multi-step agent orchestration
- external tool integration
- structured travel plan generation
- runtime observability
- first-pass validation loop

The main remaining engineering work is to make the structured draft and validation stages more deterministic.


## TODO for Tomorrow

### High Priority

1. Make `validate_constraints` consume the structured `plan` instead of only checking text summary.
2. Ensure validation tool calls are always enriched with complete arguments:
   - `request`
   - `draft`
   - `weather_summary`
   - ideally structured `plan`
3. Prevent repeated `validate_constraints` calls after one successful validation.

### Medium Priority

4. Improve structured itinerary parsing tolerance:
   - better JSON extraction
   - fallback handling when model output is partially malformed
5. Verify that `plan.days` is consistently populated across repeated runs.

### Next Feature After Stabilization

6. Implement the final external tool: location name to coordinates.
7. Attach coordinates to itinerary activities for simple frontend rendering.

### Later Work

8. Add persistence layer design after itinerary schema stabilizes.
9. Build a minimal usable frontend.
10. Re-evaluate whether goroutines or queue-based async execution are actually needed.


## Suggested Commit Theme

If you want to summarize today's work in one sentence:

> Added weather-aware multi-step travel planning, structured itinerary output, constraint validation, and runtime execution observability for the Go travel agent.
