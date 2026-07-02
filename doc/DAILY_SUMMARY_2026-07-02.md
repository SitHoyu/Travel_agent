# Daily Summary - 2026-07-02

## Today

- Completed the real frontend plan detail page rendering.
- Connected `GET /v1/plans/:id` on the frontend detail page.
- Added detail-page rendering for:
  - plan summary
  - final answer
  - validation summary
  - hotel area recommendations
  - nearby hotel list
  - hotel image rendering from returned `photo_url`
  - day-by-day itinerary
  - activity metadata such as time, type, indoor/outdoor, and coordinates

- Added AMap JS API basic integration on the frontend.
- Created reusable `MapContainer.vue`.
- Confirmed the map can render successfully with:
  - environment variables from root `.env`
  - AMap loader
  - multiple marker instances

- Moved the temporary map test area out of the plans list page.
- Migrated the map component into the plan detail page.
- Connected real coordinate points from backend detail data into the map:
  - `recommended_center`
  - `nearby_hotels[].location`
  - activity coordinates from `plan.days[].activities[]`

## Issues And Fixes

- Problem: frontend dev server failed with `window is not defined`.
  - Cause: `@amap/amap-jsapi-loader` had been imported inside `vite.config.js`, which runs in Node rather than browser context.
  - Fix: removed AMap loader imports from `vite.config.js` and kept loader usage inside Vue components only.

- Problem: frontend could not read `VITE_AMAP_KEY` and `VITE_AMAP_SECRET` from the project root `.env`.
  - Cause: Vite by default reads `.env` from the frontend project directory, not the repository root.
  - Fix: added `envDir: ".."` in `frontend/vite.config.js`.

- Problem: needed to verify whether AMap markers could be added correctly before wiring real data.
  - Fix: first tested with fixed sample markers, then replaced them with real backend coordinates.

## Current Status

- Frontend main flow is now functionally complete for the current stage:
  - register
  - login
  - generate plan
  - confirm save
  - view saved plan list
  - view saved plan detail

- Plan detail page now has useful structured visualization even before advanced map styling:
  - text summary
  - hotel cards with images
  - itinerary breakdown
  - map with actual points

- The end-to-end path has been tested successfully.

## What Was Verified Today

- AMap can be loaded successfully in the frontend.
- Multiple markers can be rendered on the map.
- Real plan detail coordinates can be displayed on the map.
- Hotel images from backend URLs can be rendered.
- Full frontend flow from login to detail display has been tested successfully.

## Next Steps

- Map marker optimization
  - distinguish hotel markers and activity markers visually
  - give recommended center its own marker style
  - add marker click info windows
  - optionally group or filter markers by day

- Backend simple concurrency
  - evaluate multiple simultaneous plan generation requests
  - observe whether bottlenecks are in:
    - `plan-orchestrator`
    - `llm-gateway`
    - remote Ollama
    - MySQL

- Async logging / trace enhancements
  - add lightweight background logging for non-critical tracing
  - keep core persistence synchronous, but move auxiliary observability data to a looser path if useful

- Future incremental options
  - simple worker-pool style task execution
  - lightweight task/job model before introducing any real MQ
  - richer detail-page interaction such as marker category toggle and day-based filtering
