# syntax=docker/dockerfile:1
# ============================================================================
# devops-admin 在线升级执行器镜像（多阶段）
#
#   构建上下文（compose context）：仓库 deploy/updater/ 目录（Dockerfile 在 context 内）
#   构建：golang:1.26-alpine（零第三方依赖，秒级编译）
#   运行：alpine:3.20 + docker-cli + docker-compose-plugin（经 exec 操作宿主 docker）
#
# 权限说明：必须以 root 运行——要访问 /var/run/docker.sock 与安装目录里
# root 属主的 .env(600)；风险面已收敛：不对宿主暴露端口（仅 prod-net 内网）、
# 写接口有 UPDATER_TOKEN 鉴权。
# ============================================================================

# ----------------------------------------------------------------------------
# 阶段 1：编译静态二进制
# ----------------------------------------------------------------------------
FROM golang:1.26-alpine AS builder

ARG GOPROXY=https://goproxy.cn,direct
# 版本注入：与 web/server 镜像同机制共用 APP_VERSION（版本化 tag 同步）
ARG APP_VERSION=dev
ARG BUILD_TIME=unknown
ENV CGO_ENABLED=0 \
    GOPROXY=${GOPROXY}

WORKDIR /build

# 零第三方依赖：无 go.sum，直接拷源码编译
COPY . .

RUN go build -trimpath \
    -ldflags="-s -w -X 'main.Version=${APP_VERSION}' -X 'main.BuildTime=${BUILD_TIME}'" \
    -o /out/devops-admin-updater .

# ----------------------------------------------------------------------------
# 阶段 2：运行镜像（docker CLI + compose 插件，安装 job 里跑 compose 用）
# 注意：alpine 仓库里 compose v2 插件的包名是 docker-cli-compose（非 docker-compose-plugin）
# ----------------------------------------------------------------------------
FROM alpine:3.20

# 国内构建走阿里云 alpine 镜像源（官方 dl-cdn 国内极慢），与 daemon.json 加速器实践一致
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache docker-cli docker-cli-compose ca-certificates

COPY --from=builder /out/devops-admin-updater /app/updater

ENTRYPOINT ["/app/updater"]
