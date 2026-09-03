# devops-admin 离线部署说明

## 包内容

```
├── images.tar.gz          # 全部运行镜像（目标机无需外网）
├── docker-compose.yml     # 生产编排：web + server + updater + pg + redis + rustfs + litellm
├── .env                   # 本次部署机密（构建时生成强随机值，勿外传；APP_VERSION=本次版本）
├── nginx/nginx.conf       # SPA 托管 + /proxy-default 反代 + /oss/ + /llm/ 反代
├── config/                # 后端 config.yaml（含注入的 credential-key）/ PG / Redis / LiteLLM 配置
├── install.sh             # 一键部署脚本
├── upgrade.sh             # 手工升级脚本（在线升级的离线兜底）
└── README-DEPLOY.md       # 本文件
```

> 构建产物另有**增量升级包**（仅自研 web/server/updater 三镜像 + 编排资产，约百 MB 级）
> 与 **manifest-<版本>.json**（版本清单），供发布服务器在线升级用，见下方「在线升级」。

## 部署步骤（目标服务器）

```bash
# 1) 解压（示例目录 /root/devops-admin，可自选）
mkdir -p /root/devops-admin && tar -xzf devops-admin-release-*.tar.gz -C /root/devops-admin
cd /root/devops-admin

# 2) 一键部署
bash install.sh
```

前置要求：Docker 24+ 与 Compose v2（脚本会自检）；数据目录默认 `/home/docker/docker-prod/`，
可用 `DEPLOY_HOME=/data/xxx bash install.sh` 覆盖（仅影响未在 .env 显式配置路径时的默认值）。

## 部署后

| 项 | 值 |
|---|---|
| 访问地址 | `http://<服务器IP>/`（web 容器 80） |
| 初始管理员 | `admin` / `.env` 中 `INIT_ADMIN_PASSWORD`（登录后立即改密） |
| AI 网关接入 | `.env` 中 `LITELLM_PUBLIC_URL`（`http://<IP>/llm`，nginx 同源反代） |
| 后端端口 | 不对宿主暴露，仅经 web 反代 |
| 数据目录 | `/home/docker/docker-prod/{pgsql,redis,rustfs,uploads,server-log}` |

## LiteLLM 端口过渡（4001 → 4000）

目标服务器 4000 已被旧 litellm 占用，本项目 litellm 暂映射 `.env` 的
`LITELLM_HOST_PORT=4001`。客户端接入统一走 nginx `/llm/` 反代（与宿主端口无关），
**切端口不影响任何接入方**。待旧实例停用后：

```bash
# 编辑 .env：LITELLM_HOST_PORT=4000
docker compose -f docker-compose.yml --env-file .env up -d
```

## 升级 / 重部署

三种方式，按场景选：

**A. 在线升级（日常推荐）**：构建机 `build-release.sh` → `publish/publish.sh` 推发布服务器
→ 生产页面「关于」→ 检查更新 → 在线升级（updater sidecar 执行，前端看进度）。
前置：生产 `.env` 配置 `UPDATE_SERVER_URL`（发布服务器地址）与 `UPDATER_TOKEN`。

**B. 手工增量升级（发布服务器不可达时的兜底）**：增量包传到生产安装目录解压
（`tar -xzf devops-admin-upgrade-<版本>.tar.gz -C <安装目录> --strip-components=1`）
→ `bash upgrade.sh`。与 updater install job 同流程：load 版本化镜像 → 保护
config/config.yaml → 替换编排资产 → .env 写新版本 → 重建 web/server/updater。

**C. 全量包重部署**：新构建的全量包解压覆盖后重跑 `bash install.sh`（离线环境首次
升级到带版本化 tag 的版本时用这一次，之后走 A/B）。

## 版本化与回滚

- 自研镜像 tag = 版本号（`devops-admin/web|server|updater:<版本>`，与 server 二进制
  ldflags 注入的 `global.Version` 同源），升级不覆盖旧镜像
- **回滚**：`.env` 的 `APP_VERSION` 改回旧版本号 → `docker compose up -d --no-build`
- 升级只重建自研三容器，pg/redis/rustfs/litellm 不动（数据面与 AI 网关转发面持续可用）
- `config/config.yaml` 的 credential-key 属生产机密（轮换即历史加密凭证不可解），
  升级包不含此文件，updater/upgrade.sh 均做暂存还原保护

## 常用运维

```bash
DC="docker compose -f docker-compose.yml --env-file .env"
$DC ps                      # 状态
$DC logs -f server          # 后端日志
$DC restart server          # 重启后端
$DC down                    # 停止（保留数据）
```

## 机密与配置维护

- 唯一机密源是 `.env`；后端启动时用 env 覆盖 `config.yaml` 敏感项（DB/Redis/JWT/RustFS/LiteLLM master-key）
- `config/config.yaml` 的 `litellm.credential-key`（凭证 AES 加密密钥）由构建时注入随机值；
  **轮换会使历史加密凭证不可解**，非必要不动
- 改 `.env` 后需 `$DC up -d` 让变更生效
