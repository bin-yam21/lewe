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
