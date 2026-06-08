SHELL := /bin/bash
BACKEND_DIR := backend
FRONTEND_DIR := frontend

.PHONY: help run-infra up down logs build run-backend run-frontend run-seeder \
        seed-images seed-images-force test test-backend test-coverage test-frontend \
        lint lint-backend lint-frontend generate clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

## ---- Run the whole app (Docker Compose: backend + frontend + MinIO + Redis) ----
up: ## Build & start backend, frontend, MinIO and Redis, wait for health, then seed images
	@echo "==> Building images and starting containers (first run can take a few minutes)..."
	docker compose up -d --build
	@echo "==> Waiting for backend to become healthy at http://localhost:8080/healthz ..."
	@for i in $$(seq 1 40); do \
		if curl -fs http://localhost:8080/healthz >/dev/null 2>&1; then \
			echo "    backend is healthy."; break; \
		fi; \
		printf '.'; sleep 2; \
		if [ $$i -eq 40 ]; then echo; echo "    ERROR: backend did not become healthy in time. Check 'make logs'."; exit 1; fi; \
	done
	@echo "==> Seeding images into MinIO..."
	@$(MAKE) seed-images
	@echo ""
	@echo "=================================================================="
	@echo " Scout is up:"
	@echo "   Frontend:      http://localhost:5173"
	@echo "   Backend:       http://localhost:8080/healthz"
	@echo "   MinIO console: http://localhost:9001  (minioadmin / minioadmin)"
	@echo "=================================================================="

down: ## Stop and remove all services
	docker compose down

logs: ## Tail service logs
	docker compose logs -f

run-infra: ## Start ONLY infrastructure (MinIO + Redis) in Docker - run backend/frontend yourself
	docker compose up -d minio redis
	@echo ""
	@echo "Infrastructure is up:"
	@echo "  MinIO:   http://localhost:9000  (console: http://localhost:9001, minioadmin / minioadmin)"
	@echo "  Redis:   localhost:6379"
	@echo "Backend and frontend were NOT started. Run them locally, e.g. 'make run-backend'."

build: ## Build backend binaries locally
	cd $(BACKEND_DIR) && CGO_ENABLED=0 go build -o bin/scout ./cmd/scout
	cd $(BACKEND_DIR) && CGO_ENABLED=0 go build -o bin/seeder-images ./cmd/seeder-images

## ---- Run individual components locally (require Go and Node installed) ----
run-backend: ## Build the backend for the host OS, then run it locally
	$(MAKE) build
	cd $(BACKEND_DIR) && ./bin/scout

run-frontend: ## Run the frontend dev server locally
	cd $(FRONTEND_DIR) && yarn install && yarn dev

run-seeder: ## Run the image seeder locally
	cd $(BACKEND_DIR) && go run ./cmd/seeder-images

## ---- Seed images into MinIO through the backend presigned-upload API ----
seed-images: ## Upload images (skips ones that already exist)
	docker compose run --rm --no-deps -e SEEDER_API_BASE_URL=http://backend:8080 \
		--entrypoint /app/seeder-images backend

seed-images-force: ## Re-upload all images, overwriting existing objects
	docker compose run --rm --no-deps -e SEEDER_API_BASE_URL=http://backend:8080 \
		--entrypoint /app/seeder-images backend --force

## ---- Tests ----
test: test-backend test-frontend ## Run backend and frontend tests

test-backend: ## Run Go tests
	cd $(BACKEND_DIR) && go test ./...

test-coverage: ## Run Go tests with coverage and print a per-function + total summary
	cd $(BACKEND_DIR) && go test -race -coverprofile=coverage.out -covermode=atomic ./...
	cd $(BACKEND_DIR) && go tool cover -func=coverage.out | tail -n 1
	cd $(BACKEND_DIR) && go tool cover -html=coverage.out -o coverage.html
	@echo "HTML report: $(BACKEND_DIR)/coverage.html"

test-frontend: ## Run frontend (vitest) tests
	cd $(FRONTEND_DIR) && yarn install && yarn test

## ---- Linters ----
lint: lint-backend lint-frontend ## Run all linters

lint-backend: ## Run golangci-lint
	cd $(BACKEND_DIR) && golangci-lint run ./...

lint-frontend: ## Run ESLint
	cd $(FRONTEND_DIR) && yarn install && yarn lint

## ---- Code generation (easyjson DTOs) ----
generate: ## Generate easyjson marshalers for response DTOs
	cd $(BACKEND_DIR) && go run github.com/mailru/easyjson/easyjson -all \
		internal/interface/api/rest/dto/response.go

clean: ## Remove build artifacts
	rm -rf $(BACKEND_DIR)/bin $(FRONTEND_DIR)/dist
