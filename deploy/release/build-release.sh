#!/usr/bin/env bash
# ============================================================================
# devops-admin 生产 Release 打包（在构建机/当前开发服务器执行）
#
# 用法：
#   bash build-release.sh [版本号]    # 默认 <global.Version>-<git短sha>，如 v0.2.0-a1b2c3d
#
# 产物（deploy/release/dist/，均附 .sha256）：
#   devops-admin-release-<版本>.tar.gz   全量包：8 镜像 + 强随机 .env + install.sh
#                                        （全新部署 / 离线手工升级，流程与既有约定一致）
#   devops-admin-upgrade-<版本>.tar.gz   增量包：仅自研 3 镜像 + 编排资产 + upgrade.sh
#                                        （在线升级载体：updater 拉取安装，或手工解压跑 upgrade.sh）
#   manifest-<版本>.json                 版本清单：版本/changelog/双包 sha256/包类型，
#                                        上传发布服务器改名 manifest.json 生效（changelog 需手工编辑）
#
# 镜像版本化：自研镜像 tag = APP_VERSION（devops-admin/web|server|updater:<版本>），三处同源注入——
#             server ldflags 进二进制 / web vite define+index.html meta 进前端产物；
#             旧版本镜像不被升级覆盖，保留本地可随时回滚。
#
# 安全：
#   - 全量包 .env 由构建机生成强随机机密（既有约定）；构建/校验阶段的临时 env
#     仅用于通过 compose 的 :? 强制校验，随机值不进任何产物
#   - 增量包不带 .env，也不带 config/config.yaml（生产 credential-key 构建期注入，
#     轮换会使历史加密凭证不可解——生产现值只能留在生产，任何包都不得携带/覆盖）
# ============================================================================
set -euo pipefail

RELEASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$RELEASE_DIR/../.." && pwd)"
PROD_DIR="$REPO_ROOT/deploy/docker-prod"

# ---- 本次部署参数（按目标环境调整，仅全量包 .env 使用）----
DEPLOY_IP=172.21.96.171          # 目标服务器（用于下发 LITELLM_PUBLIC_URL）
WEB_PORT=80                      # web 对外端口
LITELLM_HOST_PORT=4001           # litellm 宿主映射端口（宿主 4000 已被旧 litellm 占用，过渡用 4001；
                                 # 旧实例停用后改 .env 的 LITELLM_HOST_PORT=4000 即切回）

