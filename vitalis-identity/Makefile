APP=vitalis-identity
GO?=go

.PHONY: tidy run-api run-worker test migrate-up migrate-down docker-up docker-down

tidy:
	$(GO) mod tidy

run-api:
	$(GO) run ./cmd/api

run-worker:
	$(GO) run ./cmd/worker

test:
	$(GO) test ./...

migrate-up:
	goose -dir ./migrations postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir ./migrations postgres "$(DATABASE_URL)" down

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down -v

