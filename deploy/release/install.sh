#!/usr/bin/env bash
# ============================================================================
# devops-admin 一键部署脚本（在目标服务器执行）
#
# 用法（离线包解压后）:
#   bash install.sh
#
# 行为:
#   1. 前置检查（docker / compose / 宿主端口冲突预检）
#   2. 创建数据目录（默认 /home/docker/docker-prod/{pgsql,redis,rustfs,uploads,server-log}）
#   3. docker load 离线镜像
#   4. docker compose up -d --no-build（镜像已 load，目标机无需源码/外网）
#   5. 等待健康检查通过，触发自动初始化（建管理员）
#   6. 打印访问信息与后续运维提示
#
# 幂等: 重复执行 = 升级（load 覆盖同名镜像 → up -d 仅重建有变化的容器，
#        数据均在 bind mount 持久化目录，不受影响）
# ============================================================================
set -euo pipefail

DEPLOY_HOME="${DEPLOY_HOME:-/home/docker/docker-prod}"
ENV_FILE=".env"
COMPOSE_FILE="docker-compose.yml"
HEALTH_TIMEOUT=300

log() { printf '\n\033[1;32m==> %s\033[0m\n' "$*"; }
warn() { printf '\033[1;33m[警告] %s\033[0m\n' "$*"; }
die() { printf '\033[1;31m[错误] %s\033[0m\n' "$*" >&2; exit 1; }

# ---- 读取 .env 变量 ----
env_get() { grep -E "^$1=" "$ENV_FILE" 2>/dev/null | head -1 | cut -d= -f2-; }

cd "$(dirname "${BASH_SOURCE[0]}")"
[ -f "$ENV_FILE" ] || die "缺少 $ENV_FILE（部署包不完整）"
[ -f "$COMPOSE_FILE" ] || die "缺少 $COMPOSE_FILE（部署包不完整）"
[ -f "images.tar.gz" ] || die "缺少 images.tar.gz（部署包不完整）"

# ============================================================================
# 1. 前置检查
# ============================================================================
log "前置检查"
command -v docker >/dev/null || die "未安装 docker"
docker compose version >/dev/null 2>&1 || die "未安装 docker compose v2"

# 端口冲突预检：WEB_PORT 与 LITELLM_HOST_PORT（升级场景下被本项目容器占用则放行）
check_port() {
  local port="$1" name="$2"
  [ -n "$port" ] || return 0
  if ss -tln "( sport = :$port )" | grep -q LISTEN; then
    if docker ps --format '{{.Names}}' --filter "publish=$port" | grep -q '^devops-'; then
      printf '  ○ %s 端口 %s 由本项目容器占用（升级场景，放行）\n' "$name" "$port"
    else
      die "$name 端口 $port 已被占用：$(ss -tlnp "( sport = :$port )" | grep LISTEN | head -1)"
    fi
  else
    printf '  ✓ %s 端口 %s 空闲\n' "$name" "$port"
  fi
}

WEB_PORT=$(env_get WEB_PORT); WEB_PORT=${WEB_PORT:-80}
LITELLM_HOST_PORT=$(env_get LITELLM_HOST_PORT)
# 值形如 "4001" / "0.0.0.0:4001" / "127.0.0.1:4000"，取最后一个冒号后的端口
LITELLM_PORT_NUM="${LITELLM_HOST_PORT##*:}"
check_port "$WEB_PORT" "web 前端"
check_port "$LITELLM_PORT_NUM" "litellm"

# ============================================================================
# 2. 创建数据目录（幂等）
# ============================================================================
log "准备数据目录（$DEPLOY_HOME）"
for var in PG_DATA_PATH REDIS_DATA_PATH RUSTFS_DATA_PATH SERVER_UPLOADS_PATH SERVER_LOG_PATH; do
  dir=$(env_get "$var"); dir=${dir:-$DEPLOY_HOME}
  mkdir -p "$dir"
  # server 以非 root app 用户运行（Dockerfile.server: uid=100 gid=101），日志/上传目录须可写，
  # 否则 zap 持续 write error: permission denied
  case "$var" in
    SERVER_LOG_PATH|SERVER_UPLOADS_PATH) chown -R 100:101 "$dir" 2>/dev/null || chmod 777 "$dir" ;;
  esac
  printf '  ✓ %s\n' "$dir"
done

# ============================================================================
# 3. 导入离线镜像
# ============================================================================
log "导入离线镜像（约 1-3 分钟）"
docker load -i images.tar.gz

# ============================================================================
# 4. 启动全栈
# ============================================================================
log "启动服务（docker compose up -d --no-build）"
docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d --no-build

