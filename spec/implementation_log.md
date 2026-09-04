# Implementation Log

## Phase 0 — Foundation (2026-08-27)

### What Was Built

The complete project skeleton for Lewe, a barter/exchange marketplace API built in Go.

**Scope**: Auth system (register, login, JWT access/refresh tokens), user profile CRUD, database schema with migrations, middleware, and HTTP routing.

### Architecture

```
cmd/api/main.go           → Entry point, DI wiring, graceful shutdown
internal/auth/jwt.go       → JWT access token + opaque refresh token utilities
internal/config/config.go  → Env-based configuration
internal/db/db.go          → pgxpool connection
internal/db/migrate.go     → golang-migrate runner
internal/db/migrations/    → SQL migration files (users + refresh_tokens)
internal/db/queries/       → sqlc query definitions
internal/middleware/        → Auth (JWT validation) + request logging
internal/response/json.go  → Consistent JSON response helpers
internal/router/router.go  → Go 1.22+ stdlib routing with auth middleware
internal/users/            → Domain: handler → service → repository
internal/validator/         → Lightweight input validation
```

### Key Decisions

| Decision | Rationale |
|----------|-----------|
| **pgx v5** over `lib/pq` | Faster, more idiomatic Go driver, native pgtype support for nullable fields |
| **Opaque refresh tokens** (not JWT) | Simpler revocation — stored hashed (SHA-256) in DB; JWT refresh tokens can't be revoked without DB lookup anyway |
| **Token rotation on refresh** | Old refresh token is revoked when a new pair is issued — limits window if a refresh token is compromised |
| **Manual repository** (not sqlc-generated) | sqlc queries are defined in `internal/db/queries/` for later generation, but the repository layer wraps pgx directly for now to keep the learning experience explicit |
| **Go 1.22+ stdlib routing** | No `chi` or `gin` — matches the spec's goal of learning Go idioms, not framework abstractions |
| **Domain-grouped structure** | Per spec §6 — each domain (users, items, matching) is self-contained with handler/service/repository |
| **`validator.Errors` as `error` type** | Validation failures are a typed error that handlers can detect with `errors.As` and return as 422 with per-field messages |

### API Routes

| Method | Route | Auth | Description |
|--------|-------|------|-------------|
| `POST` | `/api/v1/auth/register` | No | Create account, receive tokens |
| `POST` | `/api/v1/auth/login` | No | Authenticate, receive tokens |
| `POST` | `/api/v1/auth/refresh` | No | Exchange refresh token for new pair |
| `GET` | `/api/v1/users/me` | Yes | Get authenticated user's profile |
| `PUT` | `/api/v1/users/me` | Yes | Update authenticated user's profile |
| `GET` | `/health` | No | Health check |

### Database Schema

**users**: `id` (UUID PK), `email` (unique), `password_hash`, `full_name`, `phone`, `location`, `bio`, `avatar_url`, `created_at`, `updated_at`

**refresh_tokens**: `id` (UUID PK), `user_id` (FK → users), `token_hash` (unique), `expires_at`, `created_at`, `revoked_at`

### Verification

- `go build ./...` — ✅ compiles with zero errors
- `go vet ./...` — ✅ passes static analysis
- `go mod tidy` — ✅ dependencies resolved

### What's Next

Phase 1 — Core Exchange Loop: item listings, wants specification, matching worker, match confirmation, exchange method selection, basic ratings.

---

## Phase 1 — Core Exchange Loop (2026-08-27)

### What Was Built

The complete item listing, matching, exchange confirmation, and ratings system — turning the auth-only skeleton into a functional barter marketplace.

**Scope**: 4 new DB tables, 3 new domain packages (items, matching, ratings), 14 new API endpoints, match state machine, validator extensions.

### New Files

```
internal/db/migrations/000003–000006  → items, wants, matches, ratings tables
internal/items/types.go               → Item/want request/response DTOs
internal/items/repository.go          → Transactional CRUD, dynamic filtered listing
internal/items/service.go             → Validation, business logic, pagination
internal/items/handler.go             → HTTP handlers for item CRUD + listing
internal/matching/types.go            → Match request/response DTOs
internal/matching/repository.go       → Bidirectional match finding, status transitions
internal/matching/service.go          → State machine (pending→confirmed→completed)
internal/matching/handler.go          → HTTP handlers for match lifecycle
internal/ratings/types.go             → Rating request/response DTOs
internal/ratings/repository.go        → Create, aggregate (AVG), duplicate check
internal/ratings/service.go           → Validation, cross-domain verification
internal/ratings/handler.go           → HTTP handlers for rating + user aggregate
```

