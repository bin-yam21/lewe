TEST_DATABASE_URL ?= postgres://lewe:lewe@localhost:5432/lewe_test?sslmode=disable

.PHONY: run build test test-unit lint db-up db-down

run:
	go run ./cmd/api

build:
	go build -o bin/lewe-api ./cmd/api

# Full suite, including end-to-end tests against PostgreSQL.
test:
	TEST_DATABASE_URL=$(TEST_DATABASE_URL) go test -race ./...

# Unit tests only (end-to-end tests skip without TEST_DATABASE_URL).
test-unit:
	go test ./...

lint:
	gofmt -l . | (! grep .) || (echo "gofmt needed on the files above" && exit 1)
	go vet ./...

# Start PostgreSQL in Docker and create the test database.
db-up:
	docker compose up -d db
	@until docker compose exec -T db pg_isready -U lewe >/dev/null 2>&1; do sleep 1; done
	-docker compose exec -T db createdb -U lewe lewe_test

db-down:
	docker compose down
