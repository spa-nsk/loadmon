.PHONY: build test test-db test-integration lint migrate migrate-down migrate-status seed-nc2026 seed-guide2019 run run-worker up down vulncheck

build:
	go build -o bin/cmdb-web ./cmd/cmdb-web
	go build -o bin/cmdb-worker ./cmd/cmdb-worker
	go build -o bin/migrate ./cmd/migrate

test:
	go test ./...

lint:
	golangci-lint run ./...

vulncheck:
	govulncheck ./...

migrate:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

migrate-status:
	go run ./cmd/migrate status

seed-nc2026:
	go run ./cmd/seed nc2026

seed-guide2019:
	go run ./cmd/seed guide2019

run:
	go run ./cmd/cmdb-web

run-worker:
	go run ./cmd/cmdb-worker

up:
	docker compose up -d

down:
	docker compose down

# Интеграционные тесты пишут тестовые КЕ, поэтому идут в отдельную БД cmdb_test, а не в рабочую cmdb.
TEST_DSN ?= postgres://cmdb:cmdb@localhost:5432/cmdb_test?sslmode=disable

test-db:
	@docker compose exec -T postgres psql -U cmdb -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='cmdb_test'" | grep -q 1 \
		|| docker compose exec -T postgres psql -U cmdb -d postgres -c "CREATE DATABASE cmdb_test"
	CMDB_DB_DSN='$(TEST_DSN)' go run ./cmd/migrate up
	CMDB_DB_DSN='$(TEST_DSN)' go run ./cmd/seed nc2026

test-integration: test-db
	CMDB_TEST_DB_DSN='$(TEST_DSN)' go test -count=1 ./...
