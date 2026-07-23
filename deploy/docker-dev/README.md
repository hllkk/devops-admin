# Dev 环境 Docker 依赖服务

一键启动开发环境所需的 **PostgreSQL**、**Redis** 与 **RustFS**（S3 兼容对象存储，替代 MinIO），供宿主机上 `go run` 运行的 server 通过 `127.0.0.1` 直连。

## 前置

- Docker（已验证 29.x）
- Docker Compose v2（`docker compose` 子命令，已验证 v5.x）

## 镜像拉取（国内网络）

若拉取 `docker.io` 超时（`registry-1.docker.io` 不可达），可用加速器拉取后 retag 为官方名：

```bash
docker pull docker.m.daocloud.io/library/postgres:18-alpine
docker tag  docker.m.daocloud.io/library/postgres:18-alpine postgres:18-alpine
docker pull docker.m.daocloud.io/library/redis:8-alpine
docker tag  docker.m.daocloud.io/library/redis:8-alpine redis:8-alpine
docker pull docker.m.daocloud.io/rustfs/rustfs:latest
docker tag  docker.m.daocloud.io/rustfs/rustfs:latest rustfs/rustfs:latest
docker pull docker.m.daocloud.io/minio/mc:latest
docker tag  docker.m.daocloud.io/minio/mc:latest minio/mc:latest
```

或一劳永逸：在 `/etc/docker/daemon.json` 配置 `registry-mirrors` 后 `systemctl restart docker`。

## 文件结构

```
deploy/docker-dev/
├── docker-compose.yml        # 服务定义
├── config/
│   ├── postgresql.conf       # PostgreSQL 主配置（可编辑）
│   └── redis.conf            # Redis 主配置（可编辑）
└── README.md
```

## 快速开始

```bash
# 启动（首次会自动在 /home/docker/docker-dev 下创建数据目录并初始化）
docker compose -f deploy/docker-dev/docker-compose.yml up -d

# 查看状态（等 status 变为 healthy）
docker compose -f deploy/docker-dev/docker-compose.yml ps
```

## 常用命令

```bash
# 停止（保留数据）
docker compose -f deploy/docker-dev/docker-compose.yml down

# 改完 config/*.conf 后重载（数据不丢）
docker compose -f deploy/docker-dev/docker-compose.yml restart

# 查看日志
docker logs -f devops-pgsql-dev
docker logs -f devops-redis-dev
```

> 下文示例用 `DC="docker compose -f deploy/docker-dev/docker-compose.yml"` 可简化。

## 连接信息（默认值）

| 服务 | 地址 | 用户 / 密码 | 说明 |
|---|---|---|---|
| PostgreSQL | `127.0.0.1:5432` | `postgres` / `postgres` | 超级用户；业务库由 server 初始化向导创建 |
| Redis | `127.0.0.1:6379` | 无密码 | 对齐 `server/config.yaml` 默认 |
| RustFS | `127.0.0.1:9000` | `rustfsadmin` / `rustfsadmin` | S3 兼容对象存储；bucket `devops-admin`，对齐 `server/config.yaml` 的 `minio` 段，console 同端口 |

## 与 server 的对接（开箱即用）

`server/config.yaml` 已预设好 dev 默认值，宿主机 `go run` 直连：
- Redis：`127.0.0.1:6379` 无密码
- RustFS：`minio` 段已指向 `127.0.0.1:9000`，`system.oss-type` 已设为 `minio`，文件上传自动走 rustfs（bucket `devops-admin` 由 `rustfs_init` 建桶并设匿名下载）

本地启动 server：

```bash
cd server && go run main.go
```

在初始化向导中填写数据库：
- 类型 `pgsql`，host `127.0.0.1`，port `5432`，user `postgres`，password `postgres`
- 需要用 redis 时，把 `server/config.yaml` 的 `system.use-redis` 改为 `true`
- 暂不用对象存储、想切回本地磁盘时，把 `system.oss-type` 改回 `local`（用 `minio` 前必须先 `docker compose up -d` 起 rustfs，否则上传会失败）

## 网络

依赖服务同处 `dev-net` bridge 网络（`docker compose up -d` 自动创建，实际名 `docker-dev_dev-net`），服务间用服务名互访（如 `rustfs_init` → `rustfs:9000`）。当前 server 跑宿主机、走端口映射 `127.0.0.1` 直连，无需关心网络；未来把 server 容器化时，让其加入 `dev-net` 即可改用 `pgsql:5432` / `redis:6379` / `rustfs:9000` 访问。

## 修改配置

直接编辑 `config/postgresql.conf` 或 `config/redis.conf`（文件内含中文注释），然后重启对应服务：

```bash
docker compose -f deploy/docker-dev/docker-compose.yml restart pgsql   # 只重启 pg
docker compose -f deploy/docker-dev/docker-compose.yml restart redis   # 只重启 redis
```

## 覆盖默认参数

版本 / 端口 / 凭据 / 数据路径在 `docker-compose.yml` 中以 `${VAR:-default}` 内联，可用环境变量临时覆盖：

```bash
PG_PORT=5433 PG_VERSION=17 \
  docker compose -f deploy/docker-dev/docker-compose.yml up -d
```

如需持久化覆盖，在本目录新建 `.env` 文件（compose 自动读取）：

```dotenv
PG_VERSION=17
PG_PORT=5432
PG_USER=postgres
PG_PASSWORD=postgres
PG_DATA_PATH=/home/docker/docker-dev/pgsql
REDIS_VERSION=8
REDIS_PORT=6379
REDIS_DATA_PATH=/home/docker/docker-dev/redis
RUSTFS_VERSION=latest
RUSTFS_PORT=9000
RUSTFS_CONSOLE_PORT=9001
RUSTFS_ROOT_USER=rustfsadmin
RUSTFS_ROOT_PASSWORD=rustfsadmin
RUSTFS_BUCKET=devops-admin
RUSTFS_DATA_PATH=/home/docker/docker-dev/rustfs
```

## 数据目录

- PostgreSQL：`/home/docker/docker-dev/pgsql`（PG 18 数据落在其下 `18/docker/` 子目录）
- Redis：`/home/docker/docker-dev/redis`
- RustFS：`/home/docker/docker-dev/rustfs`（容器内挂到 `/data`，以 root 运行无需修属主）

**彻底重置**（慎用，会清空数据）：停止后删除上述两个目录，再 `up -d` 会重新初始化。

```bash
docker compose -f deploy/docker-dev/docker-compose.yml down -v   # -v 仅清理 compose 命名卷，bind mount 需手动删
sudo rm -rf /home/docker/docker-dev/pgsql /home/docker/docker-dev/redis /home/docker/docker-dev/rustfs
```

## 验证

```bash
docker exec devops-pgsql-dev pg_isready -U postgres        # 期望: accepting connections
docker exec devops-redis-dev redis-cli ping                # 期望: PONG
docker compose -f deploy/docker-dev/docker-compose.yml logs rustfs_init | tail -n 2   # 期望: 末行出现 "rustfs bucket devops-admin 就绪"
# rustfs console：浏览器打开 http://127.0.0.1:9000 ，用 rustfsadmin / rustfsadmin 登录查看 bucket
```
