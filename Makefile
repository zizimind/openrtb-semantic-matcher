.PHONY: all build run test clean docker-up docker-down proto mock-data lint

# Variables
BINARY_NAME=gateway
GO=go
DOCKER_COMPOSE=docker-compose -f deployments/docker-compose.yml

# ============================================
# Build Commands
# ============================================

all: build

build:
	$(GO) build -o bin/$(BINARY_NAME) ./cmd/gateway

build-enricher:
	cd python/enricher && pip install -r requirements.txt

run: build
	./bin/$(BINARY_NAME)

# ============================================
# Development
# ============================================

dev:
	$(GO) run ./cmd/gateway

mock-data:
	python3 scripts/generate_mock_ads.py

proto:
	protoc --go_out=. --go-grpc_out=. api/proto/*.proto

lint:
	golangci-lint run ./...

test:
	$(GO) test -v -race ./...

test-coverage:
	$(GO) test -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

# ============================================
# Docker Commands
# ============================================

docker-up:
	$(DOCKER_COMPOSE) up -d

docker-down:
	$(DOCKER_COMPOSE) down

docker-build:
	$(DOCKER_COMPOSE) build

docker-logs:
	$(DOCKER_COMPOSE) logs -f

docker-ps:
	$(DOCKER_COMPOSE) ps

# ============================================
# Load Testing
# ============================================

load-test:
	@echo "🧪 Running Load Test..."
	docker run --rm -i \
		-v "$(PWD)/scripts":/scripts \
		--add-host=host.docker.internal:host-gateway \
		grafana/k6 run /scripts/load_test.js

# ============================================
# Cleanup
# ============================================

clean:
	rm -rf bin/
	rm -f coverage.out coverage.html
	$(GO) clean

# ============================================
# Setup
# ============================================

setup:
	cp configs/.env.example .env
	$(GO) mod download
	$(GO) mod tidy
