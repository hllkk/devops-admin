#!/usr/bin/env bash
# ============================================================================
# devops-admin 离线部署包构建脚本（在构建机/当前开发服务器执行）
#
# 产物：deploy/release/out/devops-admin-release-<时间戳>.tar.gz
#   ├── images.tar.gz            # 全部运行镜像（目标机无需外网拉取）
#   ├── docker-compose.yml       # 生产编排（仓库版）
#   ├── .env                     # 本次部署的强随机机密（构建时生成）
#   ├── nginx/ config/           # nginx 与后端/PG/Redis/LiteLLM 配置
#   ├── install.sh               # 目标机一键部署脚本
#   └── README-DEPLOY.md         # 部署说明
#
# 用法：bash deploy/release/build-release.sh
# 流程：compose build 应用镜像 → 校验/收集 7 个镜像 → 生成强随机 .env
#       → 注入 credential-key → tar 打包
# 幂等：可重复执行，每次产出新时间戳的包；.env 机密每次重新生成
# ============================================================================
set -euo pipefail

REPO_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
PROD_DIR="$REPO_ROOT/deploy/docker-prod"
RELEASE_DIR="$REPO_ROOT/deploy/release"
OUT_DIR="$RELEASE_DIR/out"
STAMP=$(date +%Y%m%d-%H%M)
PKG_NAME="devops-admin-release-$STAMP.tar.gz"
STAGE=$(mktemp -d)
trap 'rm -rf "$STAGE"' EXIT

# ---- 本次部署参数（按目标环境调整）----
DEPLOY_IP=172.21.96.171          # 目标服务器（用于下发 LITELLM_PUBLIC_URL）
WEB_PORT=80                      # web 对外端口
LITELLM_HOST_PORT=4001           # litellm 宿主映射端口（宿主 4000 已被旧 litellm 占用，过渡用 4001；
                                 # 旧实例停用后改 .env 的 LITELLM_HOST_PORT=4000 即切回）

# ---- 镜像清单（须与 docker-compose.yml / .env 引用的 tag 完全一致）----
IMAGES=(
  "devops-admin/web:prod"
  "devops-admin/server:prod"
  "postgres:18-alpine"
  "redis:8-alpine"
  "rustfs/rustfs:latest"
  "minio/mc:latest"
  "ghcr.io/berriai/litellm:1.99.0"
)

log() { printf '\n\033[1;32m==> %s\033[0m\n' "$*"; }
die() { printf '\033[1;31m[错误] %s\033[0m\n' "$*" >&2; exit 1; }

cd "$REPO_ROOT"

# ============================================================================
# 1. 构建应用镜像（web / server）
# ============================================================================
log "构建应用镜像（web + server）"
docker compose -f deploy/docker-prod/docker-compose.yml build

# ============================================================================
# 2. 校验镜像清单完整
# ============================================================================
log "校验镜像清单"
for img in "${IMAGES[@]}"; do
  docker image inspect "$img" >/dev/null 2>&1 || die "镜像不存在: $img（先 docker pull 或检查 tag）"
  printf '  ✓ %s\n' "$img"
done

# ============================================================================
# 3. 生成强随机 .env（生产机密，不落仓库）
# ============================================================================
log "生成生产 .env（强随机机密）"
gen_hex() { openssl rand -hex "$1"; }
# 密码用 base64 去符号取字母数字，避免 shell/env 特殊字符转义问题
gen_pw() { openssl rand -base64 24 | tr -dc 'A-Za-z0-9' | head -c 16; }

ADMIN_PASSWORD=$(gen_pw)
PG_PASSWORD=$(gen_hex 16)
REDIS_PASSWORD=$(gen_hex 16)
RUSTFS_PASSWORD=$(gen_pw)
LITELLM_MASTER_KEY="sk-$(gen_hex 32)"
CREDENTIAL_KEY=$(gen_hex 32)

sed -e "s|^WEB_PORT=.*|WEB_PORT=$WEB_PORT|" \
    -e "s|^SERVER_PORT=8888|# SERVER_PORT=8888  # 生产不对外暴露后端，仅经 web 反代|" \
    -e "s|^JWT_SIGNING_KEY=.*|JWT_SIGNING_KEY=$(gen_hex 32)|" \
    -e "s|^INIT_ADMIN_PASSWORD=.*|INIT_ADMIN_PASSWORD=$ADMIN_PASSWORD|" \
    -e "s|^POSTGRES_PASSWORD=.*|POSTGRES_PASSWORD=$PG_PASSWORD|" \
    -e "s|^REDIS_PASSWORD=.*|REDIS_PASSWORD=$REDIS_PASSWORD|" \
    -e "s|^RUSTFS_ROOT_PASSWORD=.*|RUSTFS_ROOT_PASSWORD=$RUSTFS_PASSWORD|" \
    -e "s|^LITELLM_MASTER_KEY=.*|LITELLM_MASTER_KEY=$LITELLM_MASTER_KEY|" \
    -e "s|^LITELLM_SALT_KEY=.*|LITELLM_SALT_KEY=$(gen_hex 32)|" \
    -e "s|^LITELLM_HOST_PORT=.*|LITELLM_HOST_PORT=$LITELLM_HOST_PORT|" \
    -e "s|^LITELLM_PUBLIC_URL=.*|LITELLM_PUBLIC_URL=http://$DEPLOY_IP:$LITELLM_HOST_PORT|" \
    "$PROD_DIR/.env.example" > "$STAGE/.env"

# credential-key 无 env 覆盖机制（other.go 未实现），构建时注入包内 config.yaml：
# AES-256-GCM 64 位 hex；留空会导致 AI 网关凭证拒绝写入
cp -r "$PROD_DIR/config" "$STAGE/config"
sed -i "s|^litellm:|litellm:\n    credential-key: $CREDENTIAL_KEY|" "$STAGE/config/config.yaml"

# ============================================================================
# 4. 组装部署包
# ============================================================================
log "组装部署包"
cp "$PROD_DIR/docker-compose.yml" "$STAGE/"
cp -r "$PROD_DIR/nginx" "$STAGE/"
cp "$RELEASE_DIR/install.sh" "$STAGE/"
cp "$RELEASE_DIR/README-DEPLOY.md" "$STAGE/" 2>/dev/null || true

# ============================================================================
# 5. 导出镜像
# ============================================================================
log "导出镜像（docker save | gzip，约 1-2GB，请耐心等待）"
mkdir -p "$OUT_DIR"
docker save "${IMAGES[@]}" | gzip > "$STAGE/images.tar.gz"
du -sh "$STAGE/images.tar.gz"

# ============================================================================
# 6. 打包
# ============================================================================
log "打包 $PKG_NAME"
tar -czf "$OUT_DIR/$PKG_NAME" -C "$STAGE" .
du -sh "$OUT_DIR/$PKG_NAME"

cat <<EOF

============================================================
部署包已生成: $OUT_DIR/$PKG_NAME

本次部署机密（.env 已入包，此摘要仅备查，请勿外传）:
  初始管理员密码 INIT_ADMIN_PASSWORD: $ADMIN_PASSWORD
  LITELLM_HOST_PORT                 : $LITELLM_HOST_PORT（4000 被旧实例占用，过渡端口）
  LITELLM_PUBLIC_URL                : http://$DEPLOY_IP/llm

传输与部署:
  scp $OUT_DIR/$PKG_NAME root@$DEPLOY_IP:/root/
  ssh root@$DEPLOY_IP
  mkdir -p /root/devops-admin && tar -xzf /root/$PKG_NAME -C /root/devops-admin
  cd /root/devops-admin && bash install.sh
============================================================
EOF
