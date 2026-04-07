#!/usr/bin/env bash
# Push MarkItDown HTTP + docreader images to Alibaba Cloud Container Registry (ACR).
# 不含 Notex 镜像。
#
#   ./scripts/acr-push.sh latest
#   ACR_VERSION=v1.0.0 make acr-push
#
# 登录（密码为控制台「访问凭证」）:
#   docker login --username='<阿里云账号全名>' "${ACR_REGISTRY}"
#
# 命名空间下需已创建仓库: markitdown、docreader（见 scripts/acr.env.example）。
# RAM 子用户: 企业别名不能含英文句号「.」。
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
: "${ACR_REPO_MARKITDOWN:=markitdown}"
: "${ACR_REPO_DOCREADER:=docreader}"
: "${MARKITDOWN_LOCAL_IMAGE:=youmind/markitdown-http:local}"
: "${DOCREADER_LOCAL_IMAGE:=youmind/weknora-docreader:local}"

push_one() {
	local local_img=$1
	local remote_repo=$2
	local ver=$3
	local remote="${ACR_REGISTRY}/${ACR_NAMESPACE}/${remote_repo}:${ver}"
	if ! docker image inspect "${local_img}" >/dev/null 2>&1; then
		echo "error: local image not found: ${local_img}" >&2
		return 1
	fi
	echo "tag  ${local_img} -> ${remote}"
	docker tag "${local_img}" "${remote}"
	echo "push ${remote}"
	docker push "${remote}"
}

print_server_env() {
	local ver=$1
	local reg=$2
	echo ""
	echo "=== 服务器 .env（同版本 ${ver}；VPC 时替换 REGISTRY）==="
	echo "MARKITDOWN_IMAGE=${reg}/${ACR_NAMESPACE}/${ACR_REPO_MARKITDOWN}:${ver}"
	echo "DOCREADER_IMAGE=${reg}/${ACR_NAMESPACE}/${ACR_REPO_DOCREADER}:${ver}"
}

VERSION="${1:-}"
if [[ -z "${VERSION}" ]] || [[ "${VERSION}" == -* ]]; then
	echo "usage: $0 <镜像版本号>" >&2
	echo "example: $0 latest" >&2
	exit 1
fi

echo "[acr] docker compose build markitdown docreader ..."
docker compose build markitdown docreader

push_one "${MARKITDOWN_LOCAL_IMAGE}" "${ACR_REPO_MARKITDOWN}" "${VERSION}"
push_one "${DOCREADER_LOCAL_IMAGE}" "${ACR_REPO_DOCREADER}" "${VERSION}"
print_server_env "${VERSION}" "${ACR_REGISTRY}"
echo "[acr] push done (markitdown + docreader only)."
