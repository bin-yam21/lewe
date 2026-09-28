# Lewe

Lewe is a barter marketplace API written in Go. People list items they want to
give away, describe what they want in return, and a background worker finds
**two-way swaps**: you have something I want, and I have something you want.
Both sides accept, agree how to exchange, confirm the hand-over, and rate each
other.

## Quick start

```sh
cp .env.example .env          # then set JWT_SECRET
make db-up                    # PostgreSQL 16 in Docker (also creates lewe_test)
make run                      # migrations run automatically on start-up
```

Or run everything in Docker: `docker compose up --build`.

| Variable         | Required | Default | Description |
|------------------|----------|---------|-------------|
| `DATABASE_URL`   | yes      | —       | PostgreSQL connection string |
| `JWT_SECRET`     | yes      | —       | HMAC secret for access tokens (use 32+ random chars) |
| `PORT`           | no       | `:8080` | Listen address (`8080` or `:8080`) |
| `MATCH_INTERVAL` | no       | `1m`    | How often the matching worker runs |

## Testing

```sh
make test        # unit + end-to-end tests (needs PostgreSQL, see TEST_DATABASE_URL)
make test-unit   # unit tests only; end-to-end tests are skipped
make lint        # gofmt + go vet
```

End-to-end tests in `internal/app` drive the real HTTP API against a real
database and **truncate all tables**, so point `TEST_DATABASE_URL` at a
throwaway database.

## How an exchange works

```
 list items + wants ──▶ worker finds swap ──▶ pending
                                                 │ both accept (items reserved,
                                                 ▼  competing matches cancelled)
                     declined ◀── decline ── accepted ── cancel ──▶ cancelled
                                                 │                 (items released)
                                 choose exchange method, both confirm
                                                 ▼
                                             completed ──▶ both rate each other
                                     (items exchanged, wants fulfilled)
```

A want matches an item when the category is the same, the item is at least the
want's `min_condition`, and — if the want has keywords — at least one keyword
appears in the item's title or description. A match's `score` (0–1) is higher
when the two items have similar estimated values. Each pair of items is only
ever matched once, so a declined or cancelled swap is not suggested again.

## API

All endpoints are under `/api/v1` and speak JSON. Authenticated endpoints need
`Authorization: Bearer <access_token>`. Lists accept `?limit=` (default 20,
max 100) and `?offset=` and return `{"data": [...], "limit": n, "offset": n}`.
Validation failures return `422` with `{"error": ..., "fields": {field: message}}`.

### Auth & users

| Method | Route | Auth | Description |
|--------|-------|------|-------------|
| `POST` | `/auth/register` | — | `{email, password, full_name}` → tokens + user |
| `POST` | `/auth/login` | — | `{email, password}` → tokens + user |
| `POST` | `/auth/refresh` | — | `{refresh_token}` → new token pair (old one is revoked) |
| `POST` | `/auth/logout` | — | `{refresh_token}` → revokes it |
| `GET` | `/users/me` | ✓ | Your profile |
| `PUT` | `/users/me` | ✓ | `{full_name, phone?, location?, bio?, avatar_url?}` |
| `GET` | `/users/me/items` | ✓ | Your items, any status (`?status=`) |
| `GET` | `/users/{id}` | — | Public profile with rating average and count |
| `GET` | `/users/{id}/ratings` | — | Ratings a user has received |

Access tokens last 15 minutes; refresh tokens last 7 days and are single-use.

### Items

| Method | Route | Auth | Description |
|--------|-------|------|-------------|
| `GET` | `/categories` | — | Valid categories, conditions and exchange methods |
| `GET` | `/items` | — | Browse available items (`?category=`, `?q=`, `?owner_id=`) |
| `POST` | `/items` | ✓ | Create a listing |
| `GET` | `/items/{id}` | optional | One item (withdrawn items are visible only to their owner) |
| `PUT` | `/items/{id}` | ✓ owner | Replace a listing (only while `available`) |
| `DELETE` | `/items/{id}` | ✓ owner | Withdraw a listing; cancels its pending matches |

Item body: `{title, description?, category, condition, estimated_value?, location?, image_urls?}`.
Conditions, best to worst: `new`, `like_new`, `good`, `fair`, `poor`.
Item status: `available` → `reserved` (accepted match) → `exchanged`, or `withdrawn`.

### Wants

| Method | Route | Auth | Description |
|--------|-------|------|-------------|
| `GET` | `/wants` | ✓ | Your wants (`?status=active\|fulfilled\|cancelled`) |
| `POST` | `/wants` | ✓ | `{category, keywords?: [...], min_condition?}` |
| `GET` | `/wants/{id}` | ✓ | One of your wants |
| `PUT` | `/wants/{id}` | ✓ | Replace an active want |
| `DELETE` | `/wants/{id}` | ✓ | Cancel a want and its pending matches |

### Matches & ratings

| Method | Route | Auth | Description |
|--------|-------|------|-------------|
| `GET` | `/matches` | ✓ | Your matches (`?status=`) |
| `GET` | `/matches/{id}` | ✓ | One match, from your point of view |
| `POST` | `/matches/{id}/accept` | ✓ | Accept a pending match |
| `POST` | `/matches/{id}/decline` | ✓ | Decline a pending match |
| `PUT` | `/matches/{id}/exchange` | ✓ | `{method: meetup\|shipping\|dropoff, details?}` on an accepted match |
| `POST` | `/matches/{id}/complete` | ✓ | Confirm you've handed over your item |
| `POST` | `/matches/{id}/cancel` | ✓ | Back out of an accepted match |
| `POST` | `/matches/{id}/rating` | ✓ | `{score: 1–5, comment?}` once the match is completed |

A match is returned relative to the viewer: `your_item`, `their_item`,
`other_user`, `you_accepted`/`they_accepted`, `exchange`,
`you_completed`/`they_completed` and `you_rated`. Changing the exchange method
or details resets both completion confirmations.

`GET /health` returns `{"status": "ok"}`.

## Project layout

```
cmd/api/              entry point: config, migrations, server, worker, graceful shutdown
internal/app/         dependency wiring + end-to-end tests
internal/router/      routes (Go 1.22+ stdlib mux)
internal/users/       auth + profiles        ┐
internal/items/       listings               │ each: handler → service → repository
internal/wants/       wants                  │
internal/matches/     match lifecycle + worker
internal/ratings/     ratings                ┘
internal/db/          pgx pool, transactions, embedded SQL migrations
internal/middleware/  auth, logging, panic recovery
internal/request/     JSON decoding, path IDs, pagination
internal/response/    JSON response helpers
internal/validator/   input validation
internal/catalog/     categories, conditions, exchange methods
```

See [`spec/implementation_log.md`](spec/implementation_log.md) for design decisions.
