.PHONY: api web site dev test build migrate docker-up docker-down openapi-gen

GO ?= $(shell test -x $(HOME)/go/bin/go1.24.0 && echo $(HOME)/go/bin/go1.24.0 || echo go)

api:
	cd backend && $(GO) run ./cmd/server

web:
	cd frontend && npm run dev -- --host 127.0.0.1 --port 5173

site:
	cd web && npm run dev -- --host 127.0.0.1 --port 5174

dev:
	./scripts/dev.sh

test:
	cd backend && $(GO) test ./...

build:
	cd frontend && npm run build
	cd web && npm run build
	cd backend && $(GO) build -o server ./cmd/server

migrate:
	cd backend && $(GO) run ./cmd/server --help >/dev/null 2>&1 || true
	cd backend && DATABASE_PATH=./data/app.db $(GO) run ./cmd/server

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

openapi-gen:
	cd frontend && npm run codegen:api
