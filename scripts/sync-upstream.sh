#!/usr/bin/env bash
# ============================================================
# sync-upstream.sh — 同步 SoybeanAdmin 上游框架更新到 web/
# ============================================================
#
# 用法:
#   bash scripts/sync-upstream.sh            # 预览变更（不写盘）
#   bash scripts/sync-upstream.sh --preview  # 同上
#   bash scripts/sync-upstream.sh --run      # 执行同步
#
# 同步策略（自动选择，无需手动指定）:
#   1) apply 模式（默认）：本仓库 web/ 是直接复制进来的、未经 `git subtree add`
#      正式导入，与上游没有共同合并基点，`git subtree pull` 会按“不相关历史”合并
#      引发大量无关冲突。因此把“上游相对上次同步的真实增量”作为补丁应用到 web/：
#        git diff <本地记录的上游sha> <上游HEAD> | git apply --directory=web
#      只动上游真正改过的文件，干净、0 误伤。
#   2) subtree 模式（兜底）：若检测到本地 web/ 与上游存在 subtree 合并基点
#      （即曾经 `git subtree add` 过），则回退到 `git subtree pull --squash`。
#
# 冲突策略:
#   apply   → 某文件本地已改动导致补丁不匹配 → 不改动工作区，逐文件报告，人工处理
#   subtree → 框架层(packages/,build/)接受上游；业务层(src/views,src/router/routes,
#             src/service,.env*,src/layouts,...)保留本地；package.json 等混合层人工判断
#
# 跟踪文件: web/.upstream-track 第一行 = 上次同步到的上游 commit sha。

set -euo pipefail

UPSTREAM_REPO="git@github.com:soybeanjs/soybean-admin.git"
UPSTREAM_REPO_HTTPS="https://github.com/soybeanjs/soybean-admin.git"
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

# ── fetch 上游（SSH 主路径，HTTPS 兜底）。$1=分支或sha，其余透传给 git fetch ──
fetch_upstream() {
    local ref="$1"; shift
    git fetch "$UPSTREAM_REPO" "$ref" "$@" --no-tags --quiet 2>&1 \
        || git fetch "$UPSTREAM_REPO_HTTPS" "$ref" "$@" --no-tags --quiet 2>&1
}

# ── 更新跟踪文件（沿用既有三行格式：sha / v{date} / repo）──
write_track() {
    {
        echo "$UPSTREAM_HEAD"
        echo "v$(date +%Y-%m-%d)"
        echo "$UPSTREAM_REPO"
    } > "$TRACK_FILE"
}

# ── apply 模式：确保记录的基点 commit 存在于本地（浅克隆里可能没有）──
ensure_tracked_commit() {
    if [[ -z "$LOCAL_TRACKED" ]]; then
        err "未记录基点（$TRACK_FILE 缺失），apply 模式需要一个已同步的上游基点。"
        err "请先手动同步一次并写入 $TRACK_FILE，或用 git subtree add 正式导入。"
        exit 1
    fi
    if ! git cat-file -e "${LOCAL_TRACKED}^{commit}" 2>/dev/null; then
        log "记录的基点 $LOCAL_TRACKED 不在浅克隆中，单独拉取（depth=1）..."
        if ! fetch_upstream "$LOCAL_TRACKED" --depth=1; then
            err "无法拉取基点 commit $LOCAL_TRACKED，请检查网络或手动 git fetch。"
            exit 1
        fi
    fi
}

# ── apply 模式：把上游增量补丁应用到 web/ ──
run_apply_mode() {
    ensure_tracked_commit
    log "apply 模式：增量 ${LOCAL_TRACKED:0:12} → $UPSTREAM_SHORT 应用到 $PREFIX/"

    local tmp_patch conflict=0
    tmp_patch=$(mktemp)
    git diff "$LOCAL_TRACKED" "$UPSTREAM_HEAD" > "$tmp_patch"

    if [[ ! -s "$tmp_patch" ]]; then
        write_track
        rm -f "$tmp_patch"
        ok "无文件变更（仅更新跟踪记录）"
        return 0
    fi

    # 整体预检：能干净应用就一次性 apply + 提交。
    # 必须写入临时文件再 check —— 用 $(...) 捕获补丁会吞掉结尾换行，
    # 导致多文件补丁的 `git apply --check` 误判失败（逐文件 pipe 检查却通过）。
    if git apply --directory="$PREFIX" --check "$tmp_patch" 2>/dev/null; then
        git apply --directory="$PREFIX" "$tmp_patch"
        write_track
        git add "$PREFIX"
        git diff --quiet --cached || git commit -q -m "chore(web): 同步 soybean-admin 上游更新至 $UPSTREAM_SHORT"
        rm -f "$tmp_patch"
        ok "同步完成（apply 模式）：$UPSTREAM_SHORT"
        return 0
    fi

    # 有冲突：逐文件定位，不改动工作区
    warn "以下文件本地已改动，无法干净应用上游增量（未改动工作区）："
    while IFS= read -r f; do
        if git diff "$LOCAL_TRACKED" "$UPSTREAM_HEAD" -- "$f" \
            | git apply --directory="$PREFIX" --check - 2>/dev/null; then
            printf '  [ok]       %s\n' "$f"
        else
            printf '  [CONFLICT] %s\n' "$f"
            conflict=1
        fi
    done < <(git diff --name-only "$LOCAL_TRACKED" "$UPSTREAM_HEAD")
    rm -f "$tmp_patch"

    echo ""
    err "存在冲突文件，已中止同步。手动解决后重跑；或对单个文件用 3way 合并："
    echo "  git diff ${LOCAL_TRACKED:0:12} $UPSTREAM_SHORT -- <file> | git apply --directory=$PREFIX --3way"
    return $conflict
}