# ============================================================================
# 5. 等待健康检查
# ============================================================================
log "等待服务健康（超时 ${HEALTH_TIMEOUT}s）"
elapsed=0
while true; do
  unhealthy=0
  status_line=""
  for c in devops-pgsql devops-redis devops-server; do
    st=$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}no-healthcheck{{end}}' "$c" 2>/dev/null || echo "missing")
    status_line="$status_line  $c=$st"
    [ "$st" = "healthy" ] || [ "$st" = "no-healthcheck" ] || unhealthy=1
  done
  printf '\r%s' "$status_line"
  [ "$unhealthy" -eq 0 ] && break
  [ "$elapsed" -ge "$HEALTH_TIMEOUT" ] && {
    printf '\n'
    warn "等待超时，查看日志定位: docker compose -f $COMPOSE_FILE logs --tail=50"
    docker compose -f "$COMPOSE_FILE" ps
    exit 1
  }
  sleep 5; elapsed=$((elapsed + 5))
done
printf '\n  ✓ 全部核心容器 healthy\n'

# ============================================================================
# 5.5 等待 litellm 就绪，并补偿启动时序
# litellm 首次启动跑 Prisma 建表（1-3 分钟）；server 若先于建表启动，spend 表达式
# 索引会因 LiteLLM_SpendLogs 不存在而告警跳过 —— litellm 就绪后重启 server 补建
# ============================================================================
log "等待 litellm 就绪（首次启动含数据库建表，最长 180s）"
elapsed=0
until curl -sf -m 5 "http://127.0.0.1:${LITELLM_PORT_NUM:-4000}/health/liveliness" >/dev/null 2>&1; do
  [ "$elapsed" -ge 180 ] && { warn "litellm 未就绪，继续部署（AI 网关功能可能需人工确认）"; break; }
  sleep 5; elapsed=$((elapsed + 5))
done
[ "$elapsed" -lt 180 ] && printf '  ✓ litellm 就绪（耗时 %ss）\n' "$elapsed"

log "重启 server 补建 spend 索引（时序补偿）"
docker restart devops-server >/dev/null
elapsed=0
until curl -sf -m 3 "http://127.0.0.1:${WEB_PORT}/proxy-default/health" >/dev/null 2>&1; do
  [ "$elapsed" -ge 60 ] && die "server 重启后未就绪，查看: docker logs devops-server"
  sleep 3; elapsed=$((elapsed + 3))
done
printf '  ✓ server 就绪\n'

# ============================================================================
# 6. 确认初始化状态（server 启动期已自动初始化；此调用幂等确认，异常不阻断）
# ============================================================================
log "确认初始化状态 /init/autoInitDB"
sleep 3
INIT_RESP=$(curl -s -m 20 -X POST "http://127.0.0.1:${WEB_PORT}/proxy-default/init/autoInitDB" || echo '{"err":"请求失败"}')
printf '  响应: %s\n' "$(echo "$INIT_RESP" | head -c 300)"
# 成功初始化 → 成功；启动期已自动初始化 → 「已存在数据库配置，无需初始化」，同视为就绪
echo "$INIT_RESP" | grep -q '"code":"0\|成功\|已初始化\|无需初始化' || warn "初始化接口返回异常，若首次部署请人工确认（可能已初始化过）"

# ============================================================================
# 7. 输出部署结果
# ============================================================================
ADMIN_PASSWORD=$(env_get INIT_ADMIN_PASSWORD)
LITELLM_PUBLIC_URL=$(env_get LITELLM_PUBLIC_URL)

cat <<EOF

============================================================
✅ devops-admin 部署完成

  访问地址     : http://<本机IP>:${WEB_PORT}
  初始管理员   : admin
  初始密码     : ${ADMIN_PASSWORD}（登录后请立即修改）
  AI 网关接入  : ${LITELLM_PUBLIC_URL:-（未配置）}

  容器状态     : docker compose -f $COMPOSE_FILE ps
  后端日志     : docker compose -f $COMPOSE_FILE logs -f server

【LiteLLM 端口过渡说明】
  宿主机 4000 已被旧 litellm 占用，本项目 litellm 暂映射到 ${LITELLM_PORT_NUM}；
  客户端接入统一走 nginx 反代 ${LITELLM_PUBLIC_URL:-/llm}，与端口无关。
  待旧实例停用后执行两步即切回标准端口:
    1) 编辑 .env: LITELLM_HOST_PORT=4000
    2) docker compose -f $COMPOSE_FILE --env-file .env up -d
============================================================
EOF
