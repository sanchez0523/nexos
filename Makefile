.PHONY: up down setup certs add-device check-prereqs \
        go-build go-test go-test-race go-lint go-fmt go-vet \
        svelte-dev svelte-build svelte-check svelte-install \
        migrate-up migrate-down migrate-new \
        logs logs-broker logs-ingestion ci

# Auto-load .env so targets that need its variables (e.g. migrate-*) work out
# of the box. If .env is missing, targets that need it will print a hint.
ifneq (,$(wildcard .env))
    include .env
    export
endif

# Derived DATABASE_URL for host-run migrations (container uses db:5432,
# host connects to localhost:5432 via the db container's exposed port).
HOST_DATABASE_URL ?= postgres://nexos:$(POSTGRES_PASSWORD)@localhost:5432/nexos?sslmode=disable

# ── Infrastructure ──────────────────────────────────────────────────────────

up: check-prereqs
	docker compose up -d

check-prereqs:
	@[ -f .env ] || (echo "✘ .env not found — run ./scripts/setup.sh first" && exit 1)
	@[ -f broker/passwd ] || (echo "✘ broker/passwd not found — run ./scripts/setup.sh first" && exit 1)
	@[ -f broker/certs/ca.crt ] || (echo "✘ broker/certs/ca.crt not found — run ./scripts/setup.sh first" && exit 1)
	@[ -f broker/certs/server.crt ] || (echo "✘ broker/certs/server.crt not found — run ./scripts/setup.sh first" && exit 1)
	@[ -f broker/certs/server.key ] || (echo "✘ broker/certs/server.key not found — run ./scripts/setup.sh first" && exit 1)
	@echo "✔ All prerequisites present"

down:
	docker compose down

setup:
	bash scripts/setup.sh

certs:
	bash scripts/setup.sh --certs-only

add-device:
	bash scripts/add-device.sh

# ── Go ───────────────────────────────────────────────────────────────────────

go-build:
	cd ingestion && go build ./cmd/server/...

go-test:
	cd ingestion && go test ./...

go-test-race:
	cd ingestion && go test -race -count=1 ./...

go-lint:
	cd ingestion && golangci-lint run ./...

go-fmt:
	cd ingestion && gofmt -w .

go-vet:
	cd ingestion && go vet ./...

# ── SvelteKit ────────────────────────────────────────────────────────────────

svelte-dev:
	cd dashboard && npm run dev

svelte-build:
	cd dashboard && npm run build

svelte-check:
	cd dashboard && npx svelte-check --threshold warning

svelte-install:
	cd dashboard && npm install

# ── Database migrations ──────────────────────────────────────────────────────

migrate-up:
	@[ -f .env ] || (echo "✘ .env not found — run ./scripts/setup.sh first" && exit 1)
	migrate -path ingestion/migrations -database "$(HOST_DATABASE_URL)" up

migrate-down:
	@[ -f .env ] || (echo "✘ .env not found — run ./scripts/setup.sh first" && exit 1)
	migrate -path ingestion/migrations -database "$(HOST_DATABASE_URL)" down 1

migrate-new:
	@[ -n "$(name)" ] || (echo "Usage: make migrate-new name=<migration_name>" && exit 1)
	migrate create -ext sql -dir ingestion/migrations -seq $(name)

# ── Logs ─────────────────────────────────────────────────────────────────────

logs:
	docker compose logs -f

logs-broker:
	docker compose logs -f broker

logs-ingestion:
	docker compose logs -f ingestion

# ── CI shortcut (mirrors GitHub Actions) ─────────────────────────────────────

ci: go-fmt go-vet go-lint go-test-race svelte-check
	@echo "✔ All CI checks passed"
