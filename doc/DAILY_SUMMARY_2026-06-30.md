# Daily Summary - 2026-06-30

## Today

- Completed the first full user-and-storage iteration inside `plan-orchestrator`.
- Split plan generation and persistence into two separate steps:
  - `POST /v1/agent/plan/run` now generates a plan only
  - `POST /v1/plans` now saves a user-confirmed plan
- Added MySQL persistence for `plans`.
- Added user registration and login support backed by the `users` table.
- Added JWT-based authentication middleware.
- Added `GET /v1/users/me`.
- Added `GET /v1/plans`.
- Added `GET /v1/plans/:id`.
- Added `.env` loading and environment-variable expansion for `config.yaml`.

## Issues And Fixes

- Problem: plan generation and persistence were coupled together in the original orchestrator flow.
  - Fix: split the service into generation-only and save-only entry points so future "generate again / confirm later" flows stay possible.

- Problem: `config.yaml` used `${AMAP_API_KEY}` and `${MYSQL_DSN}`, but the service did not actually expand environment variables.
  - Fix: added lightweight `.env` loading plus `os.ExpandEnv(...)` during config loading.

- Problem: early plan list/detail APIs temporarily relied on `user_id` query parameters.
  - Fix: replaced that path with JWT auth middleware and current-user context.

- Problem: local Go build hit a Windows permission issue on build cache once during validation.
  - Fix: reran build with the required permission and confirmed the service compiles successfully.

## Current Status

- User registration, login, current-user query, plan generation, plan save, plan list, and plan detail are all connected end-to-end.
- MySQL records for both `users` and `plans` are now being written successfully.
- Basic backend scope for user + plan persistence can be considered complete for this iteration.

## Test Commands

### 1. PowerShell UTF-8 setup

```powershell
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8
```

### 2. Register

```powershell
$registerJson = @{
  username = "testuser1"
  password = "Test123456"
  nickname = "测试用户1"
} | ConvertTo-Json -Depth 5

$registerBytes = [System.Text.Encoding]::UTF8.GetBytes($registerJson)

$registerResp = Invoke-RestMethod `
  -Method Post `
  -Uri http://localhost:8080/v1/auth/register `
  -ContentType "application/json; charset=utf-8" `
  -Body $registerBytes

$registerResp | ConvertTo-Json -Depth 10
```

### 3. Login

```powershell
$loginJson = @{
  username = "testuser1"
  password = "Test123456"
} | ConvertTo-Json -Depth 5

$loginBytes = [System.Text.Encoding]::UTF8.GetBytes($loginJson)

$loginResp = Invoke-RestMethod `
  -Method Post `
  -Uri http://localhost:8080/v1/auth/login `
  -ContentType "application/json; charset=utf-8" `
  -Body $loginBytes

$loginResp | ConvertTo-Json -Depth 10

$token = $loginResp.access_token
$authHeader = @{ Authorization = "Bearer $token" }
```

### 4. Get current user

```powershell
$meResp = Invoke-RestMethod `
  -Method Get `
  -Uri http://localhost:8080/v1/users/me `
  -Headers $authHeader

$meResp | ConvertTo-Json -Depth 10
```

### 5. Generate plan

```powershell
$generateJson = @{
  request_id = "agent-test-001"
  destination = "揭阳"
  start_date = "2026-06-30"
  end_date = "2026-07-02"
  budget = "3000 RMB"
  travelers = 2
  preferences = @("慢节奏","美食","古城")
  constraints = @("不赶行程","每天最多两个大景点")
} | ConvertTo-Json -Depth 5

$generateBytes = [System.Text.Encoding]::UTF8.GetBytes($generateJson)

$generateResp = Invoke-RestMethod `
  -Method Post `
  -Uri http://localhost:8080/v1/agent/plan/run `
  -Headers $authHeader `
  -ContentType "application/json; charset=utf-8" `
  -Body $generateBytes

$generateResp | ConvertTo-Json -Depth 15
```

### 6. Save plan

```powershell
$savePayload = @{
  user_id = 0
  request = @{
    request_id = "agent-test-001"
    destination = "揭阳"
    start_date = "2026-06-30"
    end_date = "2026-07-02"
    budget = "3000 RMB"
    travelers = 2
    preferences = @("慢节奏","美食","古城")
    constraints = @("不赶行程","每天最多两个大景点")
  }
  result = $generateResp
} | ConvertTo-Json -Depth 30

$saveBytes = [System.Text.Encoding]::UTF8.GetBytes($savePayload)

$saveResp = Invoke-RestMethod `
  -Method Post `
  -Uri http://localhost:8080/v1/plans `
  -Headers $authHeader `
  -ContentType "application/json; charset=utf-8" `
  -Body $saveBytes

$saveResp | ConvertTo-Json -Depth 20

$planId = $saveResp.id
```

### 7. List current user's plans

```powershell
$listResp = Invoke-RestMethod `
  -Method Get `
  -Uri "http://localhost:8080/v1/plans?page=1&page_size=20" `
  -Headers $authHeader

$listResp | ConvertTo-Json -Depth 15
```

### 8. Get plan detail

```powershell
$detailResp = Invoke-RestMethod `
  -Method Get `
  -Uri "http://localhost:8080/v1/plans/$planId" `
  -Headers $authHeader

$detailResp | ConvertTo-Json -Depth 20
```

## Next Steps

- Frontend
  - Build a simple Vue frontend for:
    - register/login
    - generate plan
    - confirm and save plan
    - plan list
    - plan detail

- Simple concurrency
  - After the frontend is usable, test multiple users submitting plan-generation requests at the same time.
  - Observe where the real bottleneck is first:
    - `plan-orchestrator`
    - `llm-gateway`
    - external APIs
    - MySQL

- Simple task queue
  - If synchronous generation becomes slow under concurrent use, consider a lightweight task/job model first instead of introducing a full MQ immediately.
  - A practical next step could be:
    - `plan_jobs` table
    - worker pool
    - task status polling

- Possible Go concurrency directions
  - worker pool for async plan generation
  - background goroutines for non-critical logging/trace/statistics
  - request timeout and cancellation improvements with `context.Context`
  - later, selective parallel tool execution if future tools become independent enough
