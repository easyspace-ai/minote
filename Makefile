# Youmind / minote — Docker Compose helpers (infra only; Notex runs outside compose).

COMPOSE ?= docker compose

.PHONY: help
help:
	@echo "Targets:"
	@echo "  make docker         - Seed .env/config if missing, pull, start infra (Postgres, Redis, …)"
	@echo "  make infra          - Same stack: docker compose up -d"
	@echo "  make notex-build    - docker build Notex 镜像（本机 docker run 用；不在 ACR 脚本里）"
	@echo "  make down           - docker compose down"
	@echo "  make acr-login-help - print ACR docker login hint (Aliyun registry)"
	@echo "  make acr-push       - build markitdown+docreader + push 到 ACR（ACR_VERSION=tag）"
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
	@test -f .env || (echo "[docker] creating .env from .env.example — set LLM keys & compose ports" && cp .env.example .env)
	@echo "[docker] pulling images (local-only tags may be skipped)"
	@$(COMPOSE) pull --ignore-pull-failures
	@echo "[docker] starting infra"
	@$(COMPOSE) up -d

.PHONY: infra
infra:
	$(COMPOSE) up -d

.PHONY: notex-build
notex-build:
	docker build -f docker/notex/Dockerfile -t youmind/notex:local .

.PHONY: down
down:
	$(COMPOSE) down

# --- Aliyun ACR (see scripts/acr.env.example, scripts/acr-push.sh) — 仅 markitdown + docreader ---
.PHONY: acr-login-help acr-push
acr-login-help:
	@echo "docker login --username='<阿里云账号全名>' crpi-mpi18r3iierw5366.cn-wulanchabu.personal.cr.aliyuncs.com"
	@echo "VPC 内: 将域名换为 crpi-mpi18r3iierw5366-vpc.cn-wulanchabu.personal.cr.aliyuncs.com"
	@echo "可选: cp scripts/acr.env.example scripts/acr.env 后改 ACR_REGISTRY"

acr-push:
	@test -n "$(ACR_VERSION)" || (echo 'Set ACR_VERSION, e.g. ACR_VERSION=latest make acr-push' && exit 1)
	@chmod +x scripts/acr-push.sh 2>/dev/null || true
	@./scripts/acr-push.sh "$(ACR_VERSION)"

# 兼容旧命令（与 acr-push 相同，不再包含 notex）
.PHONY: acr-push-all
acr-push-all: acr-push
