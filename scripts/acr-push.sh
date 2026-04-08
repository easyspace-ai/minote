#!/usr/bin/env bash
# =============================================================================
# 阿里云 ACR 镜像构建和推送脚本 (支持 Mac/ARM 本机构建 x86/AMD64 镜像)
#
# 用途:
#   - 在 Mac (ARM) 上构建 linux/amd64 镜像并推送到阿里云 ACR
#   - 服务器无需构建，直接从 ACR 拉取运行
#
# 前置条件:
#   1. 已安装 Docker Desktop 并启用 buildx
#   2. 已 docker login 到阿里云 ACR
#   3. 已创建 scripts/acr.env (从 acr.env.example 复制)
#
# 登录 ACR:
#   docker login --username='<阿里云账号>' crpi-mpi18r3iierw5366.cn-wulanchabu.personal.cr.aliyuncs.com
#
# 使用方法:
#   ./scripts/acr-push.sh latest                    # 仅推送 markitdown
#   ./scripts/acr-push.sh docreader latest          # 仅推送 docreader
#   ./scripts/acr-push.sh both latest               # 推送两个
#   ACR_VERSION=latest make acr-push                # 使用 Makefile
#
# 服务器部署 (推送到 ACR 后在服务器执行):
#   1. 在服务器上: cp .env.server.example .env
#   2. 编辑 .env，填写正确的镜像标签
#   3. docker login 到 ACR
#   4. docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
#
# 常见问题:
#   Q: 构建时提示 platform 不匹配
#   A: 脚本已设置 DOCKER_DEFAULT_PLATFORM=linux/amd64，确保构建的镜像是 x86 格式
#
#   Q: 服务器上拉取后提示 exec format error
#   A: 检查服务器 .env 中的镜像地址是否正确，然后重新 pull
# =============================================================================
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# 加载 ACR 配置
if [[ -f scripts/acr.env ]]; then
    set -a
    # shellcheck disable=SC1091
    source scripts/acr.env
    set +a
fi

# 默认值
: "${ACR_REGISTRY:=crpi-mpi18r3iierw5366.cn-wulanchabu.personal.cr.aliyuncs.com}"
: "${ACR_NAMESPACE:=metanote}"
: "${ACR_REPO_MARKITDOWN:=markitdown}"
: "${ACR_REPO_DOCREADER:=docreader}"
: "${MARKITDOWN_LOCAL_IMAGE:=youmind/markitdown-http:local}"
: "${DOCREADER_LOCAL_IMAGE:=youmind/weknora-docreader:local}"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

# 检查 docker buildx 是否可用
check_buildx() {
    if ! docker buildx version &>/dev/null; then
        log_error "Docker buildx 不可用。请安装 Docker Desktop 或启用 buildx 插件。"
        exit 1
    fi

    # 检查 Docker Desktop 是否启用了 Rosetta 或 containerd 镜像存储
    log_info "检查 Docker 多架构支持..."
    if ! docker buildx ls | grep -q "linux/amd64"; then
        log_warn "当前 Docker 可能不支持 linux/amd64 平台"
        log_info "请确保 Docker Desktop 设置中启用了："
        log_info "  Settings -> Features -> Use Rosetta for x86/amd64 emulation on Apple Silicon"
    fi
}

# 构建并推送单个镜像
build_and_push() {
    local name=$1
    local dockerfile=$2
    local context=$3
    local local_img=$4
    local repo=$5
    local version=$6
    local remote="${ACR_REGISTRY}/${ACR_NAMESPACE}/${repo}:${version}"

    log_info "构建 ${name} (linux/amd64)..."
    log_info "  Dockerfile: ${dockerfile}"
    log_info "  Context: ${context}"
    log_info "  本地镜像: ${local_img}"
    log_info "  远程镜像: ${remote}"

    # 确保使用正确的 builder
    local builder_name="youmind-builder"
    if ! docker buildx ls | grep -q "${builder_name}"; then
        log_info "创建 buildx builder: ${builder_name}"
        docker buildx create --name "${builder_name}" --driver docker-container --bootstrap 2>/dev/null || true
    fi

    # 使用 buildx 构建并推送多架构镜像
    # 注意: --load 不支持多架构，直接使用 --push 推送到远程
    docker buildx build \
        --builder "${builder_name}" \
        --platform linux/amd64 \
        --file "${dockerfile}" \
        --tag "${remote}" \
        --push \
        "${context}"

    log_success "构建并推送完成: ${remote}"

    # 为了方便本地测试，也构建一个本地镜像（单架构）
    log_info "构建本地测试镜像..."
    docker buildx build \
        --platform linux/amd64 \
        --file "${dockerfile}" \
        --tag "${local_img}" \
        --load \
        "${context}" 2>/dev/null || log_warn "本地镜像构建失败（可忽略，远程已推送）"
}

