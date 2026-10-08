#!/usr/bin/env bash
# ============================================================================
# aiops-agent 交叉构建脚本：产出 server/resource/agent/ 下的托管二进制
# (安装流按目标机架构选择文件名 aiops-agent-<ver>-linux-<arch>)。
#
# 用法: scripts/build-agent.sh [version]
#   version 缺省读 server/global/version.go 的 Version(去 v 前缀)。
#   dev 手放兜底: 无版本号文件名 aiops-agent-linux-<arch> 也被安装流识别。
# ============================================================================
set -euo pipefail

SERVER_DIR="$(cd "$(dirname "$0")/../server" && pwd)"
OUT_DIR="${SERVER_DIR}/resource/agent"
VERSION="${1:-}"
if [[ -z "${VERSION}" ]]; then
  VERSION="$(grep -oP 'Version\s*=\s*"\K[^"]+' "${SERVER_DIR}/global/version.go" | head -1 | tr -d 'v')"
fi
[[ -z "${VERSION}" ]] && VERSION="dev"

mkdir -p "${OUT_DIR}"
cd "${SERVER_DIR}"

for arch in amd64 arm64; do
  out="${OUT_DIR}/aiops-agent-${VERSION}-linux-${arch}"
  echo "→ 构建 linux/${arch} → ${out}"
  CGO_ENABLED=0 GOOS=linux GOARCH=${arch} go build \
    -trimpath -ldflags "-s -w -X main.agentVersion=${VERSION}" \
    -o "${out}" ./cmd/agent
done
echo "✓ 完成: $(ls -1 "${OUT_DIR}")"
