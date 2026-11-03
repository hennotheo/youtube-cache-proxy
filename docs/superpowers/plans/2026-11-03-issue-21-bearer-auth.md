# Issue 21 — Appliquer les rôles Bearer

## Goal

Protect REST endpoints with the configured producer and admin tokens. Producer can add and read sources and retrieve audio. Admin can perform every API action. Keep `/health` public.

## Tasks

1. Add constant-time token matching and role-aware authorization middleware, returning 401 for missing/invalid credentials and 403 when a valid producer requests an admin operation.
2. Apply producer access to source creation/list/detail/audio and admin access to run/retry/delete/worker control/status routes.
3. Pass the configured credentials from `main.go` into the API.
4. Update README with the route permission matrix and remove the outdated claim that the worker is not exposed over HTTP.

## Constraints

- Never include credentials in responses, logs, or errors.
- Health remains unauthenticated.
- Follow the issue's role matrix and retain the existing REST response behavior.
- Branch: `feat/21-appliquer-roles-bearer-issue`; pull request targets `dev`.
