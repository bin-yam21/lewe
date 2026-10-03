# Lewe Mini App

Telegram Mini App frontend for the Lewe API. React + TypeScript + Vite, no UI
framework; colours follow the user's Telegram theme.

## Run locally

```sh
# 1. API (from the repo root) — TELEGRAM_BOT_TOKEN enables Telegram sign-in
make db-up && TELEGRAM_BOT_TOKEN=<token from @BotFather> make run

# 2. Mini app
cd web && npm install && npm run dev      # http://localhost:5173, proxies /api to :8080
```

Opened in a normal browser the app falls back to email/password sign-in, so it
can be developed without Telegram.

## Try it inside Telegram

1. Create a bot with [@BotFather](https://t.me/BotFather) and keep its token.
2. Serve the app over HTTPS (e.g. `ngrok http 5173`, or deploy `npm run build`'s
   `dist/` to any static host). Set `VITE_API_URL` to the public API URL when the
   API is on a different origin, and add the app's origin to the API's
   `CORS_ALLOWED_ORIGINS`.
3. In BotFather: `/newapp` (or *Bot Settings → Menu Button*) and point it at the URL.

## How sign-in works

On launch the app sends Telegram's signed `initData` to `POST /api/v1/auth/telegram`.
The API verifies the HMAC with the bot token, rejects data older than 24h, and
returns Lewe access/refresh tokens, creating the account on first use. Tokens
are kept in `localStorage` and refreshed automatically.

## Screens

Browse · Item detail · List/edit item · My items · Wants · Matches · Match
detail (accept/decline, exchange method, confirm hand-over, cancel, chat,
rating) · Notifications · Profile · Public profile with reviews.

## Limitations

- Items take image URLs (the API has no image upload yet).
- Notifications and chat poll (30s / 10s); the API has no push channel yet.
- Email verification and password reset are not surfaced in the UI.
