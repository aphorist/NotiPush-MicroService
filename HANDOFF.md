# HANDOFF

## Task: Read TPS from .env instead of hardcoded rate limit

- **Status:** Completed
- **What:** Replace the hardcoded worker throttle with `TPS` from environment configuration while keeping the previous 2 TPS behavior as the fallback.
- **Files changed:** `main.go`, `main_test.go`, `docker-compose.yml`, `README.md`, `.env`, `go.mod`, `go.sum`
- **Verification:** Ran `go test ./...` and `docker compose run --rm push-worker`; the container logged `Worker rate limit configured at 10 TPS`.
- **Notes:** The service now loads `.env` directly for local runs, and Docker Compose mounts `.env` into `/app/.env` so runtime TPS stays in sync with the file.

Timestamp: 2026-05-15

## Task: Show NotiPush server response in debug logs

- **Status:** Completed
- **What:** When `DEBUG=true`, the service now reads and logs the actual NotiPush HTTP response body.
- **File changed:** `main.go` (function `sendPushToRealServer`)
- **Verification:** Ran `go test ./...` — tests passed.
- **Notes:** Response body is logged at the `[DEBUG]` level; warnings include response body when `DEBUG=true` for troubleshooting.

Timestamp: 2026-05-14
