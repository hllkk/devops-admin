# 版本发布流程（UPDATE.MD）

> 面向日常发版的完整操作手册。当前发布服务器 = 构建机本机（172.21.10.41:8100），
> 生产 = 172.21.96.171。本文所有路径/地址均按此环境写，环境变更时同步改这里。

## 发布链路全景

```
构建机(本机)                     发布服务器(本机 8100)              生产(172.21.96.171)
──────────────                  ─────────────────────              ──────────────────
改版本号+提交代码
      │
build-release.sh  ──三件套──▶  /opt/devops-admin-publish/
                                ├─ packages/*.tar.gz(+.sha256)
                                └─ manifest.json(改名后生效)
                                                                     页面「关于」→检查更新
                                                                     →在线升级
                                                                     updater 拉增量包重建
                                                                     web/server/updater 三容器
```

- **在线升级走增量包**（仅自研 3 镜像，约 136MB）；全量包（约 780MB，8 镜像）只用于
  全新部署/离线兜底，日常发版生产不会下载它。两个包 `build-release.sh` 都会产出，
  `publish.sh` 要求三件套齐才发布，属正常现象。
- 升级只重建自研三容器，pg/redis/rustfs/litellm 不动（数据面与 AI 网关转发面持续可用）。
- 自研镜像 tag = 版本号（`devops-admin/web|server|updater:<版本>`），旧版本镜像不覆盖，
  随时可回滚。

## 一次性准备（已就绪，仅环境变更时重做）

1. **发布服务器 nginx**：本机 `/opt/devops-admin-publish/` + `deploy/release/publish/nginx.conf.example`
   的配置（`location /devops-admin/` alias 到该目录），监听 8100。
2. **生产 `.env`**（安装目录 `/root/devops-admin/.env`）：
   ```bash
   UPDATE_SERVER_URL=http://172.21.10.41:8100/devops-admin
   UPDATER_TOKEN=<与 updater 容器一致的 token>   # 写接口鉴权用
   ```
   改完需在生产 `docker compose -f docker-compose.yml --env-file .env up -d` 生效。

## 日常发版五步

### ① 改版本号（两处同步）

| 文件 | 改什么 |
|---|---|
| `server/global/version.go` | `Version = "v1.0.x"`（**全局唯一版本源**，在线升级比对以此为准） |
| `web/package.json` | `"version": "1.0.x"`（展示用） |

### ② 提交代码

发布 tag 会带 git 短 sha（如 `v1.0.5-e8e0281`），**先 commit 再构建**，否则 sha 是旧的。

### ③ 构建

```bash
cd /home/devops-admin/deploy/release
bash build-release.sh          # 可选参数显式指定版本号；默认 <global.Version>-<git短sha>
```

- 产物在 `deploy/release/dist/`：`devops-admin-release-<版本>.tar.gz`（全量）、
  `devops-admin-upgrade-<版本>.tar.gz`（增量）、`manifest-<版本>.json`，均附 `.sha256`。
- 全程约 4~6 分钟（镜像构建 + 800MB tar 导出最慢）。
- 增量包**不含** `.env` 与 `config/config.yaml`（生产 credential-key 是生产机密，
  轮换即历史加密凭证不可解，任何包都不得携带/覆盖）。

### ④ 推送到发布目录

发布服务器就是本机时，`publish.sh` 走 ssh 会因本机 host key 不通而失败，**直接拷贝等效**：

```bash
cd /home/devops-admin/deploy/release/dist
cp devops-admin-{upgrade,release}-<版本>.tar.gz{,.sha256} /opt/devops-admin-publish/packages/
cp manifest-<版本>.json /opt/devops-admin-publish/
```

发布服务器为远端机器时用脚本：

```bash
cd deploy/release/publish
./publish.sh <user@host> [远程发布根目录，默认 /opt/devops-admin-publish] [版本号]
```

### ⑤ manifest 填 changeLog 后生效（关键，不生效生产看不到新版本）

`manifest-<版本>.json` 上传后**不直接生效**（changeLog 必须人工写，打包脚本不知道改了什么）：

```bash
cd /opt/devops-admin-publish
vi manifest-<版本>.json      # 填 changeLog / releaseTime；按需 minUpgradeVersion、forceUpgrade
mv -f manifest-<版本>.json manifest.json   # ⚠️ 必须 -f：root 的 mv 是 -i 别名，不加会卡交互
```

`mv` 原子替换，避免前端检查更新拉到半编辑状态的清单。生效后自检：

```bash
curl -s http://172.21.10.41:8100/devops-admin/manifest.json | head   # 应返回新版本号
```

## 生产升级（三选一）

**A. 在线升级（日常推荐）**：登录生产页面 → 头像「关于」弹窗 → 检查更新 → 在线升级。
前端看进度，updater sidecar 执行：load 版本化镜像 → 保护 config.yaml → 替换编排资产 →
`.env` 写新版本 → 重建三容器。

**B. 手工增量升级（发布服务器不可达时兜底）**：

```bash
scp deploy/release/dist/devops-admin-upgrade-<版本>.tar.gz root@172.21.96.171:/root/devops-admin/
ssh root@172.21.96.171
cd /root/devops-admin && tar -xzf devops-admin-upgrade-<版本>.tar.gz --strip-components=1
bash upgrade.sh
```

**C. 全量包重部署**：新全量包解压覆盖安装目录后 `bash install.sh`（离线环境或大版本变更时用）。

## 验证与回滚

升级后验证：页面「关于」显示新版本号；`docker ps` 三容器 Up；关键功能冒烟。

回滚（生产执行）：

```bash
cd /root/devops-admin
vi .env                              # APP_VERSION 改回旧版本号，如 v1.0.4-2f2e1f9
docker compose -f docker-compose.yml --env-file .env up -d --no-build
```

## 常见坑

| 坑 | 说明 |
|---|---|
| `mv` 卡住不动 | root 的 `mv` 是 `-i` 别名，覆盖必须写 `mv -f` |
| `publish.sh` 本机报 Host key verification failed | 发布服务器=本机时 ssh 自己不通属正常，改直拷（见 ④） |
| 页面检查更新无新版本 | manifest 没改名生效（还是 `manifest-<版本>.json`），或 nginx 缓存；curl 自检确认 |
| 生产拉不到清单 | `.env` 的 `UPDATE_SERVER_URL` 配错或 v1.0.3 及以前版本（env 未进容器，需先升 v1.0.4+） |
| 版本号改了一处 | `version.go` 与 `web/package.json` 必须同步，历史上漏过（v1.0.4） |
| web 构建失败 | 前端 Docker 构建须 node:22-alpine + 国内源，Dockerfile.web 已内置，别改成 node:18/20 |

## 相关文件

| 文件 | 作用 |
|---|---|
| `deploy/release/build-release.sh` | 构建三件套 |
| `deploy/release/publish/publish.sh` | 推送远端发布服务器 |
| `deploy/release/publish/nginx.conf.example` | 发布服务器 nginx 配置示例 |
| `deploy/release/README-DEPLOY.md` | 部署/升级面向目标机的说明（安装兜底） |
| `server/global/version.go` | 版本号唯一源 |
| `deploy/docker-prod/docker-compose.yml` | 生产编排（UPDATE_SERVER_URL/UPDATER_TOKEN env 通道） |
