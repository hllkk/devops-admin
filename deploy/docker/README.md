# DevOps Admin Docker 部署指南

## 目录结构

```
deploy/docker/
├── Dockerfile              # 后端多阶段构建
├── docker-compose.yml      # 容器编排
├── .env                    # 环境变量（密码）
├── bin/                    # 二进制包（不提交 Git）
│   ├── jellyfin-ffmpeg*.tar.xz
│   ├── ImageMagick-*.tar.xz  (可选，当前用 apt)
│   └── .gitignore
├── conf/
│   ├── nginx.conf          # Nginx 配置
│   └── config.yaml         # 后端 Docker 专用配置
└── scripts/
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

```bash
cd frontend
pnpm install
pnpm build
# 产物在 frontend/dist/ 目录
```

### 3. 修改环境变量

```bash
cd deploy/docker
cp .env.example .env
# 编辑 .env 修改所有默认密码
```

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

Dockerfile 使用三阶段构建：

1. **golang:1.26-bookworm** — 编译 Go 二进制
2. **debian:bookworm-slim** — 运行时镜像，包含：
   - FFmpeg/FFprobe（从 .tar.xz 解压，静态编译）
   - ImageMagick（apt 安装，6.9 版本）
   - ExifTool（apt 安装，perl 模块）

最终镜像约 350-400MB。

## 配置覆盖

`docker-compose.yml` 中将 `conf/config.yaml` 挂载为只读卷，覆盖后端默认配置。
关键覆盖项：

- MySQL/Redis 连接地址改为 Docker 服务名（mysql、redis）
- RustFS endpoint 改为 `rustfs:9000`
- 本地存储路径改为 `/app/storage`

## 故障排查

### 后端启动失败

```bash
# 查看后端日志
docker compose logs backend

# 常见问题：
# - MySQL 未就绪：等待 healthcheck 通过
# - 配置文件错误：检查 conf/config.yaml
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
