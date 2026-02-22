.PHONY: help build-orchestrator build-frontend build-all deploy-k8s clean delete-k8s logs-orchestrator logs-frontend port-forward

# Configuration
REGISTRY ?= mimir-aip
ORCHESTRATOR_IMAGE = $(REGISTRY)/orchestrator:latest
FRONTEND_IMAGE = $(REGISTRY)/frontend:latest
NAMESPACE = mimir-aip

help: ## Display this help message
	@echo "Mimir AIP - Build and Deployment Commands"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# Docker Build Commands
build-orchestrator: ## Build the orchestrator Docker image
	@echo "Building orchestrator image..."
	docker build -t $(ORCHESTRATOR_IMAGE) -f cmd/orchestrator/Dockerfile .

build-frontend: ## Build the frontend Docker image
	@echo "Building frontend image..."
	docker build -t $(FRONTEND_IMAGE) -f frontend/Dockerfile .

build-all: build-orchestrator build-frontend ## Build all Docker images
	@echo "All images built successfully!"

# Kubernetes Deployment Commands
deploy-k8s: ## Deploy all components to Kubernetes
	@echo "Deploying to Kubernetes..."
	kubectl apply -f k8s/development/00-namespace.yaml
	kubectl apply -f k8s/development/01-config.yaml
	kubectl apply -f k8s/development/02-pvc.yaml
	kubectl apply -f k8s/development/03-orchestrator.yaml
	kubectl apply -f k8s/development/04-frontend.yaml
	@echo "Waiting for deployments to be ready..."
	kubectl wait --for=condition=available --timeout=300s deployment/orchestrator -n $(NAMESPACE)
	kubectl wait --for=condition=available --timeout=300s deployment/frontend -n $(NAMESPACE)
	@echo "Deployment complete!"

delete-k8s: ## Delete all Kubernetes resources
	@echo "Deleting Kubernetes resources..."
	kubectl delete -f k8s/development/04-frontend.yaml --ignore-not-found=true
	kubectl delete -f k8s/development/03-orchestrator.yaml --ignore-not-found=true
	kubectl delete -f k8s/development/02-pvc.yaml --ignore-not-found=true
	kubectl delete -f k8s/development/01-config.yaml --ignore-not-found=true
	kubectl delete -f k8s/development/00-namespace.yaml --ignore-not-found=true
	@echo "Kubernetes resources deleted!"

redeploy: delete-k8s build-all deploy-k8s ## Full rebuild and redeploy

# Logs and Monitoring
logs-orchestrator: ## View orchestrator logs
	kubectl logs -f -n $(NAMESPACE) -l component=orchestrator

logs-frontend: ## View frontend logs
	kubectl logs -f -n $(NAMESPACE) -l component=frontend

logs-all: ## View all logs
	kubectl logs -f -n $(NAMESPACE) -l app=mimir-aip --all-containers=true

# Port Forwarding
port-forward-frontend: ## Forward frontend port to localhost:3000
	@echo "Forwarding frontend to http://localhost:3000"
	kubectl port-forward -n $(NAMESPACE) svc/frontend 3000:80

port-forward-orchestrator: ## Forward orchestrator port to localhost:8080
	@echo "Forwarding orchestrator to http://localhost:8080"
	kubectl port-forward -n $(NAMESPACE) svc/orchestrator 8080:8080

# Status and Info
status: ## Check deployment status
	@echo "=== Namespace ==="
	kubectl get namespace $(NAMESPACE)
	@echo ""
	@echo "=== Deployments ==="
	kubectl get deployments -n $(NAMESPACE)
	@echo ""
	@echo "=== Pods ==="
	kubectl get pods -n $(NAMESPACE)
	@echo ""
	@echo "=== Services ==="
	kubectl get services -n $(NAMESPACE)
	@echo ""
	@echo "=== PVCs ==="
	kubectl get pvc -n $(NAMESPACE)

describe-orchestrator: ## Describe orchestrator deployment
	kubectl describe deployment orchestrator -n $(NAMESPACE)

describe-frontend: ## Describe frontend deployment
	kubectl describe deployment frontend -n $(NAMESPACE)

# Development Commands
dev-frontend: ## Run frontend locally
	cd frontend && PORT=3000 API_URL=http://localhost:8080 go run server.go

dev-orchestrator: ## Run orchestrator locally
	cd cmd/orchestrator && go run main.go

# Cleanup
clean: ## Clean up local Docker images
	docker rmi $(ORCHESTRATOR_IMAGE) || true
	docker rmi $(FRONTEND_IMAGE) || true

# Quick access
open-frontend: ## Open frontend in browser (requires port-forward to be running)
	@echo "Opening http://localhost:3000 in browser..."
	@open http://localhost:3000|| xdg-open http://localhost:3000 || echo "Please open http://localhost:3000 in your browser"
