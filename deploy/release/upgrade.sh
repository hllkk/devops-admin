#!/usr/bin/env bash
# ============================================================================
# devops-admin 手工升级脚本（在线升级的离线兜底；全量包/增量包均携带）
#
# 用法（在既有安装目录执行，docker-compose.yml 同级须已有 .env）:
#   bash upgrade.sh
#
# 行为（与 updater install job 同流程，但不经 docker.sock/不依赖 updater 容器）:
#   1. docker load 包内镜像（增量 images/*.tar 或全量 images.tar.gz）
#   2. 暂存生产 config/config.yaml（credential-key 机密，任何升级不得覆盖）
#   3. 覆盖编排资产（compose/.env.example/nginx/VERSION/BUILD_TIME/config 除 config.yaml）
#   4. .env 写入新版本号（APP_VERSION/BUILD_TIME）
#   5. compose 镜像完整性校验 → up -d --no-build --force-recreate web server updater
#   6. 等待健康检查
#
# 幂等：重复执行 = 再次升级/修复；回滚 = .env 的 APP_VERSION 改回旧版本号 + up -d --no-build
# ============================================================================
set -euo pipefail

log() { printf '\n\033[1;32m==> %s\033[0m\n' "$*"; }
warn() { printf '\033[1;33m[警告] %s\033[0m\n' "$*"; }
die() { printf '\033[1;31m[错误] %s\033[0m\n' "$*" >&2; exit 1; }
env_get() { grep -E "^$1=" .env 2>/dev/null | head -1 | cut -d= -f2-; }

cd "$(dirname "${BASH_SOURCE[0]}")"
[ -f .env ] || die "缺少 .env（须在既有安装目录执行，全新部署请用 install.sh）"
command -v docker >/dev/null || die "未安装 docker"
docker compose version >/dev/null 2>&1 || die "未安装 docker compose v2"
[ -f VERSION ] || die "缺少 VERSION（升级包不完整）"
NEW_VERSION=$(cat VERSION)

# ---- 1. 导入镜像（增量包 images/*.tar；全量包 images.tar.gz）----------------
log "导入镜像（版本 $NEW_VERSION）"
shopt -s nullglob
TARS=(images/*.tar)
shopt -u nullglob
if [ "${#TARS[@]}" -gt 0 ]; then
  for t in "${TARS[@]}"; do docker load -i "$t"; done
elif [ -f images.tar.gz ]; then
  docker load -i images.tar.gz
else
  die "未找到 images/*.tar 或 images.tar.gz（升级包不完整）"
fi

# ---- 2. 暂存生产机密配置 ------------------------------------------------------
log "暂存生产 config/config.yaml（credential-key 不可轮换，升级后原样还原）"
BAK_DIR=".upgrade-backup"
mkdir -p "$BAK_DIR"
[ -f config/config.yaml ] && mv config/config.yaml "$BAK_DIR/config.yaml"

# ---- 3. 覆盖编排资产（保守合并式：包内有什么覆盖什么，不删生产既有文件）-------
log "覆盖编排资产"
cp -f docker-compose.yml .env.example VERSION BUILD_TIME ./ 2>/dev/null || true
[ -d nginx ] && cp -rf nginx/. ./nginx/
if [ -d config ]; then
  mkdir -p ./config
  # 包内不含 config.yaml（机密资产）；其余配置文件逐个覆盖
  for f in config/*; do
    [ "$(basename "$f")" = "config.yaml" ] && continue
    cp -rf "$f" ./config/
  done
fi

# ---- 4. .env 写入新版本（回滚 = 改回旧值 + up -d --no-build）------------------
log "更新 .env 版本号 → $NEW_VERSION"
sed -i "s|^APP_VERSION=.*|APP_VERSION=$NEW_VERSION|" .env
if grep -q '^BUILD_TIME=' .env; then
  sed -i "s|^BUILD_TIME=.*|BUILD_TIME=$(cat BUILD_TIME)|" .env
else
  echo "BUILD_TIME=$(cat BUILD_TIME)" >> .env
fi

# ---- 5. 还原生产机密配置 ------------------------------------------------------
[ -f "$BAK_DIR/config.yaml" ] && mv "$BAK_DIR/config.yaml" config/config.yaml
rmdir "$BAK_DIR" 2>/dev/null || true

DC="docker compose -f docker-compose.yml --env-file .env"

# ---- 6. 镜像完整性校验（缺失立即失败，避免 up 卡在拉取超时）-------------------
log "校验镜像完整性"
MISSING=0
while IFS= read -r img; do
  [ -z "$img" ] && continue
  docker image inspect "$img" >/dev/null 2>&1 || { warn "镜像缺失: $img"; MISSING=1; }
done < <($DC config --images)
[ "$MISSING" -eq 0 ] || die "存在缺失镜像（第三方镜像缺失说明应改用全量包）"

# ---- 7. 重建自研服务（基础设施容器不动，数据面/转发面持续可用）----------------
log "重建 web/server/updater"
$DC up -d --no-build --force-recreate web server updater

# ---- 8. 健康检查 ---------------------------------------------------------------
WEB_PORT=$(env_get WEB_PORT); WEB_PORT=${WEB_PORT:-80}
log "等待服务健康（http://127.0.0.1:${WEB_PORT}/proxy-default/health）"
elapsed=0
until curl -sf -m 3 "http://127.0.0.1:${WEB_PORT}/proxy-default/health" >/dev/null 2>&1; do
  [ "$elapsed" -ge 120 ] && die "健康检查超时，排查: $DC logs --tail=50 server"
  sleep 3; elapsed=$((elapsed + 3))
done

cat <<EOF

============================================================
✅ 升级完成: $NEW_VERSION（页面「关于」弹窗可确认版本）
  回滚: 编辑 .env 的 APP_VERSION=<旧版本> && $DC up -d --no-build
============================================================
EOF
