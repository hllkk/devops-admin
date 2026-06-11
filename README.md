# DevOps Admin

运维管理系统 - 前后端分离架构

## 项目结构

```
devops-admin/
├── frontend/     # 前端 (Vue3 + SoybeanAdmin)
├── backend/      # 后端 (Gin + GORM)
└── deploy/       # 部署配置
```

## 快速开始

### 克隆项目（包含 submodule）

```bash
git clone --recursive git@github.com:hllkk/devops-admin.git
```

### 前端

```bash
cd frontend
pnpm install
pnpm dev
```

### 后端

```bash
cd backend
go mod tidy
go run main.go
```

## 同步上游更新

### 前端（同步 SoybeanAdmin）

```bash
cd frontend
git fetch upstream
git merge upstream/main
```

### 后端

直接在 backend 目录提交即可。

## 技术栈

- **前端**: Vue3 + TypeScript + Naive UI + SoybeanAdmin
- **后端**: Go + Gin + GORM

## 分支策略

| 分支 | 用途 | 说明 |
|------|------|------|
| `main` | 上游同步 | 保持与 SoybeanAdmin 同步，不直接修改 |
| `dev` | 项目开发 | 日常开发分支，功能完成后可合并到 main |
| `feature/*` | 功能开发 | 新功能开发 |

### 同步流程

```
upstream/main → main → dev
```

上游更新会自动同步到 `main`，然后合并到 `dev` 分支。

---

## 生产环境部署

### 架构概览

```
                        ┌─────────────────────────────────────────┐
                        │            Docker 容器网络               │
                        │                                         │
  ┌──────────┐  HTTPS   │  ┌──────────┐       ┌──────────────┐   │
  │ 外部      │ ────────► │  │ 内部      │ ────► │ Go 后端       │   │
  │ Nginx    │  :443    │  │ Nginx    │ :80   │ (ops-backend) │   │
  │ (SSL终结) │          │  │ (反向代理)│       │ :8888         │   │
  └──────────┘          │  └──────────┘       └──────────────┘   │
                        │                                         │
                        │  ┌──────────┐  ┌───────┐  ┌─────────┐  │
                        │  │ MySQL    │  │ Redis │  │ RustFS  │  │
                        │  │ :3306    │  │ :6379 │  │ :9000   │  │
                        │  └──────────┘  └───────┘  └─────────┘  │
                        └─────────────────────────────────────────┘
```

- **外部 Nginx**：负责 SSL 证书管理、域名绑定、反向代理到 Docker 内部 Nginx
- **内部 Nginx**：Docker 容器内反向代理，分发前端静态资源和后端 API

### 1. 外部 Nginx 配置

在部署机上配置外部 Nginx，将域名反向代理到 Docker 内部 Nginx：

```nginx
server {
    listen 443 ssl http2;
    server_name your-domain.com;  # 替换为实际域名，如 j.chinargb.com.cn

    # SSL 证书
    ssl_certificate     /etc/nginx/ssl/your-domain.com.pem;
    ssl_certificate_key /etc/nginx/ssl/your-domain.com.key;
    ssl_protocols       TLSv1.2 TLSv1.3;

    # 反向代理到 Docker 内部 Nginx
    location / {
        proxy_pass http://127.0.0.1:80;  # Docker Nginx 映射的宿主机端口
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;  # 关键：传递 HTTPS 协议头

        # 文件上传支持
        client_max_body_size 500m;
        proxy_read_timeout 600s;
        proxy_send_timeout 600s;
    }
}

# HTTP 自动跳转 HTTPS
server {
    listen 80;
    server_name your-domain.com;
    return 301 https://$host$request_uri;
}
```

> **关键配置**：`proxy_set_header X-Forwarded-Proto $scheme;` 是企微登录和 Cookie 安全标志正确工作的前提。内部 Nginx 已通过 `map` 指令处理双层代理的协议头透传。

### 2. 企业微信登录配置

#### 2.1 获取企微应用凭证

