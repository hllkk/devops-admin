# devops-admin 在线升级执行器（updater）

compose 服务 `updater`（sidecar）：接收 server 升级指令，完成**下载（断点续传）→
sha256 校验 → 解压 → 安装**全流程。零第三方依赖（纯 Go 标准库），docker/compose
操作走 exec CLI。

## 架构：daemon + install 双模式

```
server（业务后端，不碰 docker.sock）
   │ POST /api/upgrade {downloadUrl, sha256, version}
   ▼
updater daemon（常驻）
   ├─ downloading  断点续传下载（进度落状态文件）
   ├─ verifying    sha256 校验（不符即删包拒装）
   ├─ unpacking    解压到 <安装目录>/upgrade-cache/<版本>/（拒绝路径穿越）
   └─ docker run 一次性 install job（用当前 updater 镜像）
        ├─ load 镜像（版本化 tag）
        ├─ 替换编排资产（compose/config/nginx/VERSION，.env 不动；
        │   config/config.yaml 暂存还原，生产现值永不被包内默认版覆盖）
        ├─ .env 写入新版本号（APP_VERSION）
        ├─ compose up -d --force-recreate web server updater
        ├─ 健康检查 http://web/
        └─ success / failed 写状态文件（job --rm 自动清理）
```

**安装阶段为什么 job 化**：`up -d` 会重建 updater 自身容器，进程内直接跑 compose
会被杀中断；job 容器由 docker daemon 直管、与 updater 容器生命周期解耦，能完整
跑完并写终态。新 updater 起来后从状态文件恢复展示。

**容器内跑 compose 的路径陷阱**（实测踩坑）：compose 相对路径 bind mount 以
`--project-directory` 为基准解析成绝对路径传给 daemon——必须传**宿主真实路径**
（经 `docker inspect` 反查本容器挂载源得到），否则挂到宿主不存在的安装目录空目录
（pg 配置挂空文件直接崩溃）。

**升级只动自研三件**（web/server/updater）：pgsql/redis/rustfs/litellm 不重建，
数据面与 AI 网关转发面在升级窗口内持续可用，也不触发 litellm 首启建表时序问题。

## HTTP API（prod-net 内网 `http://updater:8090`）

| 方法 | 路径 | 鉴权 | 说明 |
|---|---|---|---|
| GET | `/health` | 无 | 存活与版本 |
| GET | `/api/status` | 无 | 升级状态机当前态（读状态文件） |
| POST | `/api/upgrade` | Bearer token | 发起升级，活跃态返回 409 |

```jsonc
// POST /api/upgrade 请求体（server 从 manifest 选好包后下发）
{ "downloadUrl": "http://<发布服务器>/packages/devops-admin-upgrade-<版本>.tar.gz",
  "sha256": "<manifest 里的 sha256>", "version": "v0.2.1" }
// 响应 202 {"accepted":true}；进度轮询 /api/status
```

## 状态机（`<安装目录>/upgrade-state.json`，daemon 与 job 共享真源）

`idle → downloading(0-100%) → verifying → unpacking → installing → success | failed`

- 活跃态（downloading/verifying/unpacking/installing）拒绝新升级请求（409）
- failed 后可直接重发；downloading 中断重发**从断点续传**
- 卡在 installing（job 异常死亡）→ `docker rm -f devops-upgrade-job` 后重发；
  排查看 `<安装目录>/upgrade-install.log`

## 环境变量（compose 注入）

| 变量 | 默认 | 说明 |
|---|---|---|
| `STACK_DIR` | `/opt/stack` | 安装目录挂载点（compose 挂 `./`） |
| `UPDATER_TOKEN` | 空 | 写接口鉴权；server 侧 `.env` 的 `UPDATER_TOKEN` 填同值；空=不鉴权（仅限可信内网） |
| `UPDATER_IMAGE` | `devops-admin/updater:prod` | install job 复用的镜像（版本化 tag） |
| `UPDATER_NETWORK` | `devops-admin-prod_prod-net` | job 接入的网络（= 项目名 + `_` + 网络键，健康检查 web 用） |
| `UPDATER_LISTEN` | `:8090` | 监听地址（无 ports 映射，仅内网） |

## 本地开发

```bash
cd deploy/updater
go vet ./... && go test ./...   # 单测：sha256/tar 穿越/env 替换/状态机
go run . version
```

## 排查

| 现象 | 处理 |
|---|---|
| 状态卡 installing | `docker ps -a \| grep devops-upgrade-job` 看 job；`cat upgrade-install.log`（安装目录下） |
| 下载失败 | 重发升级请求（断点续传）；检查发布服务器连通 |
| sha256 不符 | 损坏包已自动删除，重发即可；持续不符查发布服务器上的包 |
| 升级后服务异常 | 回滚：`sed -i 's\|^APP_VERSION=.*\|APP_VERSION=<旧版本>\|' .env && docker compose up -d --no-build` |
