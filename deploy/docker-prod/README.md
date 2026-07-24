# 生产环境 Docker 全栈部署

一键编排 **devops-admin** 生产环境：nginx(前端) + server(后端) + PostgreSQL + Redis + RustFS（S3 兼容对象存储，与 dev 一致）。

借鉴 `/home/remote/devops-admin` 的做法，**用户只维护一份 `.env`**：

- 后端 `initialize/other.go` 的 `applyEnvOverrides()` 启动时用环境变量覆盖 `config.yaml` 的敏感项（DB/Redis/JWT/RustFS 密码、可信反代），`config.yaml` 静态入库、无需编辑；
- 首次初始化由 `/init/autoInitDB` 用 config 的 DB 配置 + `INIT_ADMIN_PASSWORD` 自动建库/建表/建管理员，无需向导手填。

## 前置

- Docker 24+（默认 BuildKit）、Compose v2
- BuildKit 必须可用（Dockerfile 统一放在本目录，经 BuildKit 引用 context 之外的 Dockerfile；旧版可 `export DOCKER_BUILDKIT=1`）

## 文件结构

```
deploy/docker-prod/
├── docker-compose.yml                # 全栈编排（web + server + pg + redis + rustfs）
├── .env.example                      # 唯一需维护的机密模板（copy 成 .env）
├── Dockerfile.server / Dockerfile.web
├── nginx/nginx.conf                  # SPA 托管 + /proxy-default 反代 + /oss/ 对象存储反代
└── config/
    ├── config.yaml                   # 静态入库后端配置（密码占位，由 env 覆盖，无需编辑）
    ├── postgresql.conf
    └── redis.conf
```

> `.dockerignore` 因 Docker 机制必须留在各自 build context 根：`server/.dockerignore`、`web/.dockerignore`；
> `web/.env.prod` 是 Vite 构建期读取的文件，也必须留在 `web/` 源码树。

## 快速开始

```bash
cd deploy/docker-prod

# 1) 生成唯一机密文件
cp .env.example .env
#   编辑 .env，填入必填项（留空则 docker compose up 报错）：
#     JWT_SIGNING_KEY        openssl rand -hex 32（≥32 字符，生产强制）
#     INIT_ADMIN_PASSWORD    首次自动初始化的超级管理员密码
#     POSTGRES_PASSWORD      PG 密码
#     REDIS_PASSWORD         Redis 密码
#     RUSTFS_ROOT_PASSWORD   RustFS 密码（≥8）
#   config.yaml 无需编辑（密码由 env 覆盖）

# 2) 准备数据目录
sudo mkdir -p /home/docker/docker-prod/{pgsql,redis,rustfs,uploads,server-log}

# 3) 构建并启动
docker compose up -d --build
```

## 首次初始化（自动）

启动后：

1. PG 容器用 `POSTGRES_DB` 建业务库 `devops_admin`；Redis/RustFS 按 `.env` 密码启动
2. server 启动：`applyEnvOverrides` 用 `.env` 覆盖 config 密码 → 直连 PG 成功（`OPS_DB!=nil`）→ `RegisterTables` 自动建空表
3. 前端检测 `SysUser` 无数据，进入初始化页，显示「自动初始化」按钮（`CheckDB` 返回 `autoInit=true`）
4. 点击按钮（或 `POST /init/autoInitDB`）：用 config 的 DB 配置 + `INIT_ADMIN_PASSWORD` 自动建管理员
5. 用管理员账号登录

也可跳过前端直接触发：

```bash
curl -X POST http://<宿主IP>/proxy-default/init/autoInitDB
```

> 若前端尚未适配 `autoInit` 字段，直接调上面的接口即可完成初始化。

## 配置维护（只 .env）

`.env` 是唯一机密源；`config.yaml` 静态入库、由 env 覆盖，无需手动同步：

