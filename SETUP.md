# BASTION — Setup Guide

This guide walks through setting up BASTION from a fresh clone on any machine. Follow it in order — the steps below account for real issues encountered during initial setup, so skipping ahead can reintroduce them.

## Prerequisites

- **Docker** and **Docker Compose** (v2, i.e. `docker compose`, not the legacy `docker-compose`)
- `git`
- `openssl` (used to generate secrets — available by default on macOS/Linux; on Windows use WSL or Git Bash)

## 1. Clone the repository

```bash
git clone https://github.com/xbuyan/BASTION.git
cd BASTION
```

## 2. Configure environment variables

```bash
cp .env.example .env
```

Open `.env` and replace the placeholder values for `POSTGRES_PASSWORD` and `JWT_SECRET` with real generated secrets:

```bash
openssl rand -hex 24   # use output for POSTGRES_PASSWORD
openssl rand -hex 32   # use output for JWT_SECRET
```

**Important:** Use `openssl rand -hex`, not `openssl rand -base64`.

Base64 output can contain `/` and `+` characters. The backend builds a Postgres connection string as a URL (`postgres://user:password@host:port/db`), and a `/` inside the password breaks the URL parser — the app will fail at migration time with an error like:

```
invalid port "..." after host
```

Hex output only contains `0-9a-f`, which is always safe in this context.

Do **not** leave `POSTGRES_PASSWORD` as the example placeholder (`changeme_use_strong_password`) — Postgres will happily accept it, but it's not meant to be used as-is, and it makes credential mixups likely if you ever copy `.env` between environments.

## 3. Build and start the stack

```bash
docker compose up --build
```

This builds and starts four services: `postgres`, `redis`, `backend`, `frontend`.

### What a healthy startup looks like

In the `postgres` logs, on first run you should see the full `initdb` sequence complete, ending with:
```
database system is ready to accept connections
```

In the `backend` logs, look for:
```
migrations applied successfully
...
BASTION backend started    {"port": "8080", "env": "development"}
```

If you see `FATAL` errors instead, see [Troubleshooting](#troubleshooting) below.

## 4. Open the app

```
http://localhost:80
```

Register a new account through the UI, then log in.

## 5. Verify exercises are seeded (optional but recommended)

The backend runs two migrations on startup: one creates the schema, the second seeds real study exercises. Confirm both ran:

```bash
docker compose exec postgres psql -U prds -d prds \
  -c "SELECT * FROM schema_migrations;" \
  -c "SELECT COUNT(*) FROM exercises;"
```

Expected output:
```
 version | dirty
---------+-------
       2 | f

 count
-------
     6
```

If `version` shows `1` instead of `2`, or the exercise count is `0`, the seed migration didn't run — see [Troubleshooting](#troubleshooting).

---

## Troubleshooting

### `password authentication failed for user "prds"`

**Cause:** Postgres only initializes its users/passwords once, the first time its data directory is empty. If a Docker volume already exists from a previous run (with different credentials than what's currently in `.env`), the new app can't authenticate.

**Fix:** Wipe the volume and let Postgres re-initialize with the current `.env` values. Only do this if there's no real data in the database yet — this is destructive.

```bash
docker compose down -v
docker compose up --build
```

If you suspect there are old, unrelated volumes lying around (e.g. from an earlier project name), list them and remove anything stale:

```bash
docker volume ls | grep -i bastion
docker volume rm <stale_volume_name>
```

### `invalid port "..." after host` (migration failure)

**Cause:** `POSTGRES_PASSWORD` contains a `/`, `@`, or other character that breaks URL parsing in the Postgres connection string.

**Fix:** Regenerate the password using `openssl rand -hex 24` (hex only, no special characters), update `.env`, then wipe and rebuild as above.

### `/api/v1/exercises` returns `[]` (empty array) even though login works

**Cause:** The seed migration didn't run. golang-migrate only picks up files matching the pattern `NNN_name.up.sql` — if `002_seed_data.sql` is missing the `.up.sql` suffix, it's silently skipped, and only the schema (migration 001) gets applied.

**Fix:** Check the migrations folder:
```bash
ls backend/migrations/
```
Confirm both files end in `.up.sql`. If not, rename and rebuild (migrations are baked into the Docker image, so a rename alone isn't enough — you must rebuild):
```bash
mv backend/migrations/002_seed_data.sql backend/migrations/002_seed_data.up.sql
docker compose down
docker compose up --build
```

### `nginx: [emerg] host not found in upstream "backend"`

**Cause:** This is usually a downstream symptom, not a root cause. It happens when the `backend` container isn't running (e.g. it's stuck crash-looping from a database connection failure), so nginx can't resolve its hostname on the shared Docker network.

**Fix:** Resolve whatever is causing `backend` to crash-loop (check `docker compose logs backend` for the real error — likely one of the Postgres issues above). Once `backend` stays up, this error clears on its own.

### Token / auth errors when testing the API manually with `curl`

If you're manually copying a JWT out of a JSON response and pasting it into a second command, it's easy to grab a stray character (e.g. part of the next JSON field). This produces a corrupted token and a generic `"invalid or expired token"` error — even though the token isn't actually expired.

**Fix:** Extract the token programmatically instead of copy-pasting:
```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"you@example.com","password":"yourpassword"}' \
  | grep -o '"token":"[^"]*' | cut -d'"' -f4)

curl -s http://localhost:8080/api/v1/exercises -H "Authorization: Bearer $TOKEN"
```

---

## Useful commands

```bash
# Stop everything (containers only, keep data)
docker compose down

# Stop everything and wipe all data (destructive)
docker compose down -v

# View logs for a specific service
docker compose logs -f backend

# Open a psql shell into the database
docker compose exec postgres psql -U prds -d prds

# Rebuild after code changes
docker compose up --build
```
