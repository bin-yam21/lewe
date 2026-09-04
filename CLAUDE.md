# CLAUDE.md — Lewe Project Memory

Working memory for future sessions. Read this first, then `spec/roadmap.md` for
what comes next and `spec/implementation_log.md` for the per-phase build narrative.

**This repo now holds two projects**: the Go API at the root, and the React
Native app in `mobile/`. See "The Mobile App" near the bottom before touching it.

## What Lewe Is

A barter/exchange marketplace REST API written in Go. Users list items, declare what
categories they *want*, the system finds mutual (bidirectional) matches, both sides
accept, confirm the exchange happened, then rate each other.

Module: `github.com/yeabt/lewe` · Go 1.26.5 · PostgreSQL

## Current State (as of 2026-09-04)

- **Phase 0 (Foundation)** and **Phase 1 (Core Exchange Loop)** are complete.
- `go build ./...` and `go vet ./...` both pass clean.
- Everything except Phase 0 is **uncommitted** — the repo has a single commit
  (`345bf8d chore: initialize empty main branch`) and the entire codebase sits in the
  working tree. Committing the work is the first thing worth doing.
- No automated tests exist. Manual verification is via `Lewe_API.postman_collection.json`
  (40 requests covering the full two-user flow: register A+B → list items → find match →
  both accept → both complete → both rate).

## Architecture

Hand-rolled layering, no web framework. Each domain package is self-contained:

```
handler.go     → HTTP: decode, call service, map domain errors to status codes
service.go     → validation + business rules + DTO mapping
repository.go   → pgx queries, row structs, domain error values
types.go       → request/response DTOs
```

```
cmd/api/main.go            Entry point: config → migrations → pool → DI wiring → server + graceful shutdown
internal/auth/jwt.go       HS256 access tokens (15m) + opaque random refresh tokens (7d)
internal/config/config.go  Env config; fatals if DATABASE_URL or JWT_SECRET missing
internal/db/db.go          pgxpool (MaxConns 25, MinConns 5, 5m lifetime)
internal/db/migrate.go     golang-migrate, runs Up() on every boot
internal/db/migrations/    000001 users, 000002 refresh_tokens, 000003 items,
                           000004 wants, 000005 matches, 000006 ratings
internal/db/queries/       sqlc .sql files — DEFINED BUT NOT GENERATED / NOT USED
internal/middleware/       Auth (JWT → pgtype.UUID in context), Logging (method/path/status/duration)
internal/response/json.go  JSON() + Error() helpers, {"error","message"} shape
internal/router/router.go  Go 1.22+ stdlib ServeMux patterns ("POST /api/v1/..."), auth applied per-route
internal/validator/         Errors map[string]string implementing error; Validate{Email,Required,MinLength,MaxLength,OneOf,Range}
internal/users/            + token_repository.go for refresh tokens
internal/items/            items + wants (wants are owned by an item, replaced wholesale on update)
internal/matching/         match discovery + state machine
internal/ratings/          rating creation + per-user AVG aggregate
```

Dependency direction: `ratings → matching → items`. `matching.Service` holds an
`*items.Repository` to sync item status; `ratings.Service` holds a `*matching.Service`
to verify participation and completion. Wiring lives in [main.go](cmd/api/main.go).

## Conventions (follow these when adding code)

- **No framework.** stdlib `net/http` + `ServeMux` method-and-path patterns only.
- **Concrete pointer types, not interfaces**, for repo/service dependencies. DI is manual in `main.go`.
- **pgtype everywhere at the DB boundary** (`pgtype.UUID`, `pgtype.Text`, `pgtype.Timestamptz`);
  convert to plain Go types only in the `to*Response` mapping functions.
- **Each package defines its own `uuidToString`** helper (duplicated in users/items/matching/ratings —
  deliberate; there is no shared util package yet).
- **Domain errors are package-level `errors.New` values** in `repository.go`; handlers map
  them with a `handleError` switch. Validation failures come back as `validator.Errors`
  and are detected with `errors.As` → **422** with a `fields` object.
- **Auth'd handlers** start with `middleware.UserIDFromContext(r.Context())`.
- **Path params** via `r.PathValue("id")` scanned into `pgtype.UUID`.
- Status codes in use: 200, 201, 400 (bad body/ID), 401, 403 (not owner/participant),
  404, 409 (conflict: email taken, already rated, match not completed), 422 (validation), 500.

## Domain Rules Worth Remembering

**Item status**: `active → matched → exchanged`, plus `archived` (soft delete — items are
never hard-deleted, to preserve match/rating integrity). Status is driven by the match
lifecycle, not set directly by users.

**Match state machine** (`internal/matching/service.go`):
```
pending ──accept(A)──> accepted_a ──accept(B)──> confirmed ──both complete──> completed
        └─accept(B)──> accepted_b ──accept(A)──┘
any non-terminal state ──decline / cancel──> cancelled
```
- Both items → `matched` on `confirmed`; both → `exchanged` on `completed`;
  both restored to `active` on cancel from an accepted/confirmed state.
