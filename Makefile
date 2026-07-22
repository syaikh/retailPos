.PHONY: help build-backend build-frontend build-all deploy start stop restart logs status clean test

help: ## Show this help message
	@echo 'Retail POS System - Commands:'
	@echo ''
	@echo '  make build-backend    Build backend Docker image'
	@echo '  make build-frontend   Build frontend Docker image'
	@echo '  make build-all        Build both images'
	@echo '  make test             Run unit and E2E tests'
	@echo '  make deploy           Deploy with Podman (start all services)'
	@echo '  make start            Alias for deploy'
	@echo '  make stop             Stop all services'
	@echo '  make restart          Restart all services'
	@echo '  make status           Check service status'
	@echo '  make logs             Show all logs'
	@echo '  make logs-backend     Show backend logs'
	@echo '  make logs-frontend    Show frontend logs'
	@echo '  make logs-db          Show database logs'
	@echo '  make clean            Stop and remove all containers/volumes'
	@echo '  make clean-images     Remove Docker images'
	@echo ''

# Build targets
build-backend: ## Build backend Docker image
	@echo "Building backend image..."
	podman build -t retail-pos-backend:latest -f deploy/backend/Dockerfile .

build-frontend: ## Build frontend Docker image
	@echo "Building frontend image..."
	podman build -t retail-pos-frontend:latest -f deploy/frontend/Dockerfile .

build-all: build-backend build-frontend ## Build both images

# Deployment targets
deploy: ## Deploy with Podman
	@chmod +x deploy/podman-deploy.sh
	./deploy/podman-deploy.sh start

start: deploy ## Alias for deploy

stop: ## Stop all services
	./deploy/podman-deploy.sh stop

restart: ## Restart all services
	./deploy/podman-deploy.sh restart

status: ## Check service status
	./deploy/podman-deploy.sh status

logs: ## Show all logs
	./deploy/podman-deploy.sh logs all

logs-backend: ## Show backend logs
	./deploy/podman-deploy.sh logs backend

logs-frontend: ## Show frontend logs
	./deploy/podman-deploy.sh logs frontend

logs-db: ## Show database logs
	./deploy/podman-deploy.sh logs postgres

# Development targets
dev-backend: ## Run backend locally (development)
	cd cmd/server && go run main.go

dev-frontend: ## Run frontend dev server
	cd web && npm run dev

# Testing targets
test: ## Run backend unit tests (set TEST_DB_* and JWT_SECRET in .env.test or shell)
	@echo "Running backend tests..."
ifneq ($(wildcard .env.test),)
	include .env.test
endif
	go test -p 1 -count=1 ./...

test-e2e: ## Run E2E tests only
	@echo "Ensure services are running: ./deploy/podman-deploy.sh start"
	npx playwright test --reporter=list

test-full: ## Run backend + E2E tests
	@echo "Running backend tests..."
ifneq ($(wildcard .env.test),)
	include .env.test
endif
	go test -p 1 -count=1 ./...
	@echo ""
	@echo "Running E2E tests (requires both servers running)..."
	npx playwright test --reporter=list

# Database targets
db-backup: ## Backup database to file
	@mkdir -p backups
	podman exec postgres pg_dump -U pos retail_pos > backups/backup_$$(date +%Y%m%d_%H%M%S).sql
	@echo "Backup saved to backups/"

db-restore: ## Restore database from file (usage: make db-restore FILE=backups/xxx.sql)
	podman exec -i postgres psql -U pos -d retail_pos < $(FILE)
	@echo "Restored from $(FILE)"

db-shell: ## Open psql shell
	podman exec -it postgres psql -U pos -d retail_pos

# Cleanup targets
clean: stop ## Stop and remove containers and volumes
	@echo "Removing volumes..."
	podman volume rm retail-pos-postgres-data 2>/dev/null || true
	@echo "Removing networks..."
	podman network rm retail-pos-network 2>/dev/null || true
	@echo "Clean complete!"

clean-images: ## Remove Docker images
	podman rmi retail-pos-backend retail-pos-frontend 2>/dev/null || true

# Kubernetes targets (future)
k8s-generate: ## Generate Kubernetes manifest from pod
	podman generate kube retail-pos-pod --name retail-pos > k8s/deployment.yaml
	@echo "K8s manifest generated at k8s/deployment.yaml"

# Helpful aliases
.PHONY: ps
ps: status ## Alias for status
