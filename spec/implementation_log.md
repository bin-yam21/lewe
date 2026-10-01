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

## Phase 1 — Core Exchange Loop (2026-09-28)

### What Was Built

Item listings, wants, a background matching worker, the match lifecycle
(accept/decline → exchange method → completion/cancel) and ratings. Plus
fixes to Phase 0, end-to-end tests, Docker, CI and a README.

### New Packages

```
internal/app/       → Wiring (repo → service → handler) shared by main and tests
internal/catalog/   → Fixed categories, conditions, exchange methods
internal/items/     → Listings: CRUD, public browse/search, soft delete (withdraw)
internal/wants/     → What a user is looking for: category, keywords, min condition
internal/matches/   → Match state machine + matching worker
internal/ratings/   → 1–5 ratings after a completed match
internal/request/   → Body decoding (1 MiB cap), UUID path params, pagination
```

### Database Schema

**items**: owner, title, description, category, condition, estimated_value, location, image_urls, status (`available`/`reserved`/`exchanged`/`withdrawn`)

**wants**: user, category, keywords[], min_condition, status (`active`/`fulfilled`/`cancelled`)

**matches**: user/item/want for sides A and B, score, status (`pending`/`accepted`/`completed`/`declined`/`cancelled`), per-side accepted/completed timestamps, exchange method/details/proposer, closed_by. `CHECK (item_a_id < item_b_id)` + `UNIQUE (item_a_id, item_b_id)`.

**ratings**: match, rater, ratee, score 1–5, comment. `UNIQUE (match_id, rater_id)`.

`item_condition_rank(text)` is an immutable SQL function ordering conditions so wants can set a minimum.

### Key Decisions

| Decision | Rationale |
|----------|-----------|
| **Matching as one SQL statement** | `INSERT … SELECT` joins wants to items twice to find reciprocal pairs. Postgres does the work in one round trip and `ON CONFLICT DO NOTHING` makes passes idempotent |
| **Canonical pair ordering** | Every swap is found from both sides; keeping only `item_a_id < item_b_id` dedupes it and lets a unique constraint stop repeat suggestions — including of declined/cancelled pairs |
| **Worker = ticker + trigger** | Runs every `MATCH_INTERVAL`, and immediately (debounced) after an item or want is created/updated so users see matches quickly |
| **Row locks for state changes** | Every match transition locks the match `FOR UPDATE`; accepting also locks both items in id order, so two matches can't reserve the same item |
| **Reserve on accept, cancel competitors** | Once both accept, both items become `reserved` and every other pending match involving them is cancelled |
| **Changing the exchange plan resets completion** | A confirmation applied to the old plan; both sides must confirm again |
| **Viewer-relative match responses** | `your_item`/`their_item` spares clients from working out which side they are |
| **Fixed category list** | Matching needs exact category equality; free text would fragment |
| **Soft delete for items and wants** | Matches and ratings keep referring to them |

### Phase 0 Fixes

