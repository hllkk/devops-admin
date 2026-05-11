# ============================================
# 阶段1: 构建 Go 后端
# ============================================
FROM golang:1.26-bookworm AS builder

WORKDIR /build

# 先复制依赖文件，利用 Docker 缓存
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# 复制源码并编译
COPY backend/ .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o devops-admin .

# ============================================
# 阶段2: 最终运行镜像
# ============================================
FROM debian:bookworm-slim

# 安装运行时依赖：ImageMagick + exiftool + ca证书 + 时区
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    imagemagick \
    perl \
    libimage-exiftool-perl \
    tzdata \
    curl \
    && rm -rf /var/lib/apt/lists/*

# 设置时区
ENV TZ=Asia/Shanghai

# 复制 FFmpeg 二进制（jellyfin-ffmpeg 静态编译，解压即用）
# 将 jellyfin-ffmpeg*.tar.xz 放到 deploy/docker/bin/ 目录
COPY deploy/docker/bin/jellyfin-ffmpeg*.tar.xz /tmp/
RUN mkdir -p /usr/local/src/ffmpeg && \
    cd /usr/local/src/ffmpeg && \
    tar xf /tmp/jellyfin-ffmpeg*.tar.xz && \
    ln -sf /usr/local/src/ffmpeg/ffmpeg /usr/local/bin/ffmpeg && \
    ln -sf /usr/local/src/ffmpeg/ffprobe /usr/local/bin/ffprobe && \
    rm -f /tmp/jellyfin-ffmpeg*.tar.xz

# 复制后端二进制
COPY --from=builder /build/devops-admin /app/devops-admin

# 复制默认配置文件（docker-compose 会用卷覆盖 config.yaml）
COPY backend/conf/ /app/conf/

WORKDIR /app

# 创建数据目录
RUN mkdir -p /app/storage /app/log

EXPOSE 8888

CMD ["./devops-admin"]
