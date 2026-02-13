# Sentinel Auth Project Automation

# Binary name (Use sentinel.exe if compiling strictly for Windows execution)
BINARY_NAME=sentinel
DOCKER_COMPOSE=docker-compose.yml

# .PHONY ensures these targets are run even if a file with the same name exists
.PHONY: all build test clean docker-up docker-down migrate-up

all: build test

build:
	@echo "Building binary..."
	go mod tidy
	go build -o bin/$(BINARY_NAME) cmd/api/main.go

test:
	@echo "Running unit tests..."
	go test -v ./pkg/security/... ./internal/usecase/...

test-integration:
	@echo "Running integration tests..."
	go test -v ./internal/repository/...

docker-up:
	@echo "Starting infrastructure..."
	docker-compose -f $(DOCKER_COMPOSE) up -d

docker-down:
	@echo "Stopping infrastructure..."
	docker-compose -f $(DOCKER_COMPOSE) down

migrate-up:
	@echo "Applying database schema..."
	@echo "Waiting for database to be ready..."
	# This command pipes the schema.sql file into the running postgres container
	docker exec -i sentinel-db psql -U user -d sentinel < schema.sql

clean:
	@echo "Cleaning up..."
	go clean
	rm -rf bin/