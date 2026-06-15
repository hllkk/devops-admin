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

本项目生产环境运行于**内网**，通过一台**公网网关机**上的外部 Nginx 暴露到公网，采用**双层 Nginx + 非标准端口（:3000）**架构：

```
┌─ 公网 ───────────────────────────────────────────────────────────────────────┐
│                                                                               │
│   用户浏览器                                                                   │
│      │                                                                        │
│      │  HTTPS :3000   (https://your-domain.com:3000)                          │
│      ▼                                                                        │
│   ┌──────────────────────────┐                                                │
│   │  外部 Nginx (公网网关机)   │  ◄── SSL 证书终结 / WebSocket 透传              │
│   │                          │  ◄── 企微域名校验文件 (WW_verify_*.txt)         │
│   └────────────┬─────────────┘                                                │
└────────────────┼──────────────────────────────────────────────────────────────┘
                 │  HTTP :80   proxy_pass http://<DOCKER_HOST_IP>:80
┌─ 内网 ─────────┼──────────────────────────────────────────────────────────────┐
│                 ▼                                                              │
│   ┌──────────────────────────┐    路径分发                                     │
│   │  内部 Nginx (容器 :80)    │──────► backend:8888    (/api、/wecomCallback)  │
│   │  + 前端静态资源 (dist)    │──────► onlyoffice:80   (/office，①SDK 加载)   │
│   └──────────────────────────┘──────► 前端 dist        (/)                    │
│                                                                               │
│   OnlyOffice 容器 ──②拉文档 / ③回调保存──► backend:8888  (走 docker 内网)      │
│                                                                               │
│   MySQL:3306 │ Redis:6379 │ RustFS:9000                                       │
└───────────────────────────────────────────────────────────────────────────────┘
```

- **外部 Nginx（公网网关机）**：SSL 证书终结、域名绑定、反向代理到内网 Docker 宿主机，负责 WebSocket 升级头与 `X-Forwarded-Proto` 透传
- **内部 Nginx（Docker 容器）**：按路径分发前端静态资源、后端 API、OnlyOffice 文档服务
- **非标准端口 :3000**：对外通过 `https://your-domain.com:3000` 访问；企业微信可信域名需**显式带端口**登记

### 1. 外部 Nginx 配置（公网网关机）

在一台**独立的公网网关机**上配置外部 Nginx，将 `https://your-domain.com:3000` 反向代理到内网 Docker 宿主机的内部 Nginx（`:80`）。

> 替换 `your-domain.com` 为实际域名（如 `j.chinargb.com.cn`），`<DOCKER_HOST_IP>` 为内网 Docker 宿主机 IP。

```nginx
# WebSocket 升级映射（OnlyOffice 实时协作必需，需放在 server 块之外、全局生效）
map $http_upgrade $connection_upgrade {
    default upgrade;
    ''      close;
}

# HTTP:80 → HTTPS:3000 跳转（可选，方便用户直接输入域名访问）
server {
    listen 80;
    server_name your-domain.com;
    return 301 https://$host:3000$request_uri;
}

# HTTPS 主服务（非标准端口 3000）
server {
    listen 3000 ssl http2;
    server_name your-domain.com;

    # SSL 证书
    ssl_certificate     /etc/nginx/ssl/your-domain.com.pem;
    ssl_certificate_key /etc/nginx/ssl/your-domain.com.key;
    ssl_protocols       TLSv1.2 TLSv1.3;

    client_max_body_size 500m;   # 文件上传 / OnlyOffice 文档，需与内部 Nginx 一致

    # 统一反向代理到内网 Docker 宿主机的内部 Nginx（路径分发由内部 Nginx 完成）
    location / {
        proxy_pass http://<DOCKER_HOST_IP>:80;

        proxy_set_header Host              $host;
        proxy_set_header X-Real-IP         $remote_addr;
        proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;   # ★企微 Cookie + 内部协议判断的前提

        # ★WebSocket 透传（OnlyOffice 实时编辑必需，缺失会无限转圈）
        proxy_http_version 1.1;
        proxy_set_header Upgrade    $http_upgrade;
        proxy_set_header Connection $connection_upgrade;

        # 大文件上传 / 文档合并超时
        proxy_read_timeout 600s;
        proxy_send_timeout 600s;
    }
}
```

#### 必须注意的 5 个关键点

