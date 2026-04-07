# Youmind / minote — Docker Compose 辅助命令
#
# 本地开发 (Mac):
#   make docker              # 启动基础服务 (本地构建)
#   make acr-push-both       # 构建并推送镜像到阿里云 ACR
#
# 服务器部署 (阿里云 ECS):
#   1. 在服务器上配置 .env (从 .env.server.example 复制)
#   2. ./scripts/deploy-server.sh 或 make server-deploy
#
# 详见: docker-compose.yml, docker-compose.prod.yml, scripts/acr-push.sh

COMPOSE ?= docker compose

.PHONY: help
help:
	@echo "本地开发命令:"
	@echo "  make docker              - 启动基础服务 (Postgres, Redis, MinIO, Milvus)"
	@echo "  make infra               - 同上 (docker compose up -d)"
	@echo "  make down                - 停止所有服务"
	@echo "  make logs                - 查看服务日志"
	@echo ""
	@echo "镜像构建与推送 (Mac -> 阿里云 ACR):"
	@echo "  make acr-push            - 仅推送 markitdown (ACR_VERSION=tag)"
	@echo "  make acr-push-docreader  - 仅推送 docreader (较慢)"
	@echo "  make acr-push-both       - 推送 markitdown + docreader"
	@echo ""
	@echo "服务器部署 (阿里云 ECS):"
	@echo "  make server-deploy       - 在服务器上部署 (需先配置 .env)"
	@echo "  make server-update       - 在服务器上更新镜像"
	@echo "  make server-status       - 查看服务器服务状态"
	@echo ""
	@echo "应用构建:"
	@echo "  make build-web           - 构建前端 (Vite)"
	@echo "  make build-notex         - 构建 Notex 后端"
	@echo "  make build               - 构建前端 + 后端"
	@echo "  make notex-build         - 构建 Notex Docker 镜像 (本地)"
	@echo ""
	@echo "辅助:"
	@echo "  make acr-login-help      - 显示 ACR 登录帮助"
	@echo "  make clean               - 清理构建产物和卷"

# =============================================================================
# 本地开发
# =============================================================================

.PHONY: docker
docker:
	@test -f config.yaml || (echo "[docker] 创建 config.yaml" && cp config.example.yaml config.yaml)
	@test -f .env || (echo "[docker] 创建 .env" && cp .env.example .env)
	@echo "[docker] 启动基础服务..."
	@$(COMPOSE) up -d
	@echo "[docker] 服务已启动"
	@$(COMPOSE) ps

.PHONY: infra
infra:
	$(COMPOSE) up -d

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f

# =============================================================================
# 镜像构建与推送 (Mac -> 阿里云 ACR)
# =============================================================================

.PHONY: acr-login-help acr-push acr-push-docreader acr-push-both acr-push-all

acr-login-help:
	@echo "ACR 登录命令:"
	@echo "  docker login --username='<阿里云账号全名>' crpi-mpi18r3iierw5366.cn-wulanchabu.personal.cr.aliyuncs.com"
	@echo ""
	@echo "VPC 内网登录 (ECS 同区域):"
	@echo "  docker login --username='<阿里云账号全名>' crpi-mpi18r3iierw5366-vpc.cn-wulanchabu.personal.cr.aliyuncs.com"
	@echo ""
	@echo "可选配置: cp scripts/acr.env.example scripts/acr.env 后修改 ACR_REGISTRY"

acr-push:
	@test -n "$(ACR_VERSION)" || (echo '错误: 请设置 ACR_VERSION，例如: ACR_VERSION=v1.0.0 make acr-push' && exit 1)
	@chmod +x scripts/acr-push.sh 2>/dev/null || true
	@./scripts/acr-push.sh "$(ACR_VERSION)"

acr-push-docreader:
	@test -n "$(ACR_VERSION)" || (echo '错误: 请设置 ACR_VERSION，例如: ACR_VERSION=v1.0.0 make acr-push-docreader' && exit 1)
	@chmod +x scripts/acr-push.sh 2>/dev/null || true
	@./scripts/acr-push.sh docreader "$(ACR_VERSION)"

acr-push-both:
	@test -n "$(ACR_VERSION)" || (echo '错误: 请设置 ACR_VERSION，例如: ACR_VERSION=v1.0.0 make acr-push-both' && exit 1)
	@chmod +x scripts/acr-push.sh 2>/dev/null || true
	@./scripts/acr-push.sh both "$(ACR_VERSION)"

# 兼容旧命令
acr-push-all: acr-push-both

# =============================================================================
# 服务器部署 (在阿里云 ECS 上执行)
# =============================================================================

.PHONY: server-deploy server-update server-stop server-status server-logs server-config

server-deploy:
	@chmod +x scripts/deploy-server.sh 2>/dev/null || true
	@./scripts/deploy-server.sh start

server-update:
	@chmod +x scripts/deploy-server.sh 2>/dev/null || true
	@./scripts/deploy-server.sh update

server-stop:
	@chmod +x scripts/deploy-server.sh 2>/dev/null || true
	@./scripts/deploy-server.sh stop

server-status:
	@chmod +x scripts/deploy-server.sh 2>/dev/null || true
	@./scripts/deploy-server.sh status

server-logs:
	@chmod +x scripts/deploy-server.sh 2>/dev/null || true
	@./scripts/deploy-server.sh logs

server-config:
	@chmod +x scripts/deploy-server.sh 2>/dev/null || true
	@./scripts/deploy-server.sh config

# =============================================================================
# 应用构建
# =============================================================================

.PHONY: build-web build-notex build notex-build

build-web:
	cd web && pnpm install && pnpm run build

build-notex:
	mkdir -p bin
	go build -o bin/notex ./cmd/notex

build: build-web build-notex

notex-build:
	docker build -f docker/notex/Dockerfile -t youmind/notex:local .

# =============================================================================
# 清理
# =============================================================================

.PHONY: clean clean-all

clean:
	$(COMPOSE) down -v

clean-all: clean
	docker system prune -f
