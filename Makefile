.PHONY: run build tidy migrate migrate-info test/unit test/integration test

# ─── Config ────────────────────────────────────────────────────────────────────
DB_HOST     ?= localhost
DB_PORT     ?= 5432
DB_NAME     ?= boilerplate
DB_USER     ?= postgres
DB_PASSWORD ?= postgres
DB_JDBC_URL  = jdbc:postgresql://$(DB_HOST):$(DB_PORT)/$(DB_NAME)

# ─── Development ───────────────────────────────────────────────────────────────

## run: start the API server (loads .env automatically)
run:
	go run ./cmd/api/...

## build: compile to a binary
build:
	go build -o bin/api ./cmd/api/...

## tidy: sync go.mod and go.sum
tidy:
	go mod tidy

# ─── Tests ─────────────────────────────────────────────────────────────────────

## test/unit: run use-case unit tests (no Docker required)
test/unit:
	go test -v -count=1 \
		./internal/modules/users/application/usecases/... \
		./internal/modules/health/application/usecases/...

## test/integration: spin up PostgreSQL via testcontainers and run integration tests
test/integration:
	go test -v -count=1 -timeout 120s ./internal/test/integration/...

## test: run unit tests then integration tests
test: test/unit test/integration

# ─── Migrations (Flyway via Docker) ────────────────────────────────────────────

## migrate: run pending Flyway migrations
migrate:
	docker run --rm --network host \
		-v $(PWD)/migrations:/flyway/sql \
		flyway/flyway:latest \
		-url=$(DB_JDBC_URL) \
		-user=$(DB_USER) \
		-password=$(DB_PASSWORD) \
		migrate

## migrate-info: show migration status
migrate-info:
	docker run --rm --network host \
		-v $(PWD)/migrations:/flyway/sql \
		flyway/flyway:latest \
		-url=$(DB_JDBC_URL) \
		-user=$(DB_USER) \
		-password=$(DB_PASSWORD) \
		info
