.PHONY: up down logs test build

up:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f

logs-api:
	docker compose logs -f api

logs-web:
	docker compose logs -f web

test:
	go test ./... -count=1

build:
	go build -o ./tmp/api ./cmd/api