| 项 | 维护位置 | 说明 |
|---|---|---|
| `JWT_SIGNING_KEY` | `.env`（env 覆盖 config） | ≥32 字符；`GIN_MODE=release` 下弱/默认密钥会 Fatal |
| `POSTGRES_PASSWORD` / `REDIS_PASSWORD` | `.env`（env 覆盖 config） | 不必在 config.yaml 同步 |
| `RUSTFS_ROOT_USER` / `RUSTFS_ROOT_PASSWORD` | `.env`（env 覆盖 config 的 minio 段） | 对象存储凭据 |
| `INIT_ADMIN_PASSWORD` | `.env` | 首次 autoInitDB 建管理员 |
| `TRUSTED_PROXIES` | `.env` | 反代部署必配，否则 ClientIP 取直连地址 |

## 安全设计要点

- **DB / Redis / RustFS 端口默认不对宿主暴露**，仅 `prod-net` 内服务名互访；前端访问对象存储文件经 nginx `/oss/` 同源反代。
- 仅 `web`(80) 对外；`SERVER_PORT` 可在 `.env` 注释掉以彻底不暴露后端。
- server 以**非 root** `app` 用户运行；Redis 密码经 `--requirepass` 注入；nginx 关 `server_tokens` + 基础安全头。
- 生产 `GIN_MODE=release` 下强制 `JWT_SIGNING_KEY` 非默认且 ≥32（other.go `validateProductionSecurity`）。
- 机密只在 `.env`（已 gitignore），`config.yaml` 仅含占位默认值。

## 对象存储说明

生产沿用 dev 的 **RustFS**（S3 兼容，`oss-type=minio`，minio-go 客户端直连，业务代码零改动）：

- rustfs 端口**不对宿主暴露**；前端访问上传文件统一经 nginx 的 `/oss/` 同源反代（`bucket-url` 配为 `/oss/devops_admin`）。
- 建桶与匿名下载策略由 `rustfs_init` 一次性完成，故当前模式下 bucket 内文件**公开可下载**；如需鉴权访问，请在 server 代码层改用预签名 URL。
- RustFS 官方标注仍处于快速迭代期，请关注其生产就绪状态；镜像 tag 建议在 `.env` 固定为具体版本而非 `latest`。

## 常用命令

```bash
DC="docker compose -f deploy/docker-prod/docker-compose.yml"

$DC up -d --build        # 构建并启动
$DC ps                   # 查看状态
$DC logs -f server       # 跟随后端日志
$DC restart server       # 仅重启后端
$DC down                 # 停止（保留数据）
$DC down -v              # 停止并清理命名卷（bind mount 需手动删）
```

## 镜像拉取加速（国内网络）

若 `docker.io` 不可达，参考 `deploy/docker-dev/README.md`：用加速器拉取后 retag 为官方名，或在 `/etc/docker/daemon.json` 配置 `registry-mirrors` 后重启 docker。

## 数据与备份

- PostgreSQL：`${PG_DATA_PATH}`（PG 18 数据落在其下 `18/docker/`）
- Redis：`${REDIS_DATA_PATH}`（RDB + AOF）
- RustFS：`${RUSTFS_DATA_PATH}`
- 上传文件：`${SERVER_UPLOADS_PATH}`（oss-type=local 时）
- 后端日志：`${SERVER_LOG_PATH}`

默认路径均在 `/home/docker/docker-prod/`，可在 `.env` 覆盖。

## HTTPS

本编排仅起 80 端口。生产 HTTPS 建议由**外层反代/负载均衡**（宿主 nginx、Caddy、云 SLB）终结 TLS 后再转给本编排的 80 端口；若需在 web 容器内直接终结 TLS，自行挂载证书并在 `nginx.conf` 增加 443 server 段。

## 与 docker-dev 的区别

| 维度 | docker-dev | docker-prod |
|---|---|---|
| 应用 | server 跑宿主机 `go run` | server/web 全部容器化 |
| 配置来源 | `server/config.yaml`（向导回写） | 静态 `config.yaml` + env 覆盖（**只维护 .env**） |
| 初始化 | `/init` 向导手填 DB/Redis/AdminPassword | `/init/autoInitDB` 自动（`INIT_ADMIN_PASSWORD`） |
| 对象存储 | RustFS（宿主机 9000 直连） | RustFS（端口不对外，下载经 nginx /oss/ 反代） |
| Redis | 无密码 | requirepass（.env 注入） |
| 端口暴露 | 全部映射到 127.0.0.1 | 仅 web(80) 对外，其余内网 |
| 运行用户 | — | server 非 root |
