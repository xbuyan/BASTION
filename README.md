# PRDS — Personal Research & Development System

> Your 5-year laboratory for mastering DevOps, Blockchain, and Cybersecurity.

## Architecture

```
prds/
├── backend/                    # Go REST API
│   ├── cmd/server/main.go     # Entry point
│   ├── internal/
│   │   ├── api/               # Gin router + all handlers
│   │   ├── auth/              # JWT + bcrypt
│   │   ├── db/                # PostgreSQL + Redis connections
│   │   ├── executor/          # Docker sandboxed code execution
│   │   ├── github/            # OAuth2 + contribution sync
│   │   ├── models/            # All domain types
│   │   ├── repository/        # Database queries
│   │   └── service/           # Business logic
│   ├── migrations/            # SQL migrations + seed data
│   ├── Dockerfile             # Multi-stage build
│   └── go.mod
├── frontend/
│   ├── prds.html              # Single-file frontend (copy from outputs)
│   ├── Dockerfile             # nginx serving static file
│   └── nginx.conf             # Reverse proxy to backend
├── scripts/
│   └── migrate_localstorage.js # Browser → PostgreSQL migration
├── docker-compose.yml
├── .env.example
├── Makefile
└── README.md
```

## Quick Start

### Option A: Standalone HTML (No Backend)

Open `prds.html` in your browser. Everything works offline with localStorage.

### Option B: Full Stack (Recommended)

**Prerequisites:** Docker, Docker Compose, `make`

```bash
# 1. Clone and configure
cp .env.example .env
# Edit .env — set JWT_SECRET, POSTGRES_PASSWORD

# 2. One-command setup
make setup

# 3. Open browser
open http://localhost:80
```

### Manual Setup

```bash
# Start infrastructure
docker compose up postgres redis -d

# Run migrations + seed
make migrate
make seed

# Start backend (with hot reload)
make dev

# Serve frontend
make dev-frontend
```

## API Reference

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/auth/register` | Create account |
| POST | `/api/v1/auth/login` | Get JWT token |
| GET | `/api/v1/me` | Current user |
| GET | `/api/v1/exercises` | List exercises (filter: `?phase=1&domain=crypto`) |
| GET | `/api/v1/exercises/:id` | Exercise detail |
| POST | `/api/v1/exercises/:id/run` | Execute code in Docker sandbox |
| POST | `/api/v1/exercises/:id/complete` | Mark complete (+100 XP) |
| GET | `/api/v1/review/queue` | Due spaced-repetition cards |
| POST | `/api/v1/review/:id` | Submit rating (quality 0-5) |
| GET | `/api/v1/concepts/graph` | Knowledge graph nodes + edges |
| GET | `/api/v1/projects` | Capstone projects |
| POST | `/api/v1/projects/:id/advance` | Advance stage (+200 XP) |
| GET | `/api/v1/papers` | Research papers |
| POST | `/api/v1/papers/import/arxiv` | Import from arXiv |
| GET | `/api/v1/contributions` | OSS contributions |
| POST | `/api/v1/github/sync` | Sync GitHub PRs |
| GET | `/api/v1/progress` | Full progress summary |
| GET | `/api/v1/analytics/skills` | Skill growth by domain |
| GET | `/api/v1/export/json` | Full data export |
| GET | `/api/v1/export/markdown` | Markdown progress report |
| POST | `/api/v1/migrate/localstorage` | Import localStorage data |

## Migrating from localStorage

If you've been using the HTML-only version and built up progress:

```bash
# 1. Open browser console on the PRDS page and run:
# (see scripts/migrate_localstorage.js for full script)

const login = await fetch('http://localhost:8080/api/v1/auth/login', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ email: 'you@example.com', password: 'yourpassword' })
}).then(r => r.json());

await migrateToBackend('http://localhost:8080', login.token);
```

## Code Execution Sandbox

The backend runs user Go code in isolated Docker containers:
- **Memory:** 256MB limit
- **CPU:** 1 core limit  
- **Timeout:** 10 seconds
- **Network:** Disabled
- **Filesystem:** Read-only root + tmpfs for /tmp

Rate limit: 10 executions/minute per IP.

To disable Docker execution (dev without Docker socket):
```bash
DOCKER_ENABLED=false make dev
# Falls back to local subprocess execution (no sandboxing)
```

## Monitoring

```bash
# Start with Prometheus + Grafana
make monitoring-up

# Access
# Prometheus: http://localhost:9090
# Grafana:    http://localhost:3001 (admin/admin)
```

## Production Deployment

```bash
# 1. Set production env
cp .env.example .env
# Set: ENV=production, strong JWT_SECRET, strong POSTGRES_PASSWORD

# 2. Build and start
make prod-build
make prod-up

# 3. Set up nginx reverse proxy (see deploy/nginx-prod.conf)
# 4. SSL via certbot

# Backups (run via cron)
make backup
# Restore: BACKUP_FILE=backup-20250101.sql.gz make restore
```

## Development

```bash
make test          # Run all tests
make lint          # golangci-lint
make test-cover    # Coverage report
make db-shell      # psql shell
make db-reset      # Full DB reset + reseed
```

## XP System

| Action | XP |
|--------|----|
| Complete exercise | +100 |
| Complete milestone | +25 |
| Advance project stage | +200 |
| Read research paper | +50 |
| Log session | +50 |

## License

MIT — Build revolutionary things.