- `complete` only works from `confirmed`, and sets `confirmed_a`/`confirmed_b` per side;
  the match flips to `completed` only when both are true.

**Matching is on-demand**, not a background worker: `POST /items/{id}/matches` runs the
bidirectional SQL (our wants ∋ their category AND their wants ∋ our category, both active,
different users, no existing match) and inserts match rows, then returns newly created +
pre-existing non-cancelled matches for that item.

**Ratings** require a `completed` match, one per (match, rater) enforced by a UNIQUE
constraint *and* an application-level pre-check for a clean 409.

## Running It

```powershell
# .env (gitignored) supplies DATABASE_URL, JWT_SECRET, PORT
go run ./cmd/api          # migrations run automatically on boot
```
Migrations resolve from `file://internal/db/migrations`, so **run from the repo root**.
Health check: `GET /health`.

## Known Issues / Debt

**Fixed 2026-09-04** (see the Phase 1.1 section of the implementation log):
`itoa` corruption, dead sha256 placeholder, stale go.mod deps, unwritten
`matches.exchange_method`, unreachable `GetMatchRatings`, archived-item exposure via
`?status=archived`, the silently-dropped owner filter in `Repository.List`, the
migrate driver scheme that stopped the server booting at all, and stale item statuses
in the match write-endpoint responses.

**Two lessons from that pass**, both worth keeping in mind:
- `go build` + `go vet` passing says nothing about whether the thing runs. The migrate
  scheme bug sat undetected through two "verified" phases. **Boot it against Postgres.**
- Any handler that mutates related rows must re-read before responding
  (`matching.Service.freshResponse`) — pre-transition snapshots go stale silently.

Still open:

1. **N+1 on item listing** — `ListItems` fetches wants per row in a loop (noted as
   acceptable at current scale in the log). Fix with one `WHERE item_id = ANY($1)` query.
2. **sqlc is configured but unused** — `sqlc.yaml` + `internal/db/queries/{users,refresh_tokens}.sql`
   exist, nothing is generated, repositories are hand-written pgx. Either generate or drop.
3. **No tests at all**, and no CI.
4. **Exchange-method compatibility is unchecked** — a match can settle on `shipping` even if
   one of the two items advertises `in_person` only. Needs the items' `exchange_method`
   pulled into `MatchWithItems` to validate against.
5. **Match status races** — `RespondToMatch` reads the match, then writes, with no row lock
   or status guard in the UPDATE. Two simultaneous accepts can interleave. Consider
   `SELECT … FOR UPDATE` in a transaction, or a conditional `WHERE status = $expected`.
6. **`FindMatchesForItem` swallows errors** — per-match failures `continue` silently.

## Next Up — Phase 2 (Polish & Scale)

Image upload (S3/R2 — `images` is currently just a JSONB array of URLs), notifications,
full-text search, cursor pagination, admin endpoints.

## The Mobile App (`mobile/`)

Expo SDK 57 + TypeScript + expo-router. Routes live in **`mobile/src/app/`** (not
`app/` — the template puts them under `src`). Path alias `@/*` → `mobile/src/*`.

```powershell
go run ./cmd/api          # API on :8090 (from repo root, .env sets the port)
cd mobile; npx expo start # Metro on :8081, scan the QR with Expo Go
```

**Ports matter here.** 8080 is held by an unrelated `AgentService.exe` and 8081 is
Metro's own dev server, so the API runs on **8090**.

**How the app finds the API**: `src/api/config.ts` reads Metro's own host from
`Constants.expoConfig.hostUri` and points at `http://<that LAN IP>:8090`. A phone
cannot reach `localhost`, and hardcoding an IP breaks whenever the router
reassigns one. `EXPO_PUBLIC_API_URL` overrides it.

**Conventions**, mirroring the Go side's discipline:
- No color, font size, or spacing value is written inline. Everything comes from
  `useTheme()` over `src/theme/tokens.ts`. This is what keeps dark mode working.
- Text only ever renders through `components/Text.tsx` with a `variant` and a
  token `color`.
- Server state belongs to TanStack Query (`hooks/useItems.ts`); only ephemeral UI
  state uses `useState`.
- All requests go through `api/client.ts`. It attaches the token and, on a 401,
  refreshes **once** and replays. Never call `fetch` directly — the API rotates
  refresh tokens, so a second concurrent refresh logs the user out.
- API errors arrive as `ApiError` with `status` and, for 422s, a `fields` map that
  screens render per input.

**Keep `src/api/types.ts` in sync with the Go DTOs.** They are hand-mirrored; when
a JSON tag changes on the server, the app breaks silently at runtime.

## Where Things Are Documented

- `spec/roadmap.md` — the forward plan. Track A (API) and Track B (app), with the
  two points where they interlock.
- `spec/implementation_log.md` — phase-by-phase record with decision tables. **Append a new
  section here when a phase completes**; that is the established habit for this project.
- `Lewe_API.postman_collection.json` — the de-facto integration test suite. It does
  not yet cover `GET /matches/{id}/ratings` or the `exchange_method` field on respond.
