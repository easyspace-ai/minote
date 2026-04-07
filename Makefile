# Youmind / minote — Docker Compose helpers (Notex + infra).
# Requires: Docker Compose v2, optional `docker compose watch` for rebuild-on-save (notex image).

COMPOSE ?= docker compose

.PHONY: help
help:
	@echo "Targets:"
	@echo "  make docker         - Server one-shot: seed .env/config if missing, pull, infra + Notex (profile app)"
	@echo "  make infra          - Start Postgres, Redis, MarkItDown, docreader, MinIO, … (no Notex)"
	@echo "  make up-app         - infra + Notex container (profile app)"
	@echo "  make dev-notex      - infra + notex-dev (Air hot-reload, profile dev; port NOTEX_DEV_HOST_PORT)"
	@echo "  make notex-build    - Build Notex production image only"
	@echo "  make notex-rebuild  - Rebuild and recreate Notex (app profile)"
	@echo "  make notex-restart  - Recreate Notex without rebuild"
	@echo "  make notex-logs     - Follow Notex logs"
	@echo "  make watch-notex    - Rebuild Notex image when cmd/internal/pkg change (Compose watch)"
	@echo "  make down           - docker compose down"
	@echo "  make acr-login-help - print ACR docker login hint (Aliyun registry)"
	@echo "  make acr-push-notex - build Notex + push to ACR (set ACR_VERSION=tag)"
	@echo "  make build-web      - Production Vite build -> bin/web"
	@echo "  make build-notex    - go build cmd/notex -> bin/notex"
	@echo "  make build          - build-web + build-notex"

.PHONY: build-web
build-web:
	cd web && pnpm install && pnpm run build

.PHONY: build-notex
build-notex:
	mkdir -p bin
	go build -o bin/notex ./cmd/notex

.PHONY: build
build: build-web build-notex

.PHONY: docker
docker:
	@test -f config.yaml || (echo "[docker] creating config.yaml from config.example.yaml" && cp config.example.yaml config.yaml)
	@test -f .env || (echo "[docker] creating .env from .env.example — set SILICONFLOW_API_KEY / LLM keys before relying on AI" && cp .env.example .env)
	@echo "[docker] pulling images (local-only tags may be skipped)"
	@$(COMPOSE) --profile app pull --ignore-pull-failures
	@echo "[docker] starting stack (infra + notex)"
	@$(COMPOSE) --profile app up -d

.PHONY: infra
infra:
	$(COMPOSE) up -d

.PHONY: up-app
up-app:
	@test -f config.yaml || (echo "Missing config.yaml — cp config.example.yaml config.yaml" && exit 1)
	$(COMPOSE) --profile app up -d

.PHONY: dev-notex
dev-notex:
	@test -f config.yaml || (echo "Missing config.yaml — cp config.example.yaml config.yaml" && exit 1)
	@$(COMPOSE) --profile app stop notex 2>/dev/null || true
	$(COMPOSE) --profile dev up -d notex-dev

.PHONY: notex-build
notex-build:
	$(COMPOSE) build notex

.PHONY: notex-rebuild
notex-rebuild:
	$(COMPOSE) --profile app up -d --build --force-recreate notex

.PHONY: notex-restart
notex-restart:
	$(COMPOSE) --profile app up -d --force-recreate notex

.PHONY: notex-logs
notex-logs:
	$(COMPOSE) --profile app logs -f notex

.PHONY: watch-notex
watch-notex:
	$(COMPOSE) --profile app watch notex

.PHONY: down
down:
	$(COMPOSE) down

# --- Aliyun ACR (see scripts/acr.env.example, scripts/acr-push.sh) ---
.PHONY: acr-login-help acr-push-notex
acr-login-help:
	@echo "docker login --username='<阿里云账号全名>' crpi-mpi18r3iierw5366.cn-wulanchabu.personal.cr.aliyuncs.com"
	@echo "VPC 内: 将域名换为 crpi-mpi18r3iierw5366-vpc.cn-wulanchabu.personal.cr.aliyuncs.com"
	@echo "可选: cp scripts/acr.env.example scripts/acr.env 后改 ACR_REGISTRY"

acr-push-notex: notex-build
	@test -n "$(ACR_VERSION)" || (echo 'Set ACR_VERSION, e.g. ACR_VERSION=v1.0.0 make acr-push-notex' && exit 1)
	@chmod +x scripts/acr-push.sh 2>/dev/null || true
	@./scripts/acr-push.sh "$(ACR_VERSION)"
