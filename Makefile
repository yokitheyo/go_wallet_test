.PHONY: help build up down restart logs clean test load-test

SERVICE_NAME = wallet-service
DB_NAME = postgres

help:
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build Docker image
	docker-compose build

up: ## Start containers
	docker-compose up -d

down: ## Stop containers
	docker-compose down

restart: ## Restart containers
	docker-compose restart

logs: ## Show container logs
	docker-compose logs -f

clean: ## Stop containers and remove volumes
	docker-compose down -v

test: ## Run tests
	@if [ -f "go.mod" ]; then \
		go test ./...; \
	else \
		echo "No tests found"; \
	fi

load-test: ## Run load test
	@echo "Running load test..."
	hey -z 5s -q 1000 -c 100 -m POST \
		-H "Content-Type: application/json" \
		-d '{"walletId":"22222222-4312-1234-7777-222332222222","operationType":"DEPOSIT","amount":1}' \
		http://localhost:8080/api/v1/wallet

load-test-short: ## Run short load test
	@echo "Running short load test..."
	hey -z 1s -q 1000 -c 100 -m POST \
		-H "Content-Type: application/json" \
		-d '{"walletId":"22222222-4312-1234-7777-222332222222","operationType":"DEPOSIT","amount":1}' \
		http://localhost:8080/api/v1/wallet

load-test-all: ## Run all load tests
	@echo "Running all load tests..."
	@$(MAKE) load-test
	@sleep 2
	@$(MAKE) load-test-short

health-check: ## Check service health
	@curl -f http://localhost:8080/health || echo "Service unavailable"

install-hey: ## Install hey for load testing
	@if command -v hey >/dev/null 2>&1; then \
		echo "hey is already installed"; \
	else \
		go install github.com/rakyll/hey@latest; \
	fi
