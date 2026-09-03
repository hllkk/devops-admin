#!/usr/bin/env bash
# ============================================================================
# devops-admin 版本发布辅助（构建机执行，产物 → 发布服务器）
#
# 用法：
#   ./publish.sh <user@host> [远程发布根目录] [版本号]
#     user@host        发布服务器（scp/ssh 直连；端口非 22 用 SSH_OPTS="-p 2222"）
#     远程发布根目录    默认 /opt/devops-admin-publish（与 nginx.conf.example 的 root 对应；
#                      多项目共用发布服务器时各项目独立根目录，由 nginx 按路径前缀区分）
#     版本号            默认取 dist/ 下最新的 manifest-<版本>.json
#
# 动作：
#   1. 检查 dist/ 三件套（manifest-<版本>.json + 增量包 + 全量包，含 .sha256）
#   2. 两个包与 .sha256 上传到 <根目录>/packages/
#   3. manifest-<版本>.json 上传到 <根目录>/（不直接生效）
#   4. 提示手工编辑 changeLog/releaseTime 后 mv 改名 manifest.json 生效（原子替换，旧清单保留）
#
# 之所以不改名直接生效：changeLog 必须人工编写（打包脚本不知道改了什么），
# 且 mv 原子替换避免前端检查更新拉到半编辑状态的清单。
# ============================================================================
set -euo pipefail

PUBLISH_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DIST_DIR="$(cd "$PUBLISH_DIR/.." && pwd)/dist"

SSH_OPTS="${SSH_OPTS:-}"

REMOTE="${1:-}"
[ -n "$REMOTE" ] || { echo "用法：./publish.sh <user@host> [远程发布根目录] [版本号]"; exit 1; }
REMOTE_ROOT="${2:-/opt/devops-admin-publish}"

VERSION="${3:-}"
if [ -z "$VERSION" ]; then
  VERSION="$(ls -1t "$DIST_DIR"/manifest-*.json 2>/dev/null | head -1 | sed 's|.*/manifest-\(.*\)\.json$|\1|')"
fi
[ -n "$VERSION" ] || { echo "[错误] 未指定版本且 $DIST_DIR 无 manifest-<版本>.json"; exit 1; }

MANIFEST="$DIST_DIR/manifest-$VERSION.json"
INCR_TARBALL="$DIST_DIR/devops-admin-upgrade-$VERSION.tar.gz"
FULL_TARBALL="$DIST_DIR/devops-admin-release-$VERSION.tar.gz"

# ---- 产物就位检查 -------------------------------------------------------------
for f in "$MANIFEST" "$INCR_TARBALL" "$INCR_TARBALL.sha256" "$FULL_TARBALL" "$FULL_TARBALL.sha256"; do
  [ -f "$f" ] || { echo "[错误] 缺产物：$f（先跑 build-release.sh）"; exit 1; }
done

# ---- 上传 ---------------------------------------------------------------------
echo "==> 建远程目录 $REMOTE_ROOT/packages"
ssh $SSH_OPTS "$REMOTE" "mkdir -p '$REMOTE_ROOT/packages'"

echo "==> 上传升级包（增量 $(du -h "$INCR_TARBALL" | cut -f1) / 全量 $(du -h "$FULL_TARBALL" | cut -f1)）"
scp $SSH_OPTS "$INCR_TARBALL" "$INCR_TARBALL.sha256" "$FULL_TARBALL" "$FULL_TARBALL.sha256" "$REMOTE:$REMOTE_ROOT/packages/"

echo "==> 上传版本清单（不直接生效）"
scp $SSH_OPTS "$MANIFEST" "$REMOTE:$REMOTE_ROOT/"

cat <<EOF

[下一步] 在发布服务器上手工编辑清单后生效：
  ssh $SSH_OPTS $REMOTE
  vi $REMOTE_ROOT/$(basename "$MANIFEST")     # 填 changeLog / releaseTime / minUpgradeVersion
  mv $REMOTE_ROOT/$(basename "$MANIFEST") $REMOTE_ROOT/manifest.json

生产侧检查更新地址（.env 的 UPDATE_SERVER_URL）：
  http://<发布服务器>/<devops-admin 前缀>   # 与 nginx.conf.example 的 location 对应
EOF
