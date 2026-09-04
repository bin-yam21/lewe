# Lewe — Build Roadmap

Written 2026-09-04, after the Phase 1.1 fix pass. Companion to
`spec/implementation_log.md` (what has been built) and `CLAUDE.md` (how to work
in this repo). This file is **what comes next**.

Two tracks run from here: the **Go API** (`internal/…`) and the **React Native
app** (`mobile/`). They are numbered separately because they progress
independently — the app can be built against endpoints that already exist while
the API grows underneath it.

---

## Where We Are

| | Status |
|---|---|
| **API** | Phases 0, 1, 1.1 done. 21 endpoints. Auth, items + wants, matching state machine, ratings. Runs against Postgres, end-to-end flow verified. |
| **App** | Not started — this roadmap begins it. |
| **Tests** | None. The Postman collection is the de-facto suite. |
| **Deployment** | None. Local only. |

---

## Track A — The API

### Phase A1 · Make the App Possible *(highest priority)*

This phase is defined by what `mobile/` will ask for, not by what would be nice
to have.

| Item | Why the app needs it |
|---|---|
| **Image upload** (S3 / Cloudflare R2 presigned PUT) | `items.images` is a JSONB array of URLs with no way to produce a URL. A listing without a photo is not a listing. `POST /api/v1/uploads/presign` returns `{upload_url, public_url}`; the app PUTs straight to storage and sends `public_url` in the item payload. |
| **Categories endpoint** | `category` is free text today, and matching compares categories with `=`. A typo — `"Electronics"` vs `"electronics"` — silently breaks matching. `GET /api/v1/categories` returns a fixed list and the app renders a picker instead of a text field. **This is a correctness fix wearing a feature's clothes.** |
| **Public user profile** | `GET /api/v1/users/{id}` — item detail has to show who owns the item. Returns `id`, `full_name`, `avatar_url`, `location`, `created_at` and the rating aggregate. Never `email` or `phone`. |
| **Cursor pagination** | The feed is an infinite scroll. `LIMIT/OFFSET` skips and duplicates rows when items are inserted mid-scroll. Move `GET /items` to `?cursor=<created_at,id>&limit=`. |
| **Fix the N+1** | `ListItems` runs one wants query per row. Replace with one `WHERE item_id = ANY($1)` and group in Go. |

### Phase A2 · Trust & Correctness

| Item | Detail |
|---|---|
| **Match transition races** | `RespondToMatch` reads then writes with no lock — two simultaneous accepts interleave. Wrap in a transaction with `SELECT … FOR UPDATE`, or make the UPDATE conditional (`WHERE status = $expected`) and treat 0 rows affected as a conflict. |
| **Exchange-method compatibility** | A match can settle on `shipping` when one item is `in_person` only. Pull both items' `exchange_method` into `MatchWithItems` and validate. |
| **Rate limiting** | `POST /auth/login` and `/auth/register` are unthrottled — credential stuffing and signup spam. Per-IP token bucket in middleware. |
| **Refresh-token cleanup** | Revoked and expired rows accumulate forever. `DELETE … WHERE expires_at < now() - 30d` on a ticker. |
| **Structured logging** | `log.Printf` → `log/slog`, with a request ID in context so one request can be traced across layers. |
| **Tests** | Table-driven unit tests for the match state machine and the validators first — pure logic, no DB, and where the real bugs live. Then handler tests over `httptest` against a Dockerised Postgres. |

### Phase A3 · Engagement

Push notifications (a match appeared; the other party accepted; they marked it
complete) via Expo push tokens — `POST /api/v1/devices` to register, plus a
`notifications` table for the in-app inbox. Then **in-match messaging**, which is
the honest gap in the product: two people agree to trade and currently have no
way to arrange where and when.

Also here: full-text search (`tsvector` + GIN over title and description,
replacing `ILIKE '%…%'`) and location filtering — PostGIS, or a plain lat/long
bounding box. "Within 10km" matters enormously for `in_person` trades.

### Phase A4 · Operations

Docker Compose (API + Postgres), a Makefile or Taskfile, GitHub Actions running
build/vet/test, deploy to Fly.io or Railway, Sentry, `/metrics`. Also: move
migrations off the boot path — a container that migrates on every start will race
itself the moment you run two replicas.

---

## Track B — The React Native App

### Stack

| Choice | Rationale |
|---|---|
| **Expo (managed) + TypeScript** | No Android SDK on this machine; Expo Go puts it on a real device today. Ejectable later if a native module demands it. |
| **expo-router** | File-based routing, typed routes, native stack and tabs without hand-wiring navigators. |
| **TanStack Query** | This app is almost entirely server state. Caching, refetching, optimistic updates and infinite scroll come free, and it avoids a Redux-shaped ball of async reducers. |
| **expo-secure-store** | Refresh tokens belong in the Keychain/Keystore, never `AsyncStorage`. |
| **An owned design system, not a UI kit** | A small set of primitives (`Text`, `Button`, `Input`, `Card`, `Screen`) over tokens. UI kits start fighting you the moment the design gets specific. |

