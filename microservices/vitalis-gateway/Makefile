GO?=go

.PHONY: tidy run test build docker-up docker-down

tidy:
	$(GO) mod tidy

run:
	$(GO) run ./cmd/api

test:
	$(GO) test ./...

build:
	$(GO) build -o bin/gateway ./cmd/api

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down