1. **`listen 3000 ssl`** —— 非标准端口要显式监听；公网访问入口为 `https://your-domain.com:3000`。
2. **`X-Forwarded-Proto $scheme`** —— 内部 Nginx 已通过 `map $http_x_forwarded_proto` 据此判断协议。漏传会导致 Cookie 的 `Secure` 标志失效、企微回调协议判断异常。
3. **WebSocket 三件套**（`map` + `proxy_http_version 1.1` + `Upgrade/Connection`）—— OnlyOffice 打开文档后依赖 WebSocket 维持实时协作，缺失会导致文档加载后卡死。
4. **`client_max_body_size 500m`** —— 必须与内部 Nginx 一致，否则大文件上传在外层被 `413` 截断。
5. **企微域名校验文件** —— 校验文件在「系统设置 → 认证 → 企业微信」录入文件名与内容，由后端动态响应（内部 Nginx 已重写 `/WW_verify_*.txt` → 后端接口），外部 Nginx 透传即可，**无需放置文件、无需重新构建前端**。验证：`https://your-domain.com:3000/WW_verify_xxx.txt` 返回录入内容。

> 若外部 Nginx 与 Docker 在**同一台机器**，将 `proxy_pass` 改为 `http://127.0.0.1:80` 即可（内部 Nginx 在宿主机映射的端口）。

### 2. 企业微信登录配置

企微登录凭证（CorpID / AgentSecret / AgentID / 回调地址）与可信域名校验文件**全部在系统设置页面配置**，存数据库（AES 加密），修改后即时生效、无需重启后端。

#### 2.1 获取企微应用凭证

