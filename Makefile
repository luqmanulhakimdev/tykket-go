.PHONY: dev build test lint docker-up docker-down migrate-up migrate-down migrate-create migrate-version seed tidy

# Load .env if present
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# Development
dev:
	go run cmd/api/main.go

build:
	@mkdir -p bin
	go build -o bin/api cmd/api/main.go
	go build -o bin/migrate cmd/migrate/main.go

tidy:
	go mod tidy

test:
	go test -v -race ./...

# Docker operations
docker-up:
	docker compose up -d --build

docker-db:
	docker compose up -d postgres

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

docker-seed:
	docker compose run --rm api /app/seed

# Database migrations & seeds
migrate-up:
	go run cmd/migrate/main.go up

migrate-down:
	go run cmd/migrate/main.go down

migrate-version:
	go run cmd/migrate/main.go version

migrate-create:
	@if [ -z "$(name)" ]; then echo "Error: Migration name required. Example: make migrate-create name=add_tags"; exit 1; fi
	go run cmd/migrate/main.go create $(name)

seed:
	go run cmd/seed/main.go

