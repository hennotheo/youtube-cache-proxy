# Issue 20 — Exposer les routes REST

## Goal

Expose the eleven endpoints listed in the project brief using the existing SQLite store, downloader, and worker. Keep authentication, standardized request/error handling, advanced Range behavior, and the admin web session in their dedicated issues (#21, #22, #23, #25, #26).

## File map

- `internal/httpapi/router.go`: dependency wiring and route registration.
- `internal/httpapi/api.go`: narrow API dependencies and constructor.
- Focused handler files for sources, downloads, audio, worker controls, status, and shared JSON responses.
- `internal/store`: source/run operations already exist; add only persisted worker status queries needed by the REST contract.
- `cmd/server/main.go`: construct downloader/worker/API with runtime configuration.
- `README.md`: replace the “health-only” status with the implemented API surface and current limitations.

## Tasks

1. Add JSON DTOs and POST/GET source handlers, including pagination/filter query parameters and source detail.
2. Add run/retry/delete handlers and map store errors to HTTP status codes.
3. Add audio serving, with safe disposition filename and standard-library file serving.
4. Add pause/resume/status endpoints and persisted aggregate status queries.
5. Wire API, worker, downloader, and settings in `main.go`; keep `/health`.
6. Update README API status and documented limitations.

## Constraints

- No framework; standard `net/http` ServeMux.
- Keep one type per file, short handlers, and comments for exported elements.
- Do not claim auth, uniform error DTOs, CSRF, or admin session protection before their issues land.
- API run start remains asynchronous through `worker.Start`.
- Follow the user’s instruction to stop after this issue’s PR is ready for review.