在[企业微信管理后台](https://work.weixin.qq.com/wework_admin/frame)获取以下信息：

| 参数 | 说明 | 获取位置 |
|------|------|----------|
| `Corp ID` | 企业 ID | 我的企业 → 企业信息 → 企业ID |
| `Agent ID` | 应用 ID | 应用管理 → 自建应用 → AgentId |
| `Agent Secret` | 应用密钥 | 应用管理 → 自建应用 → Secret |

#### 2.2 配置授权回调域

在企业微信管理后台 → 应用管理 → 自建应用 → 网页授权及JS-SDK → **可信域名** 中填入：

```
your-domain.com
```

> 仅填写域名，不带协议和端口。

#### 2.3 配置环境变量

编辑 `deploy/docker/.env.local`（不存在则创建）：

```bash
# 企业微信登录配置
WECOM_CORP_ID=ww1234567890abcdef          # 替换为你的企业 ID
WECOM_AGENT_SECRET=your-agent-secret      # 替换为应用密钥
WECOM_AGENT_ID=1000106                     # 替换为应用 AgentId
WECOM_REDIRECT_URI=https://your-domain.com/wecomCallback  # 替换为实际域名
WECOM_ENABLED=true                         # 启用企微登录
COOKIE_SECURE=true                         # HTTPS 环境必须设为 true
```

> `.env.local` 已被 `.gitignore` 忽略，不会提交到 git。

#### 2.4 同步 config.yaml 中的 redirect-uri

编辑 `deploy/docker/conf/config.yaml`，确保 `wecom` 部分的 `redirect-uri` 与 `.env.local` 一致：

```yaml
wecom:
  enable-work-wechat: true
  corp-id: ""           # 留空，由环境变量 OPS_WECOM_CORP_ID 覆盖
  agent-secret: ""      # 留空，由环境变量 OPS_WECOM_AGENT_SECRET 覆盖
  agent-id: 1000106     # 默认值，由环境变量 OPS_WECOM_AGENT_ID 覆盖
  redirect-uri: https://your-domain.com/wecomCallback  # 替换为实际域名
```

> 环境变量会覆盖 config.yaml 中的值，但 redirect-uri 字段需要在 config.yaml 中也正确配置，因为后端同时读取两处。

### 3. 企微登录流程

```
┌────────┐    ① 请求扫码     ┌────────┐   ② 返回 OAuth URL    ┌────────┐
│ 浏览器  │ ──────────────► │ 后端    │ ──────────────────► │ 浏览器  │
│ (PC端)  │                  │        │                       │ (PC端)  │
└────────┘                  └────────┘                       └────────┘
                                                                 │
                                                           ③ 展示二维码
                                                                 │
┌────────┐    ④ 扫码确认    ┌────────┐                           │
│ 企微App │ ──────────────► │ 企微   │ ◄──── ④ 用户扫码 ────────┘
│ (手机)  │                  │ 服务器  │
└────────┘                  └────────┘
                                 │
                           ⑤ 回调 redirect-uri
                           GET /wecomCallback?code=xxx&state=yyy
                                 │
┌────────┐    ⑥ 轮询状态     ┌────────┐   ⑤ 请求到达           ┌────────┐
│ 浏览器  │ ──────────────► │ 后端    │ ◄──────────────────── │ 外部    │
│ (PC端)  │ ◄────────────── │        │ ◄──────────────────── │ Nginx  │
└────────┘   ⑦ 返回 Token   └────────┘   → 内部 Nginx        └────────┘
```

1. 前端请求 `GET /api/v1/auth/wecomLogin` 获取 OAuth URL
2. 后端生成带 `sceneId` 的 OAuth URL 返回给前端
3. 前端展示二维码，用户用企微扫码
4. 企微服务器回调 `https://your-domain.com/wecomCallback?code=xxx&state=sceneId`
5. 请求经 **外部 Nginx → 内部 Nginx → 后端** 到达
6. 后端通过 `code` 获取企微用户信息，完成登录，更新 Redis 状态
7. 前端轮询 `GET /api/v1/auth/qrCodeStatus?sceneId=xxx` 获取登录 Token

### 4. 环境变量参考

| 环境变量 | 对应配置 | 必填 | 说明 |
|----------|----------|------|------|
| `WECOM_CORP_ID` | `wecom.corp-id` | 是 | 企业 ID |
| `WECOM_AGENT_SECRET` | `wecom.agent-secret` | 是 | 应用密钥 |
| `WECOM_AGENT_ID` | `wecom.agent-id` | 是 | 应用 AgentId |
| `WECOM_REDIRECT_URI` | `wecom.redirect-uri` | 是 | OAuth 回调地址（HTTPS） |
| `WECOM_ENABLED` | `system.enable-wecom-login` | 是 | 启用企微登录（`true`/`false`） |
| `COOKIE_SECURE` | `system.cookie-secure` | HTTPS 必填 | Cookie Secure 标志（`true`） |

### 5. 启动服务

```bash
cd deploy/docker

# 构建前端（首次或更新后）
cd ../../frontend && pnpm build && cd ../deploy/docker

# 启动所有服务
docker compose up -d

# 查看后端日志确认企微配置加载
docker logs ops-backend 2>&1 | grep -i wecom
```

### 6. 验证清单

- [ ] 外部 Nginx SSL 证书有效，`https://your-domain.com` 可访问
- [ ] 外部 Nginx 正确传递 `X-Forwarded-Proto: https`
- [ ] 企微管理后台可信域名已配置
- [ ] `.env.local` 中 `WECOM_ENABLED=true`
- [ ] `.env.local` 中 `COOKIE_SECURE=true`
- [ ] 登录页面显示企微扫码入口
- [ ] 扫码后回调 URL 可达（检查 `https://your-domain.com/wecomCallback`）
- [ ] 登录成功后 Token 正确写入 Cookie

## BUG收集
```

```

### 启动ClaudeCode
```
IS_SANDBOX=1 claude -c --dangerously-skip-permissions
```
