# PRDS Makefile
# Usage: make <target>

.PHONY: help dev build test clean migrate seed lint docker-up docker-down

# ─── Colors ───────────────────────────────────────────────────────────────────
GREEN  := \033[0;32m
CYAN   := \033[0;36m
YELLOW := \033[0;33m
RESET  := \033[0m

help: ## Show this help
	@echo ""
	@echo "  $(CYAN)PRDS — Personal Research & Development System$(RESET)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(RESET) %s\n", $$1, $$2}'
	@echo ""

# ─── Development ──────────────────────────────────────────────────────────────
dev: ## Start backend with hot reload (requires air)
	cd backend && air -c .air.toml

dev-frontend: ## Serve frontend on port 3000
	python3 -m http.server 3000 --directory frontend

dev-all: ## Start all services (postgres, redis) + backend hot reload
	docker compose up postgres redis -d
	$(MAKE) dev

# ─── Build ────────────────────────────────────────────────────────────────────
build: ## Build backend binary
	cd backend && CGO_ENABLED=0 go build -ldflags="-w -s" -o ../bin/prds-server ./cmd/server
	@echo "$(GREEN)✓ Built: bin/prds-server$(RESET)"

build-docker: ## Build all Docker images
	docker compose build

# ─── Database ─────────────────────────────────────────────────────────────────
migrate: ## Run database migrations
	cd backend && go run -tags migrate cmd/server/main.go --migrate-only
	@echo "$(GREEN)✓ Migrations applied$(RESET)"

migrate-down: ## Rollback last migration
	migrate -path backend/migrations -database "$(DATABASE_URL)" down 1

seed: ## Seed database with exercises, concepts, papers
	psql "$(DATABASE_URL)" -f backend/migrations/002_seed_data.sql
	@echo "$(GREEN)✓ Data seeded$(RESET)"

db-shell: ## Open psql shell
	docker compose exec postgres psql -U prds -d prds

db-reset: ## Drop and recreate database (DESTRUCTIVE)
	docker compose exec postgres psql -U prds -c "DROP DATABASE IF EXISTS prds;"
	docker compose exec postgres psql -U prds -c "CREATE DATABASE prds;"
	$(MAKE) migrate seed

# ─── Testing ──────────────────────────────────────────────────────────────────
test: ## Run all tests
	cd backend && go test ./... -v -timeout 60s

test-cover: ## Run tests with coverage report
	cd backend && go test ./... -coverprofile=coverage.out
	cd backend && go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✓ Coverage report: backend/coverage.html$(RESET)"

test-integration: ## Run integration tests (requires running DB)
	cd backend && go test ./... -tags integration -v

# ─── Code Quality ─────────────────────────────────────────────────────────────
lint: ## Run golangci-lint
	cd backend && golangci-lint run ./...

fmt: ## Format Go code
	cd backend && gofmt -w .
	cd backend && goimports -w .

vet: ## Run go vet
	cd backend && go vet ./...

# ─── Docker ───────────────────────────────────────────────────────────────────
docker-up: ## Start all services
	docker compose up -d
	@echo "$(GREEN)✓ Services started$(RESET)"
	@echo "  Frontend: http://localhost:80"
	@echo "  Backend:  http://localhost:8080"
	@echo "  API Docs: http://localhost:8080/api/v1/docs"

docker-down: ## Stop all services
	docker compose down

docker-logs: ## Tail logs for all services
	docker compose logs -f

docker-ps: ## Show running services
	docker compose ps

monitoring-up: ## Start with Prometheus + Grafana
	docker compose --profile monitoring up -d
	@echo "$(GREEN)✓ Monitoring started$(RESET)"
	@echo "  Prometheus: http://localhost:9090"
	@echo "  Grafana:    http://localhost:3001 (admin/admin)"

# ─── Migration Script ─────────────────────────────────────────────────────────
migrate-ls: ## Export localStorage data for migration (run in browser first)
	@echo "Run this in your browser console to export data:"
	@echo ""
	@echo "  JSON.stringify({xp: STATE.xp, streak: STATE.streak,"
	@echo "    completed: [...STATE.completedExercises],"
	@echo "    inProgress: [...STATE.inProgressExercises],"
	@echo "    concepts: [...STATE.masteredConcepts],"
	@echo "    codes: Object.fromEntries(Object.keys(localStorage)"
	@echo "      .filter(k=>k.startsWith('code_'))"
	@echo "      .map(k=>[k,localStorage[k]])),"
	@echo "    notes: Object.fromEntries(Object.keys(localStorage)"
	@echo "      .filter(k=>k.startsWith('notes_'))"
	@echo "      .map(k=>[k,localStorage[k]])),"
	@echo "    sr: Object.fromEntries(Object.keys(localStorage)"
	@echo "      .filter(k=>k.startsWith('sr_'))"
	@echo "      .map(k=>[k,JSON.parse(localStorage[k])]))"
	@echo "  })"

# ─── Production ───────────────────────────────────────────────────────────────
prod-build: ## Build production images
	ENV=production docker compose build

prod-up: ## Start in production mode
	ENV=production docker compose up -d

backup: ## Backup PostgreSQL database
	docker compose exec postgres pg_dump -U prds prds | gzip > backup-$(shell date +%Y%m%d-%H%M%S).sql.gz
	@echo "$(GREEN)✓ Backup created$(RESET)"

restore: ## Restore from backup (BACKUP_FILE=./backup.sql.gz make restore)
	gunzip -c $(BACKUP_FILE) | docker compose exec -T postgres psql -U prds prds

# ─── Setup ────────────────────────────────────────────────────────────────────
setup: ## First-time setup
	@echo "$(CYAN)Setting up PRDS...$(RESET)"
	cp -n .env.example .env || true
	@echo "$(YELLOW)⚠ Edit .env with your configuration$(RESET)"
	cd backend && go mod download
	$(MAKE) docker-up
	@sleep 5
	$(MAKE) migrate
	$(MAKE) seed
	@echo ""
	@echo "$(GREEN)✓ PRDS is ready!$(RESET)"
	@echo "  Open: http://localhost:80"

clean: ## Remove build artifacts
	rm -rf bin/ backend/coverage.out backend/coverage.html
	docker compose down -v --remove-orphans