### App Architecture

```
mobile/
  app/                      expo-router routes (the file tree IS the navigation)
    (auth)/                 unauthenticated: welcome, login, register
    (tabs)/                 authenticated shell: feed, matches, create, profile
    item/[id].tsx           item detail
  src/
    api/                    typed client, one module per API domain
    components/             design-system primitives + composed components
    theme/                  tokens: color, spacing, type, radius
    hooks/                  useAuth, useItems, … (TanStack Query wrappers)
    lib/                    storage, formatting, validation
```

**The API client is the load-bearing piece.** One `fetch` wrapper that attaches
the access token and, on a 401, refreshes once and replays the request —
serialising concurrent refreshes so that ten parallel 401s cause one refresh, not
ten. Get this right and every screen above it stays simple.

### Visual Direction — Clean & Modern

High-contrast neutrals, one vivid accent, crisp typography, flat cards, tight
purposeful spacing. This is an app where strangers agree to meet and exchange
property, so above all it has to feel *credible*.

- **Ink** `#0B0B0F` on **paper** `#FFFFFF`, with a true grey ramp between.
- **Accent** a single saturated indigo, used *only* for primary actions and
  active state. Never decorative.
- **Type** system stack, tight tracking on headings. Four sizes, three weights,
  no more.
- **Space** a 4px base scale. Cards sit flat on a subtly tinted ground instead of
  floating on shadows.
- **Motion** short and functional — 150–200ms, ease-out. Press states matter more
  than screen transitions.
- Dark mode designed in at the token layer, not retrofitted later.

### Phase B1 · Foundation + Auth + Items  ← **today**

1. **Scaffold** — Expo + TypeScript + expo-router, in `mobile/`.
2. **Theme** — tokens and primitives. Everything after this is assembly.
3. **API client** — typed fetch wrapper, error normalisation, token refresh.
4. **Auth** — welcome / register / login, secure token storage, session restore
   on launch, and a route guard splitting `(auth)` from `(tabs)`.
5. **Item feed** — infinite list, search, category filter, empty and loading states.
6. **Item detail** — images, condition, wants, owner and their rating.
7. **Create listing** — a multi-step form: details → wants → review.

### Phase B2 · The Exchange Loop

Matches list grouped by state, match detail showing both items side by side, the
accept/decline decision, exchange-method selection, completion confirmation, and
the rating sheet. This is where the app becomes the product — the state machine
in `internal/matching` needs a UI that makes "who is waiting on whom" obvious at
a glance.

### Phase B3 · Depth

Profile editing with avatar upload, my-listings management (edit, archive), push
notifications, in-match messaging, offline-tolerant caching.

### Phase B4 · Ship

Icon and splash, EAS Build, store listings, crash reporting, a real onboarding pass.

---

## Sequencing

The two tracks interlock at exactly two points, and both sit in **A1**:

1. **Categories must become a fixed list before the app ships a category picker.**
   Free-text categories that don't match exactly are invisible bugs — two users
   both typing what they mean and no match ever being found.
2. **Image upload must exist before "create listing" is real.**

Everything else in Track A can proceed while the app is built against what
already works.

---

## Today's Session Plan (Phase B1)

| # | Step | Done when |
|---|------|-----------|
| 1 | Scaffold Expo app in `mobile/` | `npx expo start` serves a QR code |
| 2 | Theme tokens + primitives | `Button`, `Input`, `Card`, `Text`, `Screen` render in both color schemes |
| 3 | API client + auth storage | A token refresh replays a 401'd request exactly once |
| 4 | Welcome / Register / Login | A real account created against the local Go API from the phone |
| 5 | Session guard + tab shell | Cold launch lands on the feed when logged in, welcome when not |
| 6 | Item feed + search + detail | Items created via curl appear on the phone |
| 7 | Create listing | An item created on the phone appears in `GET /items` |

**Connectivity note.** The phone reaches the API at **`http://10.200.0.90:8081`** —
the LAN IP of this machine, not `localhost`. Port **8081**, not 8080: an unrelated
`AgentService.exe` already holds 8080 on this machine, so `.env` now sets
`PORT=:8081`. Two things have to be true for the device to connect: the Go server
binds on all interfaces (it does), and Windows Firewall allows inbound 8081 on a
*private* network. If a request hangs rather than failing fast, the firewall is
the first suspect, not the code.
