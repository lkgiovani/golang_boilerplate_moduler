.PHONY: run build tidy migrate migrate-info test/unit test/integration test

DB_HOST     ?= localhost
DB_PORT     ?= 5432
DB_NAME     ?= boilerplate
DB_USER     ?= postgres
DB_PASSWORD ?= postgres
DB_JDBC_URL  = jdbc:postgresql://$(DB_HOST):$(DB_PORT)/$(DB_NAME)

run:
	go run ./cmd/api/...

build:
	go build -o bin/api ./cmd/api/...

tidy:
	go mod tidy

test/unit:
	go test -v -count=1 \
		./internal/modules/users/application/usersusecases/... \
		./internal/modules/health/application/healthusecases/...

test/integration:
	go test -v -count=1 -timeout 120s ./internal/test/integration/...

test: test/unit test/integration

migrate:
	docker run --rm --network host \
		-v $(PWD)/migrations:/flyway/sql \
		flyway/flyway:latest \
		-url=$(DB_JDBC_URL) \
		-user=$(DB_USER) \
		-password=$(DB_PASSWORD) \
		migrate

migrate-info:
	docker run --rm --network host \
		-v $(PWD)/migrations:/flyway/sql \
		flyway/flyway:latest \
		-url=$(DB_JDBC_URL) \
		-user=$(DB_USER) \
		-password=$(DB_PASSWORD) \
		info