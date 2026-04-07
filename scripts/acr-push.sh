#!/usr/bin/env bash
# Tag and push a local Notex image to Alibaba Cloud Container Registry (ACR).
#
# 1) 登录（密码为仓库密码，在控制台「访问凭证」设置）:
#    docker login --username='<阿里云账号全名>' "${ACR_REGISTRY}"
# 2) 构建本地镜像:
#    make notex-build
# 3) 推送:
#    cp scripts/acr.env.example scripts/acr.env   # 按需改 VPC 域名等
#    ./scripts/acr-push.sh v1.0.0
#
# RAM 子用户登录时，企业别名不能带英文句号「.」（阿里云限制）。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [[ -f scripts/acr.env ]]; then
	set -a
	# shellcheck disable=SC1091
	source scripts/acr.env
	set +a
fi

: "${ACR_REGISTRY:=crpi-mpi18r3iierw5366.cn-wulanchabu.personal.cr.aliyuncs.com}"
: "${ACR_NAMESPACE:=metanote}"
: "${ACR_REPO:=metanote}"
: "${NOTEX_LOCAL_IMAGE:=youmind/notex:local}"

VERSION="${1:-}"
if [[ -z "${VERSION}" ]] || [[ "${VERSION}" == -* ]]; then
	echo "usage: $0 <镜像版本号>" >&2
	echo "example: $0 v1.0.0" >&2
	exit 1
fi

REMOTE="${ACR_REGISTRY}/${ACR_NAMESPACE}/${ACR_REPO}:${VERSION}"

if ! docker image inspect "${NOTEX_LOCAL_IMAGE}" >/dev/null 2>&1; then
	echo "error: local image not found: ${NOTEX_LOCAL_IMAGE}" >&2
	echo "run: make notex-build   (or docker compose build notex)" >&2
	exit 1
fi

echo "tag  ${NOTEX_LOCAL_IMAGE} -> ${REMOTE}"
docker tag "${NOTEX_LOCAL_IMAGE}" "${REMOTE}"
echo "push ${REMOTE}"
docker push "${REMOTE}"
echo "done. on ECS: docker pull ${REMOTE}"
