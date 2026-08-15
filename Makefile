.PHONY: help setup setup-db setup-env deps dev dev-backend dev-frontend build build-backend build-frontend test minio kill-ports install-frontend fix-crlf run-prod-backend run-prod-frontend deploy

SHELL := /bin/bash
ROOT := $(shell pwd)
export PATH := /usr/local/go/bin:$(PATH)

BACKEND_PORT ?= $(shell grep -E '^PORT=' backend/.env 2>/dev/null | tail -1 | cut -d= -f2- | tr -d '\r')
BACKEND_PORT := $(if $(BACKEND_PORT),$(BACKEND_PORT),8084)
FRONTEND_PORT ?= 3008

help:
	@echo "KasQ — Makefile (WSL/Linux)"
	@echo ""
	@echo "  make setup        Install deps + copy .env files"
	@echo "  make setup-db     Verify/create PostgreSQL database"
	@echo "  make fix-crlf     Fix Windows CRLF in Makefile (run once if bash errors)"
	@echo "  make deps         go mod tidy + npm install"
	@echo "  make dev          Run backend + frontend"
	@echo "  make dev-backend  Run Go server only"
	@echo "  make dev-frontend Run SvelteKit dev server only"
	@echo "  make minio        Start MinIO locally"
	@echo "  make test         Run backend tests"
	@echo "  make build        Build backend + frontend"
	@echo "  make deploy       Build lokal + upload artefak ke server (deploy/deploy.env)"

setup: setup-env deps

setup-env:
	@if [[ ! -f backend/.env ]]; then cp backend/.env.example backend/.env && echo "Created backend/.env"; else echo "backend/.env already exists, skipped"; fi
	@if [[ ! -f frontend/.env ]]; then cp frontend/.env.example frontend/.env && echo "Created frontend/.env"; else echo "frontend/.env already exists, skipped"; fi
	@echo "Edit backend/.env if PostgreSQL/MinIO credentials differ."

setup-db:
	@set -euo pipefail; \
	ENV_FILE="$(ROOT)/backend/.env"; \
	if [[ ! -f "$$ENV_FILE" ]]; then echo "ERROR: backend/.env belum ada. Jalankan: make setup"; exit 1; fi; \
	DATABASE_URL="$$(grep -E '^DATABASE_URL=' "$$ENV_FILE" | tail -1 | cut -d= -f2- | tr -d '\r' | sed -e 's/^"//' -e 's/"$$//' -e "s/^'//" -e "s/'$$//")"; \
	if [[ -z "$$DATABASE_URL" ]]; then \
		echo "ERROR: DATABASE_URL belum diset di backend/.env"; exit 1; \
	fi; \
	DB_NAME="$$(echo "$$DATABASE_URL" | sed -E 's|.*/([^/?]+)(\\?.*)?$$|\1|')"; \
	echo "==> KasQ database setup"; \
	echo "    Database: $$DB_NAME"; \
	echo "==> Mengecek koneksi database..."; \
	if psql "$$DATABASE_URL" -c "SELECT 1" >/dev/null 2>&1; then \
		echo "==> OK — koneksi berhasil, database sudah siap."; \
		exit 0; \
	fi; \
	echo "==> Database belum ada, mencoba buat '$$DB_NAME'..."; \
	PG_SUPERUSER="$${PG_SUPERUSER:-postgres}"; \
	PG_HOST="$${PG_HOST:-localhost}"; \
	PG_PORT="$${PG_PORT:-5432}"; \
	if [[ -n "$${PGPASSWORD:-}" ]]; then \
		psql -U "$$PG_SUPERUSER" -h "$$PG_HOST" -p "$$PG_PORT" -d postgres -v ON_ERROR_STOP=1 -c "CREATE DATABASE $$DB_NAME;" 2>/dev/null || true; \
	elif sudo -u postgres psql -v ON_ERROR_STOP=1 -c "CREATE DATABASE $$DB_NAME;" 2>/dev/null; then \
		:; \
	else \
		echo "ERROR: Gagal buat database. Buat manual: sudo -u postgres psql -c \"CREATE DATABASE $$DB_NAME;\""; \
		exit 1; \
	fi; \
	if psql "$$DATABASE_URL" -c "SELECT 1" >/dev/null 2>&1; then \
		echo "==> Done — database '$$DB_NAME' siap."; \
	else \
		echo "ERROR: Database dibuat tapi koneksi masih gagal."; exit 1; \
	fi