在[企业微信管理后台](https://work.weixin.qq.com/wework_admin/frame)获取以下信息（后续填入系统设置页面）：

| 参数 | 说明 | 获取位置 |
|------|------|----------|
| `Corp ID` | 企业 ID | 我的企业 → 企业信息 → 企业ID |
| `Agent ID` | 应用 ID | 应用管理 → 自建应用 → AgentId |
| `Agent Secret` | 应用密钥 | 应用管理 → 自建应用 → Secret |

#### 2.2 在系统设置页面配置

登录系统 → **系统设置 → 认证 → 企业微信**，填写并保存：

| 字段 | 值 |
|------|----|
| 启用企业微信 | 打开开关 |
| 企业 ID | 企微 Corp ID |
| 应用 AgentId | 企微 AgentId |
| 应用 Secret | 企微 Agent Secret（加密存储，返回时脱敏；留空保存则保留原值） |
| 回调地址 | `https://your-domain.com:3000/wecomCallback` |

> 保存后即时生效，**无需重启**。Agent Secret 使用 AES-GCM 加密入库（密钥取自 `config.yaml` 的 `jwt.signing-key`），更换 signing-key 前需在此重新录入。

> **可信域名校验文件**：同页面「可信域名校验」折叠区，填入企微后台下载的校验文件名（如 `WW_verify_xxx.txt`）与内容。系统通过 `/WW_verify_*.txt` 动态响应（内部 Nginx 已配置重写），**无需手动放置文件、无需重新构建前端**。

#### 2.3 配置企微后台可信域名

在企业微信管理后台 → 应用管理 → 自建应用 → 网页授权及JS-SDK → **可信域名** 中填入：

```
your-domain.com:3000
```

> ⚠️ **非标准端口必须显式带端口号**。企微要求可信域名与「回调地址」的域名+端口**完全一致**；`:3000` 时必须登记为 `your-domain.com:3000`（不带协议头）。配置时企微会校验 `https://your-domain.com:3000/WW_verify_xxx.txt` 是否可访问（由 2.2 的校验文件配置自动响应）。

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
4. 企微服务器回调 `https://your-domain.com:3000/wecomCallback?code=xxx&state=sceneId`
5. 请求经 **外部 Nginx → 内部 Nginx → 后端** 到达
6. 后端通过 `code` 获取企微用户信息，完成登录，更新 Redis 状态
7. 前端轮询 `GET /api/v1/auth/qrCodeStatus?sceneId=xxx` 获取登录 Token

### 4. OnlyOffice 公网部署配置（系统设置）

> ⚠️ OnlyOffice 配置**存储在数据库**（`sys_setting` 表的 `disk` 设置），通过**系统设置页面**配置，**不在 `config.yaml` 或环境变量中**。这是公网部署最易出错的一环。

#### 4.1 三方通信原理

OnlyOffice 涉及三方通信，公网部署的核心是让「浏览器 → 文档服务器」走公网、「文档服务器 → 后端」走内网：

```
① 浏览器          ──加载 JS SDK──►  https://your-domain.com:3000/office/.../api.js   (公网, 经外部+内部 Nginx)
② OnlyOffice 容器 ──拉取文档──────►  http://backend:8888/api/v1/office/file/:id     (内网, docker 网络)
③ OnlyOffice 容器 ──回调保存──────►  http://backend:8888/api/v1/office/callback      (内网, docker 网络)
```

- ① 的地址由系统设置的 **ServerUrl（文档服务器地址）** 决定，浏览器加载 SDK 用，**必须公网可达**
- ②③ 的地址由系统设置的 **CallbackUrl（回调地址）** 决定，OnlyOffice 容器访问后端用，**用内网服务名**（容器与 `backend` 同属 `ops-net` 网络，且 docker-compose 已设 `ALLOW_PRIVATE_IP_ADDRESS=true`）

#### 4.2 系统设置配置项

登录系统 → 系统设置 → 磁盘设置 → OnlyOffice，按下表配置：

| 配置项 | 推荐值 | 说明 |
|--------|--------|------|
| 文档服务器地址（ServerUrl） | `https://your-domain.com:3000/office` | 浏览器加载 SDK，**必须公网** |
| 回调地址（CallbackUrl） | `http://backend:8888/api/v1` | 容器访问后端，**用内网服务名 `backend`** |
| JWT 密钥（TokenSecret） | 与 docker-compose `JWT_SECRET` 完全一致 | 容器与后端 JWT 校验需一致 |

> docker-compose 中 OnlyOffice 容器已设 `JWT_SECRET=53f82208...`，系统设置的 TokenSecret **必须与之完全一致**，否则文档加载报 JWT 校验失败。

#### 4.3 已知权衡：健康检查

后端的 OnlyOffice 健康检查会访问 `ServerUrl/healthcheck`（即公网地址）。从 Docker 内的后端容器访问公网域名，若网关机与宿主机之间 NAT 回环不通，系统设置页可能显示「OnlyOffice 不可达」。

**这不影响实际编辑**——文档编辑走的是 ①（浏览器→SDK）+ ②③（容器内网→后端），不依赖健康检查。若编辑正常但健康检查标红，可忽略；或确认后端容器到公网网关机 `:3000` 的网络可达性。

---

### 5. 环境变量参考

| 环境变量 | 对应配置 | 必填 | 说明 |
|----------|----------|------|------|
| `COOKIE_SECURE` | `system.cookie-secure` | HTTPS 必填 | Cookie Secure 标志（`true`） |

> 企微登录凭证（CorpID / AgentSecret / AgentID / 回调地址）与可信域名校验文件由「系统设置 → 认证 → 企业微信」页面配置（数据库存储），不再使用环境变量。

### 6. 启动服务

```bash
cd deploy/docker

# 构建前端（首次或更新后）
cd ../../frontend && pnpm build && cd ../deploy/docker

# 启动所有服务
docker compose up -d

# 查看后端日志确认企微配置加载
docker logs ops-backend 2>&1 | grep -i wecom
```

### 7. 验证清单

#### 基础连通性
- [ ] `https://your-domain.com:3000` 可访问，SSL 证书有效
- [ ] HTTP `:80` 正常跳转到 `https://your-domain.com:3000`
- [ ] 外部 Nginx 正确传递 `X-Forwarded-Proto: https` 与 WebSocket 升级头（`Upgrade`/`Connection`）

#### 企业微信登录
- [ ] 系统设置「认证 → 企业微信」已填入凭证并启用，保存后无需重启
- [ ] 系统设置「可信域名校验」已填文件名+内容，`https://your-domain.com:3000/WW_verify_*.txt` 返回录入内容
- [ ] 企微管理后台可信域名已登记为 `your-domain.com:3000`（**带端口**）
- [ ] `.env.local` 中 `COOKIE_SECURE=true`
- [ ] 登录页显示企微扫码入口
- [ ] 扫码后回调 `https://your-domain.com:3000/wecomCallback` 可达
- [ ] 登录成功后 Token 正确写入 Cookie（`Secure` 标志生效）

#### OnlyOffice 文档
- [ ] `https://your-domain.com:3000/office/healthcheck` 返回 `true`
- [ ] `https://your-domain.com:3000/office/web-apps/apps/api/documents/api.js` 可加载（浏览器控制台无 SDK 加载失败）
- [ ] 系统设置 OnlyOffice：ServerUrl = 公网 `/office`、CallbackUrl = `http://backend:8888/api/v1`、TokenSecret 与 docker-compose `JWT_SECRET` 一致
- [ ] 打开 Office 文档能正常加载、编辑、保存（保存后文件回写成功，版本历史更新）

## BUG收集
```

```

### 启动ClaudeCode
```
IS_SANDBOX=1 claude -c --dangerously-skip-permissions
claude --resume 042fe9ee-d7ac-4ba0-89eb-9eccfb5f979f
```