# ── subtree 模式（兜底；仅在检测到合并基点时使用）──
run_subtree_mode() {
    log "执行 git subtree pull --prefix=$PREFIX --squash..."
    if git subtree pull --prefix="$PREFIX" "$UPSTREAM_REPO" "$UPSTREAM_BRANCH" --squash \
        -m "chore(web): 同步 soybean-admin 上游更新至 $UPSTREAM_SHORT" 2>&1; then
        write_track
        git add "$TRACK_FILE"
        ok "同步完成（subtree 模式）：$UPSTREAM_SHORT"
        return 0
    fi

    # ── subtree 冲突处理 ──
    warn "存在合并冲突，按策略自动解决..."
    CONFLICT_FILES=$(git diff --name-only --diff-filter=U 2>/dev/null || true)
    if [[ -z "$CONFLICT_FILES" ]]; then
        err "未检测到冲突文件但 merge 仍未完成，请手动检查"
        return 1
    fi

    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  冲突文件列表:"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    for f in $CONFLICT_FILES; do
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
    warn "标记为 ⚠ manual 的文件需要你手动合并，完成后执行:"
    echo "  git add -A && git commit -m 'chore(web): 同步 soybean-admin 上游更新至 $UPSTREAM_SHORT'"
    write_track && git add "$TRACK_FILE"
    return 1
}

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

# --run 要求工作区干净（apply/subtree 都会自己产生提交）
if [[ "$RUN_MODE" == "run" ]]; then
    if ! git diff-index --quiet HEAD --; then
        err "存在未提交的更改，请先提交或 stash"
        git status --short
        exit 1
    fi
fi

if [[ ! -d "$PREFIX" ]]; then
    err "未找到 $PREFIX 目录，请在项目根目录运行此脚本"; exit 1
fi

# ── 获取上游最新信息（SSH 主路径，HTTPS 兜底）──
log "获取上游最新信息..."
git fetch "$UPSTREAM_REPO" "$UPSTREAM_BRANCH" --depth=100 --no-tags --quiet 2>&1 || {
    warn "SSH fetch 失败，尝试 HTTPS..."
    git fetch "$UPSTREAM_REPO_HTTPS" "$UPSTREAM_BRANCH" --depth=100 --no-tags --quiet 2>&1 || {
        err "仍无法访问上游，请检查网络连接"
        exit 1
    }
}

UPSTREAM_HEAD=$(git rev-parse FETCH_HEAD)
UPSTREAM_SHORT="${UPSTREAM_HEAD:0:12}"
UPSTREAM_DATE=$(git log -1 FETCH_HEAD --format='%ai' 2>/dev/null || echo "unknown")

# 读取本地跟踪的上游版本
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

# 用 subtree split 把本地 web/ 内容映射到根命名空间，再和上游 HEAD 对比。
# 上游文件在仓库根、本地在 web/ 下，直接 `git diff .. -- web/` 会让本地所有文件
# 被误判为“上游删除”。先 split 到同命名空间才能得到真实的变更面。
if SPLIT_HASH=$(git subtree split --prefix="$PREFIX" 2>/dev/null); then
    echo ""
    log "新增上游提交数: $(git rev-list --count "${LOCAL_TRACKED:-${SPLIT_HASH}}..${UPSTREAM_HEAD}" 2>/dev/null || echo "?")"
    echo ""
    log "变更文件列表 (本地 $PREFIX ↔ 上游，前 60 行):"
    git diff --stat "${SPLIT_HASH}..${UPSTREAM_HEAD}" 2>/dev/null | head -60 || true
    echo ""
else
    # 兜底：git-subtree 不可用时退回上游提交日志，避免输出误导性的文件 diff
    echo ""
    log "上游新增提交 (最近 60 条):"
    git log --oneline "${UPSTREAM_HEAD}" 2>/dev/null | head -60 || true
    echo ""
    warn "git-subtree 不可用，仅展示提交日志；安装后可见文件级变更: dnf install git-subtree"
fi

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if [[ "$RUN_MODE" != "run" ]]; then
    echo ""
    warn "预览模式，未实际执行。使用 --run 执行同步。"
    exit 0
fi

# ── 执行同步：自动选择策略 ──
log "判定同步策略（检测 subtree 合并基点是否存在）..."
SYNC_RC=0
if SPLIT_HASH=$(git subtree split --prefix="$PREFIX" 2>/dev/null) \
   && git merge-base "$SPLIT_HASH" "$UPSTREAM_HEAD" >/dev/null 2>&1; then
    log "存在 subtree 合并基点 → 使用 subtree pull"
    run_subtree_mode || SYNC_RC=$?
else
    log "无 subtree 合并基点（web/ 非正式导入）→ 使用 apply 模式"
    run_apply_mode || SYNC_RC=$?
fi

if [[ $SYNC_RC -eq 0 ]]; then
    echo ""
    log "同步后建议:"
    echo "  1. cd $PREFIX && pnpm install  (如 lockfile 有变更)"
    echo "  2. pnpm dev                     (验证前端是否正常运行)"
    echo "  3. git push origin master       (推送更新)"
fi
exit $SYNC_RC
