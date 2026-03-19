#!/bin/bash
# 前端代码同步脚本 - 分支版本
# main 分支同步上游，dev 分支用于开发

set -e

FRONTEND_DIR="/home/devops-admin/frontend"
LOG_FILE="/home/devops-admin/logs/sync-frontend.log"
CONFLICT_FLAG="/home/devops-admin/logs/.sync-conflict-pending"

mkdir -p "$(dirname "$LOG_FILE")"

log() {
    echo "[$(date "+%Y-%m-%d %H:%M:%S")] $1" | tee -a "$LOG_FILE"
}

cd "$FRONTEND_DIR"

# 记录当前分支
CURRENT_BRANCH=$(git branch --show-current)

# 1. 切换到 main 分支
log "🔄 切换到 main 分支..."
git checkout main

# 2. 暂存未提交的更改（如果有）
if ! git diff-index --quiet HEAD --; then
    log "⚠️ 存在未提交的更改，先暂存..."
    git stash push -m "auto-stash-$(date +%Y%m%d%H%M%S)"
    STASHED=true
fi

# 3. 获取上游最新代码
log "📥 获取上游最新代码..."
git fetch upstream

# 4. 检查是否有更新
UPSTREAM_HASH=$(git rev-parse upstream/main)
LOCAL_HASH=$(git rev-parse main)

if [ "$UPSTREAM_HASH" = "$LOCAL_HASH" ]; then
    log "✅ main 已是最新版本"
    
    # 恢复原来的分支
    git checkout "$CURRENT_BRANCH"
    [ "$STASHED" = true ] && git stash pop 2>/dev/null || true
    
    rm -f "$CONFLICT_FLAG"
    exit 0
fi

# 5. 更新 main 分支
log "🔄 更新 main 分支..."
git merge upstream/main --no-edit 2>> "$LOG_FILE" || {
    log "❌ main 分支合并失败，请检查"
    git checkout "$CURRENT_BRANCH"
    exit 1
}

git push origin main
log "✅ main 分支已更新"

# 6. 合并到 dev 分支
log "🔄 合并更新到 dev 分支..."
git checkout dev
git merge main --no-edit 2>> "$LOG_FILE" || MERGE_FAILED=true

if [ "$MERGE_FAILED" = true ]; then
    CONFLICTS=$(git diff --name-only --diff-filter=U 2>/dev/null || true)
    
    if [ -n "$CONFLICTS" ]; then
        log "⚠️ dev 分支发现冲突，暂停等待处理"
        
        echo "CONFLICT_DATE=$(date "+%Y-%m-%d %H:%M:%S")" > "$CONFLICT_FLAG"
        echo "BRANCH=dev" >> "$CONFLICT_FLAG"
        echo "CONFLICT_FILES:" >> "$CONFLICT_FLAG"
        echo "$CONFLICTS" >> "$CONFLICT_FLAG"
        
        log "冲突文件：$CONFLICTS"
        exit 2
    fi
fi

git push origin dev
log "✅ dev 分支已更新"

# 恢复原来的分支
[ "$CURRENT_BRANCH" != "dev" ] && git checkout "$CURRENT_BRANCH"
[ "$STASHED" = true ] && git stash pop 2>/dev/null || true

rm -f "$CONFLICT_FLAG"
log "========== 同步完成 =========="