# 构建并推送 markitdown
push_markitdown() {
    local version=$1
    build_and_push \
        "markitdown" \
        "docker/markitdown-http/Dockerfile" \
        "docker/markitdown-http" \
        "${MARKITDOWN_LOCAL_IMAGE}" \
        "${ACR_REPO_MARKITDOWN}" \
        "${version}"
}

# 构建并推送 docreader
push_docreader() {
    local version=$1
    build_and_push \
        "docreader" \
        "docker/weknora/Dockerfile.docreader" \
        "docker/weknora" \
        "${DOCREADER_LOCAL_IMAGE}" \
        "${ACR_REPO_DOCREADER}" \
        "${version}"
}

# 打印服务器环境变量配置
print_server_env() {
    local version=$1

    echo ""
    echo "==================================================================="
    echo "服务器 .env 配置 (复制到服务器的 .env 文件中)"
    echo "==================================================================="
    echo ""
    echo "# 阿里云 ACR 镜像地址"
    echo "MARKITDOWN_IMAGE=${ACR_REGISTRY}/${ACR_NAMESPACE}/${ACR_REPO_MARKITDOWN}:${version}"
    echo "DOCREADER_IMAGE=${ACR_REGISTRY}/${ACR_NAMESPACE}/${ACR_REPO_DOCREADER}:${version}"
    echo ""
    echo "# 基础服务端口 (根据服务器环境调整)"
    echo "MARKITDOWN_PORT=8787"
    echo "DOCREADER_PORT=50051"
    echo "DB_HOST_PORT=5432"
    echo "REDIS_HOST_PORT=6379"
    echo "MINIO_PORT=9000"
    echo "MINIO_CONSOLE_PORT=9001"
    echo "MILVUS_PORT=19530"
    echo ""
    echo "# 数据库配置"
    echo "DB_USER=postgres"
    echo "DB_PASSWORD=<设置强密码>"
    echo "DB_NAME=youmind"
    echo "POSTGRES_URL=postgres://postgres:<密码>@localhost:5432/youmind?sslmode=disable"
    echo ""
    echo "# 其他服务 URL"
    echo "MARKITDOWN_URL=http://127.0.0.1:8787"
    echo "DOCREADER_ADDR=127.0.0.1:50051"
    echo "REDIS_ADDR=127.0.0.1:6379"
    echo ""
    echo "==================================================================="
    echo "服务器部署命令:"
    echo "  docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d"
    echo "==================================================================="
}

usage() {
    echo "用法: $0 <tag>                 # 仅推送 markitdown"
    echo "      $0 docreader <tag>       # 仅推送 docreader"
    echo "      $0 both <tag>            # 推送 markitdown + docreader"
    echo ""
    echo "示例:"
    echo "      $0 v1.0.0"
    echo "      $0 docreader v1.0.0"
    echo "      $0 both latest"
    exit 1
}

main() {
    # 解析参数
    local MODE=markitdown
    local VERSION=""

    if [[ "${1:-}" == "docreader" ]]; then
        MODE=docreader
        VERSION="${2:-}"
    elif [[ "${1:-}" == "both" ]]; then
        MODE=both
        VERSION="${2:-}"
    elif [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
        usage
    else
        VERSION="${1:-}"
    fi

    if [[ -z "${VERSION}" || "${VERSION}" == -* ]]; then
        usage
    fi

    # 检查环境
    check_buildx

    # 确保已登录 ACR
    if ! docker info 2>/dev/null | grep -q "Username"; then
        log_warn "未检测到 docker 登录状态，请确保已运行:"
        log_warn "  docker login --username='<阿里云账号>' ${ACR_REGISTRY}"
    fi

    log_info "开始构建和推送 (版本: ${VERSION}, 平台: linux/amd64)"
    echo ""

    case "${MODE}" in
    markitdown)
        push_markitdown "${VERSION}"
        print_server_env "${VERSION}"
        log_success "完成! (markitdown)"
        ;;
    docreader)
        push_docreader "${VERSION}"
        print_server_env "${VERSION}"
        log_success "完成! (docreader)"
        ;;
    both)
        push_markitdown "${VERSION}"
        echo ""
        push_docreader "${VERSION}"
        print_server_env "${VERSION}"
        log_success "完成! (markitdown + docreader)"
        ;;
    *)
        usage
        ;;
    esac
}

main "$@"
