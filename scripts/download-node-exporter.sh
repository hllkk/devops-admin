#!/usr/bin/env bash
# ============================================================================
# node_exporter 官方二进制下载到 server/resource/node-exporter/（安装流托管产物）。
# 多源回退：官方 release 直连 → gh-proxy 国内代理。
# 用法: scripts/download-node-exporter.sh [version]   # 缺省 1.9.1
# ============================================================================
set -euo pipefail

VER="${1:-1.9.1}"
SERVER_DIR="$(cd "$(dirname "$0")/../server" && pwd)"
OUT_DIR="${SERVER_DIR}/resource/node-exporter"
mkdir -p "${OUT_DIR}"

for arch in amd64 arm64; do
  file="node_exporter-${VER}.linux-${arch}.tar.gz"
  url="https://github.com/prometheus/node_exporter/releases/download/v${VER}/${file}"
  echo "→ 下载 ${arch} ..."
  if ! curl -fsSL --retry 2 -o "/tmp/${file}" "${url}"; then
    echo "→ 官方源失败,切换 gh-proxy ..."
    curl -fsSL --retry 3 -o "/tmp/${file}" "https://gh-proxy.com/${url}"
  fi
  tar -xzf "/tmp/${file}" -C /tmp
  mv "/tmp/node_exporter-${VER}.linux-${arch}/node_exporter" "${OUT_DIR}/node_exporter-${arch}"
  rm -rf "/tmp/${file}" "/tmp/node_exporter-${VER}.linux-${arch}"
  echo "✓ ${OUT_DIR}/node_exporter-${arch}"
done
