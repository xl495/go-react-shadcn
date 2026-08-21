.PHONY: server admin web dev test build migrate docker-up docker-down openapi-gen e2e

GO ?= $(shell test -x $(HOME)/go/bin/go1.24.0 && echo $(HOME)/go/bin/go1.24.0 || echo go)

server:
	cd server && $(GO) run ./cmd/server

admin:
	cd admin && npm run dev -- --host 127.0.0.1 --port 5173

web:
	cd web && npm run dev -- --host 127.0.0.1 --port 5174

dev:
	./scripts/dev.sh

test:
	cd server && $(GO) test ./...

build:
	cd admin && npm run build
	cd web && npm run build
	cd server && $(GO) build -o server ./cmd/server

migrate:
	cd server && DATABASE_PATH=./data/app.db $(GO) run ./cmd/migrate

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

openapi-gen:
	cd admin && npm run codegen:api

e2e:
	cd e2e && npm test
