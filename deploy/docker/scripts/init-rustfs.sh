#!/bin/bash
# RustFS 初始化脚本 - 创建存储桶
# 在 docker compose up 后运行一次

set -e

RUSTFS_ALIAS="rustfs"
RUSTFS_ENDPOINT="http://localhost:9000"
RUSTFS_USER="${RUSTFS_ROOT_USER:-devops-admin}"
RUSTFS_PASSWORD="${RUSTFS_ROOT_PASSWORD:-devops_rustfs_2026}"
BUCKET_NAME="devops-admin"

echo "=== RustFS 初始化 ==="

# 等待 RustFS 启动
echo "等待 RustFS 服务就绪..."
for i in $(seq 1 30); do
    if curl -sf http://localhost:9000/minio/health/live > /dev/null 2>&1; then
        echo "RustFS 服务已就绪"
        break
    fi
    echo "  等待中... ($i/30)"
    sleep 2
done

# 检查 mc 是否已配置
if ! mc alias list | grep -q "$RUSTFS_ALIAS"; then
    echo "配置 mc 别名..."
    mc alias set "$RUSTFS_ALIAS" "$RUSTFS_ENDPOINT" "$RUSTFS_USER" "$RUSTFS_PASSWORD"
fi

# 创建存储桶
echo "创建存储桶: $BUCKET_NAME"
mc mb "$RUSTFS_ALIAS/$BUCKET_NAME" 2>/dev/null || echo "存储桶已存在"

echo "=== RustFS 初始化完成 ==="
