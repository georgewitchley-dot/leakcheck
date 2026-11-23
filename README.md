# LeakCheck — password leak checker

Type a password and see whether it appears in known data breaches, and how many times. Free, no accounts, no tracking.

**Your password never leaves the browser.** The frontend hashes it with SHA-1 (Web Crypto), sends only the first
5 hex characters of the hash to the backend, and matches the returned suffixes locally. This is the
[k-anonymity model](https://haveibeenpwned.com/API/v3#SearchingPwnedPasswordsByRange) of Have I Been Pwned's
Pwned Passwords range API.

## Stack

| Part | What it does |
| --- | --- |
| `frontend/` — React + TypeScript (Vite) | Hashing, prefix lookup, local suffix match, UI |
| `backend/` — Go (`net/http`, pgx) | `GET /api/range/{prefix}` proxy to HIBP with `Add-Padding`, Postgres cache (24h TTL, serves stale if HIBP is down), per-IP rate limit (in memory), `GET /api/stats`, `GET /healthz`, strict CSP, serves the built frontend |
| Postgres | `range_cache(prefix, body, fetched_at)` and `check_stats(day, checks)` — no passwords, hashes, or IPs stored |

## Run locally

```sh
docker compose up --build
# open http://localhost:8080
```

Without Docker:

```sh
cd frontend && npm ci && npm run build
cd ../backend && STATIC_DIR=../frontend/dist go run -tags nopg .   # in-memory store, no Postgres
```

## Tests

```sh
cd backend && go test -tags nopg ./...
cd frontend && npm test
```

## Deploy (Railway)

1. Add a Postgres database to the project.
2. Add a service from this repo (it builds the root `Dockerfile`).
3. Set `DATABASE_URL=${{Postgres.DATABASE_URL}}` on the service, healthcheck path `/healthz`, and generate a domain.

## Configuration

| Env var | Default | |
| --- | --- | --- |
| `DATABASE_URL` | — (required unless built with `-tags nopg`) | Postgres connection string |
| `PORT` | `8080` | Listen port |
| `STATIC_DIR` | `./public` | Built frontend directory |
| `HIBP_BASE_URL` | `https://api.pwnedpasswords.com` | Upstream (override for testing) |

Breach data: [Have I Been Pwned](https://haveibeenpwned.com/Passwords) Pwned Passwords (CC BY 4.0).