- **Migrations never ran against a `postgres://` URL**: golang-migrate's pgx v5 driver only registers `pgx5://`. The URL is now rewritten, and migrations are embedded with `go:embed` so the binary works from any directory.
- **Refresh-token reuse race**: lookup and revoke were two statements, so two concurrent requests could redeem one token. It is now one `UPDATE … RETURNING`.
- Emails are trimmed and lower-cased (they were case-sensitive).
- Passwords over 72 bytes are rejected (bcrypt can't hash them).
- Added `POST /auth/logout` and public profiles with a rating summary.
- Request bodies are capped at 1 MiB; handler panics return a 500 instead of dropping the connection; 500s log the underlying error.

### Verification

- `go build ./...`, `go vet ./...`, `gofmt` — ✅
- Unit tests (validator, JWT, request helpers, migration URL) — ✅
- End-to-end tests against PostgreSQL 16 (`go test -race ./...`) covering auth, items, wants, the full match lifecycle, decline/cancel/withdraw and races with item availability — ✅
- Manual smoke test of the built binary: migrations, the worker finding a match right after a want is created, graceful shutdown — ✅
- All down migrations tested in reverse order — ✅

### Not Done / Possible Next Steps

See Phase 2 below.

---

## Phase 2 — Communication & Hardening (2026-09-30)

### What Was Built

- **Messaging** (`internal/messages`): a chat thread per match. Participants
  can post while the match is pending, accepted or completed; declined or
  cancelled matches keep their history readable but reject new messages.
- **In-app notifications** (`internal/notifications`): the matching worker,
  every match transition, messages and ratings notify the affected users.
  Endpoints list them, count unread and mark one/all as read.
- **Auth rate limiting**: per-IP token bucket on `/auth/*` (`AUTH_RATE_LIMIT`,
  default 20/min), `429` + `Retry-After`. `TRUST_PROXY` uses
  `X-Forwarded-For` behind a reverse proxy.
- **CORS** for browser clients (`CORS_ALLOWED_ORIGINS`).
- `app.New` now takes an `app.Config`; `router.New` takes `router.Options`.

### Database Schema

**messages**: match, sender, body, created_at. Indexed by `(match_id, created_at)`.

**notifications**: user, type, match, read_at, created_at. Partial index on
unread rows for the unread count.

### Key Decisions

| Decision | Rationale |
|----------|-----------|
| **Notifications written in the same transaction as the event** | A notification can never describe something that was rolled back, and never goes missing for something that happened |
| **Worker notifies inside its single SQL statement** | `INSERT … RETURNING` feeds a second `INSERT` into notifications via data-modifying CTEs — still one round trip |
| **`notifications.CancelPendingMatches` helper** | Items, wants and matches all cancel pending matches as a side effect; one helper cancels them and notifies both users consistently |
| **`created_at DEFAULT clock_timestamp()` on notifications** | One transaction can create several notifications for the same user (e.g. "confirmed" plus "competing match cancelled"). `now()` is fixed per transaction, which made their order random; `clock_timestamp()` preserves insertion order |
| **Notification text generated from `type`** | Clients get a ready-to-show `message` but can localise by `type` |
| **In-memory rate limiter** | No new dependency or infrastructure; fine for a single instance. Multiple instances would need a shared store (e.g. Redis) |
| **Rate limiting only on `/auth/*`** | Those are the endpoints worth brute-forcing; everything else requires a token |

### Verification

- `go build`, `go vet`, `gofmt` — ✅
- Unit tests for the rate limiter (refill, per-client buckets, sweeping, proxy
  header) and CORS — ✅
- End-to-end tests for messaging, the notification for every lifecycle step,
  read/unread handling, auth rate limiting and CORS preflight
  (`go test -race ./...`, repeated runs) — ✅
- Smoke test of the built binary and all down migrations in reverse order — ✅

### Still Not Done

See Phase 3 below.

---

## Phase 3 — Account Recovery (2026-10-01)

### What Was Built

- **Email verification**: sign-up emails a link; `POST /auth/verify-email`
  confirms it; `POST /users/me/verify-email` sends a new one. Profiles expose
  `email_verified`.
- **Password reset**: `POST /auth/forgot-password` emails a 1-hour link;
  `POST /auth/reset-password` sets the new password, signs out every session
  and marks the email verified.
- **Change password**: `PUT /users/me/password` checks the current password,
  signs out other sessions and returns a fresh token pair.
- **`internal/mail`**: `Mailer` interface with an SMTP implementation
  (STARTTLS or implicit TLS on 465, header-injection safe), a log mailer for
  development and an in-memory mailer for tests.
- The users repository now scans through one `userColumns` list.

### Database Schema

**users**: + `email_verified_at`.

**account_tokens**: user, purpose (`verify_email` / `reset_password`),
token_hash (unique), expires_at, used_at.

### Key Decisions

| Decision | Rationale |
|----------|-----------|
| **Hash emailed tokens like refresh tokens** | A database leak doesn't expose working links |
| **Consume with one `UPDATE … RETURNING`** | A link can't be used twice, even concurrently |
| **A new link invalidates older unused ones** | Only the most recent email works |
| **Forgot-password always returns 202 with the same message** | Callers can't discover which emails have accounts |
| **Mail failures don't fail sign-up or forgot-password** | They're logged; the user can request another link |
| **Verification isn't enforced yet** | `email_verified` is exposed so a policy (e.g. verified users only can list items) can be added later without a schema change |
| **Log mailer by default** | Local development works with no SMTP server; links appear in the server log |

### Verification

- End-to-end test of sign-up verification, resend (older link stops working),
  forgot/reset password (no account enumeration, single use, sessions
  revoked, expiry), and change password — ✅
- Unit test for email header injection — ✅
- `go test -race ./...`, `make lint`, down migrations, and a smoke test
  showing emailed links in the log without SMTP — ✅

### Still Not Done

- Image upload (items take image URLs only)
- Email/push delivery of notifications (they are in-app only)
- Real-time delivery (WebSocket/SSE); clients poll `/notifications/unread-count`
- Multi-party (A→B→C→A) swap cycles; matching is two-way only
- The sqlc query files still cover only users and refresh tokens; repositories use pgx directly
