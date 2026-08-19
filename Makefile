# SS Multi-State Tax Compliance Engine — one-command build/run/test/migrate.
# Most targets shell into docker-compose so the whole stack behaves identically
# on any machine.

# Load .env if present so DATABASE_URL etc. are available to local targets.
ifneq (,$(wildcard .env))
include .env
export
endif

COMPOSE := docker compose

.PHONY: up down build migrate seed test logs fmt help

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'

up: ## Bring the whole stack online (db + backend + frontend)
	$(COMPOSE) up --build -d
	@echo "Frontend: http://localhost:5174   API: http://localhost:8080"

down: ## Stop and remove all containers
	$(COMPOSE) down

build: ## Build backend + frontend images
	$(COMPOSE) build

migrate: ## Run DB migrations inside the backend container
	$(COMPOSE) run --rm backend /bin/server migrate

seed: ## Load the tax_rules table for the current tax year
	$(COMPOSE) run --rm backend /bin/server seed

test: ## Run Go engine unit tests
	cd backend && go test ./...

logs: ## Tail all container logs
	$(COMPOSE) logs -f

fmt: ## Format Go + frontend code
	cd backend && go fmt ./...
	cd frontend && npm run format --if-present