# ---- 版本号：显式参数 > global.Version-<git短sha> ----
VERSION="${1:-}"
if [ -z "$VERSION" ]; then
  BASE_VERSION="$(sed -n 's/^.*Version = "\([^"]*\)".*$/\1/p' "$REPO_ROOT/server/global/version.go" | head -1)"
  BASE_VERSION=${BASE_VERSION:-v0.0.0}
  GIT_SHA="$(cd "$REPO_ROOT" && git rev-parse --short HEAD 2>/dev/null || echo nogit)"
  VERSION="$BASE_VERSION-$GIT_SHA"
fi
BUILD_TIME="$(date +%Y-%m-%dT%H:%M:%S%z)"

DIST_DIR="$RELEASE_DIR/dist"
STAGING="$RELEASE_DIR/.staging"
FULL_STAGE="$STAGING/devops-admin-release"
INCR_STAGE="$STAGING/devops-admin-upgrade"

log() { printf '\n\033[1;32m==> %s\033[0m\n' "$*"; }
die() { printf '\033[1;31m[错误] %s\033[0m\n' "$*" >&2; exit 1; }

# ---- 环境检查 ---------------------------------------------------------------
command -v docker >/dev/null || die "未安装 docker"
docker info >/dev/null 2>&1 || die "docker daemon 不可用"
docker compose version >/dev/null 2>&1 || die "未安装 docker compose v2"
command -v openssl >/dev/null || die "缺 openssl（生成构建期随机值）"

rm -rf "$FULL_STAGE" "$INCR_STAGE"
mkdir -p "$FULL_STAGE" "$INCR_STAGE" "$DIST_DIR"

# ---- 构建期临时 env：通过 compose 的 :? 强制校验（随机值不进产物）-----------
BUILD_ENV="$STAGING/.build-env"
cleanup() { rm -rf "$STAGING"; }
trap cleanup EXIT

gen_hex() { openssl rand -hex "$1"; }
gen_pw() { openssl rand -base64 24 | tr -dc 'A-Za-z0-9' | head -c 16; }

# 版本注入：APP_VERSION=发布版本号（同时成为自研镜像 tag），经 compose build.args → ldflags 进二进制
sed -e "s|^JWT_SIGNING_KEY=.*|JWT_SIGNING_KEY=$(gen_hex 32)|" \
    -e "s|^INIT_ADMIN_PASSWORD=.*|INIT_ADMIN_PASSWORD=$(gen_pw)|" \
    -e "s|^POSTGRES_PASSWORD=.*|POSTGRES_PASSWORD=$(gen_hex 16)|" \
    -e "s|^REDIS_PASSWORD=.*|REDIS_PASSWORD=$(gen_hex 16)|" \
    -e "s|^RUSTFS_ROOT_PASSWORD=.*|RUSTFS_ROOT_PASSWORD=$(gen_pw)|" \
    -e "s|^LITELLM_MASTER_KEY=.*|LITELLM_MASTER_KEY=sk-$(gen_hex 32)|" \
    -e "s|^LITELLM_SALT_KEY=.*|LITELLM_SALT_KEY=$(gen_hex 32)|" \
    -e "s|^APP_VERSION=.*|APP_VERSION=$VERSION|" \
    -e "s|^BUILD_TIME=.*|BUILD_TIME=$BUILD_TIME|" \
    "$PROD_DIR/.env.example" > "$BUILD_ENV"

dc() { docker compose -f "$PROD_DIR/docker-compose.yml" --env-file "$BUILD_ENV" "$@"; }

# 编排资产（compose/.env.example/nginx/VERSION/BUILD_TIME——注意不带 .env 机密）
# 增量包 config 目录排除 config.yaml（生产 credential-key 机密，见头部安全说明）
copy_orchestration() {
  local target="$1"
  cp "$PROD_DIR/docker-compose.yml" "$target/"
  cp "$PROD_DIR/.env.example" "$target/"
  cp -r "$PROD_DIR/nginx" "$target/nginx"
  mkdir -p "$target/config"
  for f in postgresql.conf redis.conf litellm.yaml; do
    [ -f "$PROD_DIR/config/$f" ] && cp "$PROD_DIR/config/$f" "$target/config/"
  done
  echo "$VERSION" > "$target/VERSION"
  echo "$BUILD_TIME" > "$target/BUILD_TIME"
}

# ============================================================================
# 1. 构建自研镜像（web/server/updater，版本化 tag）+ 拉取第三方镜像
# ============================================================================
log "构建自研镜像（版本化 tag = $VERSION）"
dc build

log "拉取第三方镜像（--ignore-buildable 跳过自研）"
dc pull --ignore-buildable

# 镜像清单从 compose 动态获取，不手工维护（防漂移）
mapfile -t ALL_IMAGES < <(dc config --images | sort -u)
mapfile -t OWN_IMAGES < <(dc config --images | grep '^devops-admin/' | sort -u)
[ "${#ALL_IMAGES[@]}" -ge 8 ] || die "镜像清单异常（应含 8 个）：${ALL_IMAGES[*]}"
[ "${#OWN_IMAGES[@]}" -eq 3 ] || die "自研镜像清单异常（应含 web/server/updater）：${OWN_IMAGES[*]}"
for img in "${ALL_IMAGES[@]}"; do printf '  ✓ %s\n' "$img"; done

# ============================================================================
# 2. 全量包（既有离线部署流程：强随机 .env + 全部镜像 + install.sh）
# ============================================================================
log "组装全量包"
# .env：本次部署机密（既有约定——构建机生成，install.sh 直接使用）
sed -e "s|^WEB_PORT=.*|WEB_PORT=$WEB_PORT|" \
    -e "s|^SERVER_PORT=8888|# SERVER_PORT=8888  # 生产不对外暴露后端，仅经 web 反代|" \
    -e "s|^JWT_SIGNING_KEY=.*|JWT_SIGNING_KEY=$(gen_hex 32)|" \
    -e "s|^INIT_ADMIN_PASSWORD=.*|INIT_ADMIN_PASSWORD=$(gen_pw)|" \
    -e "s|^POSTGRES_PASSWORD=.*|POSTGRES_PASSWORD=$(gen_hex 16)|" \
    -e "s|^REDIS_PASSWORD=.*|REDIS_PASSWORD=$(gen_hex 16)|" \
    -e "s|^RUSTFS_ROOT_PASSWORD=.*|RUSTFS_ROOT_PASSWORD=$(gen_pw)|" \
    -e "s|^LITELLM_MASTER_KEY=.*|LITELLM_MASTER_KEY=sk-$(gen_hex 32)|" \
    -e "s|^LITELLM_SALT_KEY=.*|LITELLM_SALT_KEY=$(gen_hex 32)|" \
    -e "s|^LITELLM_HOST_PORT=.*|LITELLM_HOST_PORT=$LITELLM_HOST_PORT|" \
    -e "s|^LITELLM_PUBLIC_URL=.*|LITELLM_PUBLIC_URL=http://$DEPLOY_IP:$LITELLM_HOST_PORT|" \
    -e "s|^APP_VERSION=.*|APP_VERSION=$VERSION|" \
    -e "s|^BUILD_TIME=.*|BUILD_TIME=$BUILD_TIME|" \
    -e "s|^UPDATER_TOKEN=.*|UPDATER_TOKEN=$(gen_hex 16)|" \
    "$PROD_DIR/.env.example" > "$FULL_STAGE/.env"

# credential-key 无 env 覆盖机制（other.go 未实现），构建时注入包内 config.yaml：
# AES-256-GCM 64 位 hex；留空会导致 AI 网关凭证拒绝写入
cp -r "$PROD_DIR/config" "$FULL_STAGE/config"
sed -i "s|^litellm:|litellm:\n    credential-key: $(gen_hex 32)|" "$FULL_STAGE/config/config.yaml"

cp "$PROD_DIR/docker-compose.yml" "$FULL_STAGE/"
cp -r "$PROD_DIR/nginx" "$FULL_STAGE/"
cp "$RELEASE_DIR/install.sh" "$RELEASE_DIR/upgrade.sh" "$FULL_STAGE/"
cp "$RELEASE_DIR/README-DEPLOY.md" "$FULL_STAGE/" 2>/dev/null || true
echo "$VERSION" > "$FULL_STAGE/VERSION"
echo "$BUILD_TIME" > "$FULL_STAGE/BUILD_TIME"

log "导出全量镜像（docker save | gzip，约 1-2GB，请耐心等待）"
docker save "${ALL_IMAGES[@]}" | gzip > "$FULL_STAGE/images.tar.gz"

# ============================================================================
# 3. 增量包（在线升级载体：仅自研 3 镜像 + 编排资产，无 .env/无第三方镜像）
# ============================================================================
log "组装增量包"
copy_orchestration "$INCR_STAGE"
cp "$RELEASE_DIR/upgrade.sh" "$INCR_STAGE/"
mkdir -p "$INCR_STAGE/images"
log "导出增量镜像（仅自研 web/server/updater）"
docker save -o "$INCR_STAGE/images/devops-admin-images-incr.tar" "${OWN_IMAGES[@]}"

# ============================================================================
# 4. 压缩产物 + sha256 + manifest
# ============================================================================
log "压缩产物"
FULL_TARBALL="$DIST_DIR/devops-admin-release-$VERSION.tar.gz"
INCR_TARBALL="$DIST_DIR/devops-admin-upgrade-$VERSION.tar.gz"
tar -C "$STAGING" -czf "$FULL_TARBALL" "$(basename "$FULL_STAGE")"
tar -C "$STAGING" -czf "$INCR_TARBALL" "$(basename "$INCR_STAGE")"
# sha256 记录相对文件名，发布服务器/目标机同目录 `sha256sum -c` 可直接校验
( cd "$DIST_DIR" && sha256sum "$(basename "$FULL_TARBALL")" > "$(basename "$FULL_TARBALL").sha256" )
( cd "$DIST_DIR" && sha256sum "$(basename "$INCR_TARBALL")" > "$(basename "$INCR_TARBALL").sha256" )

# manifest：上传发布服务器后改名 manifest.json 生效（原子替换）；
# changeLog/releaseTime/minUpgradeVersion 留待发布前手工编辑
size_of() { stat -c%s "$1"; }
sha_of() { cut -d' ' -f1 "$1.sha256"; }
MANIFEST="$DIST_DIR/manifest-$VERSION.json"
cat > "$MANIFEST" <<EOF
{
  "version": "$VERSION",
  "buildTime": "$BUILD_TIME",
  "releaseTime": "",
  "changeLog": "",
  "minUpgradeVersion": "v0.2.0",
  "forceUpgrade": false,
  "packages": [
    {
      "type": "incr",
      "url": "/packages/devops-admin-upgrade-$VERSION.tar.gz",
      "sha256": "$(sha_of "$INCR_TARBALL")",
      "sizeBytes": $(size_of "$INCR_TARBALL")
    },
    {
      "type": "full",
      "url": "/packages/devops-admin-release-$VERSION.tar.gz",
      "sha256": "$(sha_of "$FULL_TARBALL")",
      "sizeBytes": $(size_of "$FULL_TARBALL")
    }
  ]
}
EOF

# ============================================================================
# 5. 摘要
# ============================================================================
ADMIN_PASSWORDPlaceholder="（见包内 .env 的 INIT_ADMIN_PASSWORD）"
cat <<EOF

============================================================
✅ Release 打包完成（版本 $VERSION）

  全量包  : $FULL_TARBALL（$(du -sh "$FULL_TARBALL" | cut -f1)，8 镜像 + .env + install.sh）
  增量包  : $INCR_TARBALL（$(du -sh "$INCR_TARBALL" | cut -f1)，仅自研镜像）
  清单    : $MANIFEST（发布前手工编辑 changeLog/releaseTime，后改名 manifest.json）
  版本    : $VERSION（自研镜像 tag = 版本号，旧镜像保留可回滚）

全新部署（全量包）：
  scp $FULL_TARBALL root@$DEPLOY_IP:/root/
  ssh root@$DEPLOY_IP "mkdir -p /root/devops-admin && tar -xzf /root/$(basename "$FULL_TARBALL") -C /root/devops-admin --strip-components=1"
  ssh root@$DEPLOY_IP "cd /root/devops-admin && bash install.sh"

在线升级（增量包，二选一）：
  A. 发布服务器：publish.sh 推送 → 编辑 manifest 填 changelog → 改名 manifest.json 生效
     → 生产「关于」弹窗检查更新 → 在线升级
  B. 手工直传：scp 增量包到生产安装目录解压（--strip-components=1）→ bash upgrade.sh
============================================================
EOF
