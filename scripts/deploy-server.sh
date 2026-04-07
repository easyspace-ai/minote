#!/usr/bin/env bash
# =============================================================================
# 服务器部署脚本 (在阿里云服务器上执行)
#
# 用途:
#   - 从 ACR 拉取最新镜像并部署服务
#   - 简化服务器上的操作流程
#
# 前置条件:
#   1. 已配置 .env 文件 (从 .env.server.example 复制并修改)
#   2. 已 docker login 到阿里云 ACR
#   3. 已安装 Docker 和 Docker Compose
#
# 使用方法:
#   ./scripts/deploy-server.sh          # 启动服务
#   ./scripts/deploy-server.sh update   # 拉取最新镜像并更新
#   ./scripts/deploy-server.sh stop     # 停止服务
#   ./scripts/deploy-server.sh logs     # 查看日志
#   ./scripts/deploy-server.sh status   # 查看状态
# =============================================================================
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

COMPOSE_FILES="-f docker-compose.yml -f docker-compose.prod.yml"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查环境
check_env() {
    if [[ ! -f .env ]]; then
        log_error "未找到 .env 文件"
        log_info "请复制 .env.server.example 到 .env 并配置:"
        log_info "  cp .env.server.example .env"
        exit 1
    fi

    # 检查必需的环境变量
    local required_vars=("MARKITDOWN_IMAGE" "DOCREADER_IMAGE")
    local missing_vars=()

    for var in "${required_vars[@]}"; do
        if ! grep -q "^${var}=" .env || grep -q "^${var}=.*<版本号>" .env; then
            missing_vars+=("$var")
        fi
    done

    if [[ ${#missing_vars[@]} -gt 0 ]]; then
        log_error ".env 中以下变量未配置或包含占位符:"
        for var in "${missing_vars[@]}"; do
            log_error "  - ${var}"
        done
        log_info "请确保设置了正确的 ACR 镜像地址"
        exit 1
    fi

    # 检查 Docker 登录状态
    if ! docker info 2>/dev/null | grep -q "Username"; then
        log_warn "未检测到 Docker 登录状态"
        log_info "请先登录阿里云 ACR:"
        log_info "  docker login --username='<阿里云账号>' crpi-mpi18r3iierw5366.cn-wulanchabu.personal.cr.aliyuncs.com"
    fi
}

# 启动服务
cmd_start() {
    log_info "启动服务..."
    docker compose ${COMPOSE_FILES} up -d
    log_success "服务已启动"
    log_info "查看状态: docker compose ${COMPOSE_FILES} ps"
    log_info "查看日志: docker compose ${COMPOSE_FILES} logs -f"
}

# 更新服务
cmd_update() {
    log_info "拉取最新镜像..."
    docker compose ${COMPOSE_FILES} pull

    log_info "重启服务..."
    docker compose ${COMPOSE_FILES} up -d

    log_success "服务已更新"
    cmd_status
}

# 停止服务
cmd_stop() {
    log_info "停止服务..."
    docker compose ${COMPOSE_FILES} down
    log_success "服务已停止"
}

# 查看日志
cmd_logs() {
    docker compose ${COMPOSE_FILES} logs -f --tail=100 "$@"
}

# 查看状态
cmd_status() {
    echo ""
    echo "========================================="
    echo "服务状态"
    echo "========================================="
    docker compose ${COMPOSE_FILES} ps

    echo ""
    echo "========================================="
    echo "资源使用"
    echo "========================================="
    docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.Status}}" 2>/dev/null | grep -E "(NAME|youmind-)" || true
}

# 查看配置
cmd_config() {
    log_info "当前配置 (MARKITDOWN_IMAGE 和 DOCREADER_IMAGE):"
    grep -E "^(MARKITDOWN_IMAGE|DOCREADER_IMAGE)=" .env
}

# 显示帮助
cmd_help() {
    echo "服务器部署脚本"
    echo ""
    echo "用法: $0 [命令]"
    echo ""
    echo "命令:"
    echo "  start    启动服务 (默认)"
    echo "  update   拉取最新镜像并更新服务"
    echo "  stop     停止服务"
    echo "  logs     查看日志"
    echo "  status   查看服务状态"
    echo "  config   查看当前镜像配置"
    echo "  help     显示帮助"
    echo ""
    echo "示例:"
    echo "  $0              # 启动服务"
    echo "  $0 update       # 更新到最新镜像"
    echo "  $0 logs -f      # 实时查看日志"
}

main() {
    local cmd="${1:-start}"

    case "${cmd}" in
    start|up)
        check_env
        cmd_start
        ;;
    update|pull)
        check_env
        cmd_update
        ;;
    stop|down)
        cmd_stop
        ;;
    logs)
        shift || true
        cmd_logs "$@"
        ;;
    status|ps)
        cmd_status
        ;;
    config)
        cmd_config
        ;;
    help|-h|--help)
        cmd_help
        ;;
    *)
        log_error "未知命令: ${cmd}"
        cmd_help
        exit 1
        ;;
    esac
}

main "$@"