### Modified Files

```
internal/validator/validator.go       → Added ValidateOneOf, ValidateRange
internal/router/router.go            → Registered all new routes
cmd/api/main.go                      → Wired items/matching/ratings DI chain
```

### Key Decisions

| Decision | Rationale |
|----------|-----------|
| **On-demand matching** (not background worker) | Simpler; matches are discovered when a user explicitly triggers search — avoids goroutine lifecycle management and job queue complexity at this scale |
| **Bidirectional match query** | Single SQL query joins items↔wants in both directions; a match requires mutual interest (A wants B's category AND B wants A's category) |
| **Soft-delete (archive)** for items | Items are never hard-deleted — `status = 'archived'` preserves data integrity for existing matches/ratings |
| **Match state machine** with 6 states | `pending → accepted_a/accepted_b → confirmed → completed → cancelled` — dual-accept prevents one-sided trades; dual-confirm prevents premature completion |
| **Item status sync** | Items automatically move `active → matched → exchanged` as their match progresses; cancel restores to `active` |
| **Images as JSONB** | Array of URLs stored as JSONB; actual upload infrastructure (S3/R2) deferred to later phase |
| **Dynamic UPDATE queries** | Item updates use partial payloads — only provided fields are SET, preventing accidental null-overwrites |
| **Ratings require completed match** | Cross-domain check via matching service — prevents gaming (can't rate before exchange is done) |
| **One rating per user per match** | UNIQUE(match_id, rater_id) constraint + application-level check for clear error messages |

### API Routes (Phase 1)

| Method | Route | Auth | Description |
|--------|-------|------|-------------|
| `POST` | `/api/v1/items` | Yes | Create item listing with wants |
| `GET` | `/api/v1/items` | No | List/search items (paginated) |
| `GET` | `/api/v1/items/{id}` | No | Get item details |
| `PUT` | `/api/v1/items/{id}` | Yes | Update item (owner) |
| `DELETE` | `/api/v1/items/{id}` | Yes | Archive item (owner) |
| `GET` | `/api/v1/users/me/items` | Yes | List my items |
| `POST` | `/api/v1/items/{id}/matches` | Yes | Find matches for my item |
| `GET` | `/api/v1/matches` | Yes | List my matches |
| `GET` | `/api/v1/matches/{id}` | Yes | Get match details |
| `POST` | `/api/v1/matches/{id}/respond` | Yes | Accept/decline match |
| `POST` | `/api/v1/matches/{id}/complete` | Yes | Confirm exchange done |
| `POST` | `/api/v1/matches/{id}/cancel` | Yes | Cancel match |
| `POST` | `/api/v1/matches/{id}/rate` | Yes | Rate exchange partner |
| `GET` | `/api/v1/users/{id}/rating` | No | Get user's aggregate rating |

### Database Schema (New Tables)

**items**: `id`, `user_id` (FK), `title`, `description`, `category`, `condition`, `exchange_method`, `images` (JSONB), `location`, `status`, `created_at`, `updated_at`

**wants**: `id`, `item_id` (FK → items), `category`, `description`, `created_at`

**matches**: `id`, `item_a_id` (FK), `item_b_id` (FK), `status`, `exchange_method`, `confirmed_a`, `confirmed_b`, `created_at`, `updated_at` — UNIQUE(item_a_id, item_b_id)

**ratings**: `id`, `match_id` (FK), `rater_id` (FK), `ratee_id` (FK), `score` (1–5), `comment`, `created_at` — UNIQUE(match_id, rater_id)

### Verification

- `go build ./...` — ✅ compiles with zero errors
- `go vet ./...` — ✅ passes static analysis

### What's Next

Phase 2 — Polish & Scale: image upload (S3/R2), notifications, search improvements (full-text search), pagination cursors, admin endpoints.


---

## Phase 1.1 — Bug Fixes & Hardening (2026-09-04)

### What Was Fixed

A cleanup pass over the Phase 1 code before moving on to Phase 2. Seven issues,
found by reading the codebase end to end.

| # | Issue | Fix |
|---|-------|-----|
| 1 | **`itoa` corrupted validation keys.** `internal/items/service.go` had a hand-rolled `itoa` using `string(rune('0'+i))` — correct only for 0–9. An item with 11+ wants produced garbage error keys (`wants[:].category`) instead of `wants[10].category`. | Replaced with `strconv.Itoa`; deleted the helper. |
| 2 | **Dead code.** `var _ = sha256.Sum256` placeholder + its import in items/service.go. | Removed both. |
| 3 | **Stale dependencies.** `go.mod` carried gin, mongo-driver, quic-go, an MCP SDK and go-playground/validator as indirect deps — none imported anywhere. | `go mod tidy`; down to 5 direct + 6 indirect (all pgx/migrate transitives). |
| 4 | **Archived items were publicly listable.** `GET /items?status=archived` returned every user's soft-deleted items — the repo defaulted to excluding archived, but an explicit filter bypassed that. | Status filters are now validated against an allowlist: `active`/`matched`/`exchanged` publicly, plus `archived` only on `GET /users/me/items`. Unknown values return 422 instead of an empty page. |
| 5 | **Owner filter could silently widen.** `Repository.List` dropped the `user_id` clause when the UUID failed to scan — turning "my items" into "everyone's items". | Returns an error instead of dropping the clause. |
| 6 | **`matches.exchange_method` was never written** — column, CHECK constraint, and DTO field all existed unused, leaving Phase 1's "exchange method selection" unfinished. | `POST /matches/{id}/respond` now takes an optional `exchange_method` (`in_person`/`shipping`) when accepting; persisted via `Repository.SetExchangeMethod`. |
| 7 | **`GetMatchRatings` was unreachable** — service method with no handler or route. | Wired as `GET /api/v1/matches/{id}/ratings`, restricted to match participants. |
| 8 | **The server could not start at all.** `migrate.New` was handed the raw `DATABASE_URL`; golang-migrate resolves its driver from the URL scheme, and the pgx/v5 driver registers itself as `pgx5` — so a standard `postgres://…` URL died with `unknown driver postgres`. Phases 0 and 1 were only ever verified with `go build`/`go vet`, never actually run. | `migrateURL` rewrites `postgres://`/`postgresql://` to `pgx5://` for the migrate instance only; pgxpool keeps the original URL. |
| 9 | **Write endpoints returned stale item statuses.** `respond`, `complete` and `cancel` built their response from the match snapshot read *before* the transition, so a completed exchange reported its items as `matched` instead of `exchanged`. The DB was correct — only the response body lied. | Added `Service.freshResponse`, which re-reads the match after the transition; all three endpoints return it. |

### Also Changed

- `GET /api/v1/users/me/items` now honours `category`, `status`, and `q` filters
  (previously it accepted only pagination and ignored the rest).
- `gofmt` applied to three files that had drifted (misaligned var blocks).

### New / Changed API Surface

| Method | Route | Auth | Change |
|--------|-------|------|--------|
| `GET` | `/api/v1/matches/{id}/ratings` | Yes | **New** — ratings on a match, participants only |
| `POST` | `/api/v1/matches/{id}/respond` | Yes | Body gains optional `exchange_method` |
| `GET` | `/api/v1/items` | No | `status=archived` now rejected with 422 |
| `GET` | `/api/v1/users/me/items` | Yes | Accepts `category`, `status` (incl. `archived`), `q` |

All changes are backward compatible except the deliberate `status=archived` rejection
on the public listing.

### Verification

- `go build ./...` — ✅
- `go vet ./...` — ✅
- `gofmt -l .` — ✅ clean
- **Ran against a real Postgres for the first time.** Migrations apply, the server boots,
  and a scripted end-to-end flow passes: register A+B → create two mutually-wanting items
  → find match → both accept (with `exchange_method`) → both complete → both rate → read
  match ratings. Negative paths checked as well: an 11th want renders as
  `wants[10].category` (issue 1), invalid `exchange_method` → 422, non-participant reading
  ratings → 403, `?status=archived` on the public list → 422, owner's archived list → 200.

### What's Next

Phase 2 — Polish & Scale: image upload (S3/R2), notifications, full-text search,
cursor pagination, admin endpoints. Plus the open debt tracked in `CLAUDE.md`:
N+1 on item listing, the sqlc decision, tests/CI, exchange-method compatibility
validation, and match status race protection.

---

## Phase B1 — React Native App: Foundation, Auth & Items (2026-09-04)

### What Was Built

The Lewe mobile app, from nothing to a running client against the local Go API.

**Stack**: Expo SDK 57 (React Native 0.86, React 19.2), TypeScript, expo-router,
TanStack Query, expo-secure-store. Lives in `mobile/`.

### Structure

```
mobile/src/
  app/                     expo-router routes
    _layout.tsx            providers: gesture root, safe area, query, theme, auth
    index.tsx              session gate — validates the stored token, then redirects
    (auth)/                welcome, login, register
    (tabs)/                browse (feed), create, profile
    item/[id].tsx          item detail
  theme/                   tokens.ts (color/space/type/radius) + ThemeProvider
  components/              Text, Button, Input, layout (Screen/Card/Row/Stack/Divider),
                           feedback (Chip/Badge/Skeleton/EmptyState/ErrorState/Avatar), ItemCard
  api/                     config (base-URL resolution), client (fetch + refresh), endpoints, types
  hooks/                   useAuth, useItems, useDebounced
  lib/                     storage (SecureStore, localStorage on web)
```

### Key Decisions

| Decision | Rationale |
|---|---|
| **Owned design system over a UI kit** | Five primitives on top of a token file. Every color and every gap comes from `useTheme()`, so dark mode is a token swap rather than a rewrite, and the visual language stays consistent as screens multiply. |
| **Single-flight token refresh** | The API *rotates* refresh tokens, so two concurrent refreshes would burn the token and log the user out. Concurrent 401s now await one shared refresh promise, and a request is replayed exactly once. |
| **API base URL derived from Metro's host** | A physical device cannot reach `localhost`. The packager URL already contains this machine's LAN IP, so the app reads it from `Constants.expoConfig.hostUri` instead of anyone hardcoding an address that changes with the router. `EXPO_PUBLIC_API_URL` overrides it. |
| **Port 8090 for the API** | 8080 is held by an unrelated service on this machine and 8081 is Metro's own dev server. `.env` now sets `PORT=:8090`. |
| **Session restored by fetching the profile** | On cold launch the app calls `/users/me` rather than trusting that a stored token exists — an expired or revoked token then fails at launch instead of on the first real interaction. |
| **Category picker over a text field** | `category` is free text in the API and matching compares with `=`, so two users typing what they mean never match. The app ships a fixed list until `GET /categories` exists (roadmap A1). |
| **TanStack Query for all server state** | Caching, refetch-on-pull, and infinite scroll without hand-rolled reducers. A 4xx is never retried — a 422 will not fix itself. |

### Screens

| Route | What it does |
|---|---|
| `/(auth)/welcome` | Value proposition, three-point explainer, routes to register/login |
| `/(auth)/register` | Full name / email / password, client-side 8-char rule matching the API, per-field 422 rendering |
| `/(auth)/login` | Email / password, show-password toggle, inline error banner |
| `/(tabs)` | Item feed: debounced search, category filter rail, infinite scroll, pull-to-refresh, skeleton / empty / error states |
| `/(tabs)/create` | Three-step listing form (details → wants → review) with a progress rail |
| `/(tabs)/profile` | Identity, rating aggregate, own listings with status badges, sign out |
| `/item/[id]` | Hero image, condition/category/status badges, description, **wants**, owner + rating |

### Verification

- `npx tsc --noEmit` — ✅ clean
- `npx expo export --platform web` — ✅ 1434 modules, all 15 routes render
- Android bundle via Metro — ✅ HTTP 200, 6.6MB, no resolution errors
- API reachable on the LAN interface (`http://10.200.0.90:8090/health`) — ✅
- Feed seeded with 8 demo listings across 3 accounts, with mutually-compatible
  wants so Phase B2's matching has something real to work against

### Known Gaps

1. **No images yet.** `images` is sent as `[]` — there is no upload endpoint (roadmap A1).
   Cards and the detail hero fall back to a placeholder.
2. **Owner names are not shown** on item detail (only the rating), because there is
   no public `GET /users/{id}` yet.
3. **The match loop is not in the app** — that is Phase B2.
4. **Firewall.** Metro's node.exe is already permitted inbound; `lewe-api.exe` is not,
   so a physical device may fail to reach port 8090 until a rule is added.

### What's Next

Phase B2 — the exchange loop: matches list, match detail, accept/decline,
exchange-method selection, completion, rating. In parallel, roadmap A1 (image
upload, categories endpoint, public profiles) unblocks the gaps above.
