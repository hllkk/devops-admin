#!/usr/bin/env bash
# ============================================================
# sync-upstream.sh — 同步 SoybeanAdmin 上游框架更新
# ============================================================
#
# 用法:
#   bash scripts/sync-upstream.sh          # 预览变更
#   bash scripts/sync-upstream.sh --run    # 执行同步
#   bash scripts/sync-upstream.sh --dry-run  # 模拟运行（同预览）
#
# 原理: git subtree pull, 将上游新 commit 压缩成一个 squash merge
# 冲突策略:
#   框架层 (packages/, build/)   → 接受上游
#   业务层 (src/views/, src/router/routes/) → 保留本地
#   混合层 (package.json 等)     → 人工判断

set -euo pipefail

UPSTREAM_REPO="https://github.com/soybeanjs/soybean-admin.git"
UPSTREAM_BRANCH="main"
PREFIX="web"
TRACK_FILE="$PREFIX/.upstream-track"

# ── 颜色 ──
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; NC='\033[0m'

log()  { echo -e "${CYAN}[INFO]${NC}  $*"; }
ok()   { echo -e "${GREEN}[OK]${NC}    $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC}  $*"; }
err()  { echo -e "${RED}[ERR]${NC}   $*"; }

# ── 参数 ──
RUN_MODE="preview"
while [[ $# -gt 0 ]]; do
    case "$1" in
        --run)     RUN_MODE="run" ;;
        --dry-run) RUN_MODE="dry" ;;
        --preview) RUN_MODE="preview" ;;
        *) err "未知参数: $1"; exit 2 ;;
    esac
    shift
done

# ── 前置检查 ──
if ! git status >/dev/null 2>&1; then
    err "不在 git 仓库中"; exit 1
fi

# 检查是否有未提交更改
if [[ "$RUN_MODE" == "run" ]]; then
    if ! git diff-index --quiet HEAD --; then
        err "存在未提交的更改，请先提交或 stash"
        git status --short
        exit 1
    fi
fi

# 确认在项目根目录
if [[ ! -d "$PREFIX" ]]; then
    err "未找到 $PREFIX 目录，请在项目根目录运行此脚本"; exit 1
fi

# ── 获取上游最新信息 ──
log "获取上游最新信息..."
git fetch "$UPSTREAM_REPO" "$UPSTREAM_BRANCH" --quiet 2>&1 || {
    err "无法 fetch 上游仓库 $UPSTREAM_REPO"
    exit 1
}

UPSTREAM_HEAD=$(git rev-parse FETCH_HEAD)
UPSTREAM_SHORT="${UPSTREAM_HEAD:0:12}"
UPSTREAM_DATE=$(git log -1 FETCH_HEAD --format='%ai' 2>/dev/null || echo "unknown")

# 读取当前本地跟踪的上游版本
LOCAL_TRACKED=""
if [[ -f "$TRACK_FILE" ]]; then
    LOCAL_TRACKED=$(head -1 "$TRACK_FILE")
fi

log "本地记录的上游版本: ${LOCAL_TRACKED:-未记录}"
log "当前上游最新版本:    $UPSTREAM_SHORT ($UPSTREAM_DATE)"

if [[ "$LOCAL_TRACKED" == "$UPSTREAM_HEAD" ]]; then
    ok "已是最新版本，无需同步"
    exit 0
fi

# ── 统计变更 ──
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
log "上游变更统计 (相对于本地记录版本)"

# 如果有 subtree 历史，对比当前目录和上游
LOCAL_HASH=$(git log -1 --format='%H' -- "$PREFIX" 2>/dev/null || echo "")
if [[ -n "$LOCAL_HASH" ]]; then
    echo ""
    log "新增提交数: $(git rev-list --count "${LOCAL_HASH}..${UPSTREAM_HEAD}" 2>/dev/null || echo "?")"
    echo ""
    log "变更文件列表 (前 60 行):"
    git diff --stat "${LOCAL_HASH}..${UPSTREAM_HEAD}" -- "$PREFIX" 2>/dev/null | head -60 || true
    echo ""
fi

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if [[ "$RUN_MODE" != "run" ]]; then
    echo ""
    warn "预览模式，未实际执行。使用 --run 执行同步。"
    exit 0
fi

# ── 执行 subtree pull ──
log "执行 git subtree pull --prefix=$PREFIX --squash..."

if git subtree pull --prefix="$PREFIX" "$UPSTREAM_REPO" "$UPSTREAM_BRANCH" --squash \
    -m "chore(web): 同步 soybean-admin 上游更新至 $UPSTREAM_SHORT" 2>&1; then

    # ── 更新跟踪文件 ──
    echo "$UPSTREAM_HEAD" > "$TRACK_FILE"
    echo "v$(date +%Y-%m-%d)" >> "$TRACK_FILE"
    echo "$UPSTREAM_REPO" >> "$TRACK_FILE"
    git add "$TRACK_FILE"

    ok "同步完成: $UPSTREAM_SHORT"

else
    # ── 冲突处理 ──
    warn "存在合并冲突，进入交互式冲突解决..."

    CONFLICT_FILES=$(git diff --name-only --diff-filter=U 2>/dev/null || true)

    if [[ -z "$CONFLICT_FILES" ]]; then
        err "未检测到冲突文件但 merge 仍未完成，请手动检查"
        exit 1
    fi

    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  冲突文件列表:"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    for f in $CONFLICT_FILES; do
        # 自动判定策略
        STRATEGY="manual"
        case "$f" in
            web/packages/*|web/build/*|web/pnpm-lock.yaml|web/CHANGELOG*|web/.github/*|web/eslint*|web/.oxlint*|web/.oxfmt*|web/tsconfig.json)
                STRATEGY="upstream (框架层，接受上游)"
                git checkout --theirs "$f" 2>/dev/null && git add "$f" || true
                ;;
            web/src/views/*|web/src/router/routes/*|web/src/service/*|web/.env|web/.env.prod|web/.env.test|web/vite.config.ts|web/src/layouts/*|web/src/store/*|web/src/locales/*)
                STRATEGY="local (业务层，保留本地)"
                git checkout --ours "$f" 2>/dev/null && git add "$f" || true
                ;;
            web/package.json)
                STRATEGY="⚠ manual (混合层，需人工合并)"
                ;;
            *)
                STRATEGY="⚠ manual"
                ;;
        esac
        printf "  %-50s → %s\n" "$f" "$STRATEGY"
    done

    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    warn "自动解决的冲突已按策略处理完成"
    warn "标记为 ⚠ manual 的文件需要你手动合并:"
    echo ""
    for f in $CONFLICT_FILES; do
        case "$f" in
            web/package.json)
                echo "  - $f (检查上游新依赖 vs 本地自定义脚本)"
                ;;
        esac
    done
    echo ""
    warn "手动合并完成后，执行:"
    echo "  git add -A"
    echo "  git commit -m 'chore(web): 同步 soybean-admin 上游更新至 $UPSTREAM_SHORT'"
    echo "  echo '$UPSTREAM_HEAD' > $TRACK_FILE && echo '$(date +%F)' >> $TRACK_FILE && git add $TRACK_FILE"
fi

echo ""
log "同步后建议:"
echo "  1. cd web && pnpm install  (如 lockfile 有变更)"
echo "  2. pnpm dev                 (验证前端是否正常运行)"
echo "  3. git push origin master   (推送更新)"