fix-crlf:
	@sed -i 's/\r$$//' Makefile 2>/dev/null || true
	@sed -i 's/\r$$//' backend/.env 2>/dev/null || true
	@echo "Done. Coba lagi: make setup-db"

install-frontend:
	@rm -rf frontend/node_modules
	@cd frontend && npm install
	@echo "Done. Run: make dev"

kill-ports:
	@for port in $(BACKEND_PORT) $(FRONTEND_PORT); do fuser -k "$$port/tcp" 2>/dev/null || true; done
	@pkill -f "go run.*cmd/server" 2>/dev/null || true
	@pkill -f "vite dev" 2>/dev/null || true
	@sleep 1
	@echo "Ports cleared."

deps:
	cd backend && go mod tidy
	cd frontend && rm -rf node_modules && npm install

deps-frontend:
	cd frontend && rm -rf node_modules package-lock.json && npm install

dev:
	@set -euo pipefail; \
	export BACKEND_PORT="$(BACKEND_PORT)"; \
	$(MAKE) --no-print-directory kill-ports; \
	echo "==> Starting backend on :$$BACKEND_PORT"; \
	(cd backend && go mod tidy && go run ./cmd/server) & \
	BACKEND_PID=$$!; \
	cleanup() { if kill -0 $$BACKEND_PID 2>/dev/null; then echo ""; echo "Stopping backend (pid $$BACKEND_PID)..."; kill $$BACKEND_PID 2>/dev/null || true; fi; }; \
	trap cleanup EXIT INT TERM; \
	echo "    Waiting for backend..."; \
	for i in $$(seq 1 15); do \
		code=$$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:$$BACKEND_PORT/api/me" || echo "000"); \
		if [[ "$$code" == "401" || "$$code" == "200" ]]; then echo "    Backend ready on :$$BACKEND_PORT"; break; fi; \
		if ! kill -0 $$BACKEND_PID 2>/dev/null; then echo "ERROR: Backend crashed. Cek log di atas."; exit 1; fi; \
		sleep 1; \
	done; \
	echo "==> Starting frontend on :$(FRONTEND_PORT) (proxy /api -> :$$BACKEND_PORT)"; \
	cd frontend && exec npm run dev -- --host 0.0.0.0 --port $(FRONTEND_PORT)

dev-backend:
	cd backend && go run ./cmd/server

dev-frontend:
	cd frontend && npm run dev -- --host 0.0.0.0 --port $(FRONTEND_PORT)

minio:
	@set -euo pipefail; \
	MINIO_BIN="$${MINIO_BIN:-$$HOME/.local/bin/minio}"; \
	MINIO_DATA="$${MINIO_DATA:-$$HOME/minio-data}"; \
	MINIO_ROOT_USER="$${MINIO_ROOT_USER:-minioadmin}"; \
	MINIO_ROOT_PASSWORD="$${MINIO_ROOT_PASSWORD:-minioadmin}"; \
	mkdir -p "$$MINIO_DATA" "$$(dirname "$$MINIO_BIN")"; \
	if [[ ! -x "$$MINIO_BIN" ]]; then \
		echo "==> Downloading MinIO for Linux amd64..."; \
		curl -fsSL https://dl.min.io/server/minio/release/linux-amd64/minio -o "$$MINIO_BIN"; \
		chmod +x "$$MINIO_BIN"; \
	fi; \
	echo "==> Starting MinIO"; \
	echo "    API:     http://localhost:9000"; \
	echo "    Console: http://localhost:9001  (login: $$MINIO_ROOT_USER/$$MINIO_ROOT_PASSWORD)"; \
	echo "    Data:    $$MINIO_DATA"; \
	echo ""; \
	echo "Buat bucket 'kasq' via console setelah MinIO jalan."; \
	export MINIO_ROOT_USER MINIO_ROOT_PASSWORD; \
	exec "$$MINIO_BIN" server "$$MINIO_DATA" --console-address ":9001"

test:
	cd backend && go test ./...

build: build-backend build-frontend

build-backend:
	cd backend && go build -o kasq-server ./cmd/server

build-frontend:
	cd frontend && npm run build

run-prod-backend:
	cd backend && ./kasq-server

run-prod-frontend:
	cd frontend && PORT=$(FRONTEND_PORT) node build/index.js

deploy:
	@sed -i 's/\r$$//' deploy/deploy.sh 2>/dev/null || true
	@bash deploy/deploy.sh
