#!/bin/bash
# 前端代码同步脚本 - 安全版本
# 检测到冲突时暂停，通知用户处理

set -e

FRONTEND_DIR="/home/devops-admin/frontend"
LOG_FILE="/home/devops-admin/logs/sync-frontend.log"
CONFLICT_FLAG="/home/devops-admin/logs/.sync-conflict-pending"

mkdir -p "$(dirname "$LOG_FILE")"

log() {
    echo "[$(date "+%Y-%m-%d %H:%M:%S")] $1" | tee -a "$LOG_FILE"
}

cd "$FRONTEND_DIR"

# 1. 检查是否有未提交的更改
if ! git diff-index --quiet HEAD --; then
    log "⚠️ 存在未提交的更改，先暂存..."
    git stash push -m "auto-stash-$(date +%Y%m%d%H%M%S)"
    STASHED=true
fi

# 2. 获取上游最新代码
log "📥 获取上游最新代码..."
git fetch upstream

# 3. 检查是否有更新
UPSTREAM_HASH=$(git rev-parse upstream/main)
LOCAL_HASH=$(git rev-parse main)

if [ "$UPSTREAM_HASH" = "$LOCAL_HASH" ]; then
    log "✅ 已是最新版本，无需更新"
    rm -f "$CONFLICT_FLAG"
    exit 0
fi

# 4. 查看更新的文件
CHANGED_COUNT=$(git diff --name-only main upstream/main | wc -l)
log "📋 上游有 $CHANGED_COUNT 个文件更新"

# 5. 尝试合并
log "🔄 开始合并上游代码..."
if git merge upstream/main --no-edit 2>> "$LOG_FILE"; then
    log "✅ 合并成功，无冲突"
    git push origin main
    log "✅ 已推送到 origin"
    rm -f "$CONFLICT_FLAG"
else
    # 合并失败，检查冲突
    CONFLICTS=$(git diff --name-only --diff-filter=U 2>/dev/null || true)
    
    if [ -n "$CONFLICTS" ]; then
        log "⚠️ 发现冲突，暂停合并等待处理"
        
        # 创建冲突标记文件
        echo "CONFLICT_DATE=$(date "+%Y-%m-%d %H:%M:%S")" > "$CONFLICT_FLAG"
        echo "UPSTREAM_HASH=$UPSTREAM_HASH" >> "$CONFLICT_FLAG"
        echo "CONFLICT_FILES:" >> "$CONFLICT_FLAG"
        echo "$CONFLICTS" >> "$CONFLICT_FLAG"
        
        log "冲突文件列表："
        log "$CONFLICTS"
        
        # 不自动处理，退出等待人工介入
        exit 2
    else
        log "❌ 合并失败（非冲突原因），请手动处理"
        exit 1
    fi
fi

# 恢复暂存的更改
[ "$STASHED" = true ] && git stash pop 2>/dev/null || true

log "========== 同步完成 =========="
