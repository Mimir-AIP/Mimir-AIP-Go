.PHONY: help build-orchestrator build-worker build-frontend build-all test lint clean

# Image registry prefix — override with: make build-all REGISTRY=ghcr.io/your-org
REGISTRY ?= mimir-aip
TAG      ?= latest

ORCHESTRATOR_IMAGE = $(REGISTRY)/orchestrator:$(TAG)
WORKER_IMAGE       = $(REGISTRY)/worker:$(TAG)
FRONTEND_IMAGE     = $(REGISTRY)/frontend:$(TAG)

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-25s\033[0m %s\n", $$1, $$2}'

# ── Build ──────────────────────────────────────────────────────────────────────

build-orchestrator: ## Build the orchestrator Docker image
	docker build -t $(ORCHESTRATOR_IMAGE) -f cmd/orchestrator/Dockerfile .

build-worker: ## Build the worker Docker image
	docker build -t $(WORKER_IMAGE) -f cmd/worker/Dockerfile .

build-frontend: ## Build the frontend Docker image
	docker build -t $(FRONTEND_IMAGE) -f frontend/Dockerfile .

build-all: build-orchestrator build-worker build-frontend ## Build all Docker images

# ── Push ───────────────────────────────────────────────────────────────────────

push-all: ## Push all images to the configured registry
	docker push $(ORCHESTRATOR_IMAGE)
	docker push $(WORKER_IMAGE)
	docker push $(FRONTEND_IMAGE)

# ── Test ───────────────────────────────────────────────────────────────────────

test: ## Run all unit tests
	go test ./pkg/...

lint: ## Run go vet
	go vet ./...

# ── Local dev ─────────────────────────────────────────────────────────────────

dev-orchestrator: ## Run the orchestrator locally (requires Go)
	go run ./cmd/orchestrator/main.go

dev-frontend: ## Run the frontend server locally
	cd frontend && PORT=3000 API_URL=http://localhost:8080 go run server.go

# ── Cleanup ───────────────────────────────────────────────────────────────────

clean: ## Remove locally built Docker images
	docker rmi $(ORCHESTRATOR_IMAGE) $(WORKER_IMAGE) $(FRONTEND_IMAGE) 2>/dev/null || true
