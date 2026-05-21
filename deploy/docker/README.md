# DevOps Admin Docker 部署指南

## 目录结构

```
deploy/docker/
├── Dockerfile              # 后端多阶段构建
├── docker-compose.yml      # 容器编排
├── .env                    # 环境变量（密码）
├── bin/                    # 二进制包（不提交 Git）
│   ├── jellyfin-ffmpeg*.tar.xz
│   └── .gitignore
├── conf/
│   ├── nginx.conf          # Nginx 配置
│   └── config.yaml         # 后端完整配置（Docker 专用）
└── scripts/
    ├── init-mysql.sql      # MySQL 初始化占位
    └── init-rustfs.sh      # RustFS 初始化脚本
```

## 前置准备

### 1. 准备 FFmpeg 二进制包

将 jellyfin-ffmpeg 的 .tar.xz 包放到 `bin/` 目录：

```bash
# 示例：从当前服务器复制
cp /usr/local/src/ffmpeg/jellyfin-ffmpeg_*.tar.xz deploy/docker/bin/
```

### 2. 构建前端

**必须在 `docker compose up` 之前完成**，否则 Nginx 返回 403。

```bash
cd frontend
pnpm install
pnpm build
# 产物在 frontend/dist/ 目录
cd ../..
```

验证前端已构建：
```bash
ls frontend/dist/index.html
```

### 3. 修改环境变量

```bash
cd deploy/docker
cp .env.example .env
# 编辑 .env 修改所有默认密码
```

> **重要：修改 .env 中的密码后，必须同步修改 conf/config.yaml 中对应的值！**
>
> | .env 变量 | config.yaml 对应位置 |
> |-----------|---------------------|
> | MYSQL_ROOT_PASSWORD | mysql.password |
> | REDIS_PASSWORD | redis.password, redis-list[0].password |
> | RUSTFS_ROOT_USER | rustfs.access-key-id |
> | RUSTFS_ROOT_PASSWORD | rustfs.secret-access-key |

## 启动服务

### 首次部署

```bash
cd deploy/docker

# 1. 构建并启动所有服务
docker compose up -d --build

# 2. 等待 MySQL 和 Redis 健康检查通过
docker compose ps

# 3. 初始化 RustFS（创建存储桶）
bash scripts/init-rustfs.sh

# 4. 访问服务
#    前端: http://your-server/
#    后端 API: http://your-server/api/
#    RustFS 控制台: http://your-server:9001
```

### 日常管理

```bash
# 查看日志
docker compose logs -f backend
docker compose logs -f nginx

# 重启后端（代码更新后）
docker compose up -d --build backend

# 停止所有服务
docker compose down

# 停止并清除数据卷（危险！会丢失数据）
docker compose down -v
```

## 容器清单

| 服务 | 端口 | 说明 |
|------|------|------|
| nginx | 80 | 反向代理 + 前端静态资源 |
| backend | 8888（内部） | Go 后端 |
| mysql | 3306 | 主数据库 |
| redis | 6379 | 缓存 |
| onlyoffice | 8443 | 文档在线编辑 |
| rustfs | 9000, 9001 | 对象存储 |

## 数据持久化

所有数据通过 Docker 命名卷持久化：

| 卷名 | 用途 |
|------|------|
| mysql-data | MySQL 数据 |
| redis-data | Redis 持久化 |
| rustfs-data | RustFS 对象数据 |
| onlyoffice-data | OnlyOffice 缓存 |
| onlyoffice-logs | OnlyOffice 日志 |
| app-storage | 后端本地文件存储 |
| app-logs | 后端日志 |

## 后端构建说明

Dockerfile 使用两阶段构建：

1. **golang:1.26-bookworm** — 编译 Go 二进制
2. **debian:bookworm-slim** — 运行时镜像，包含：
   - FFmpeg/FFprobe（从 .tar.xz 解压，静态编译）
   - ImageMagick（apt 安装）
   - ExifTool（apt 安装）
   - 中文字体（Noto CJK）

最终镜像约 350-400MB。

仓库根目录的 `.dockerignore` 确保构建时排除 `storage/`、`frontend/` 等不需要的目录。

## 配置说明

`conf/config.yaml` 是后端的完整配置文件，已针对 Docker 环境调整：
- MySQL/Redis/RustFS 连接地址使用 Docker 服务名
- 本地存储路径设为 `/app/storage`（通过命名卷持久化）
- 日志目录设为 `/app/log`
- 已启用 RustFS 作为 OSS 存储后端

## 故障排查

### 后端启动失败

```bash
# 查看后端日志
docker compose logs backend

# 常见问题：
# - MySQL 未就绪：等待 healthcheck 通过
# - 配置文件错误：检查 conf/config.yaml 是否完整
# - 端口绑定失败：确认 8888 端口未被占用
```

### Nginx 403 Forbidden

```bash
# 前端未构建
ls frontend/dist/index.html
# 如果不存在，执行：
cd frontend && pnpm install && pnpm build && cd ../..
```

### FFmpeg/ImageMagick 检查失败

```bash
# 进入容器验证
docker compose exec backend sh
ffmpeg -version
convert -version
```

### Nginx 502 Bad Gateway

```bash
# 后端未启动或端口错误
docker compose ps backend
docker compose logs backend
```

### RustFS 连接失败

```bash
# 检查 RustFS 健康状态
curl http://localhost:9000/minio/health/live

# 检查后端配置中的 endpoint
docker compose exec backend cat conf/config.yaml | grep rustfs
```

### OnlyOffice 无法编辑

```bash
# OnlyOffice 启动较慢（约2分钟），检查健康状态
docker compose ps onlyoffice

# 查看日志
docker compose logs onlyoffice
```
