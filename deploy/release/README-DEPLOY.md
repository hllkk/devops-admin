# devops-admin 离线部署说明

## 包内容

```
├── images.tar.gz          # 全部运行镜像（目标机无需外网）
├── docker-compose.yml     # 生产编排：web + server + pg + redis + rustfs + litellm
├── .env                   # 本次部署机密（构建时生成强随机值，勿外传）
├── nginx/nginx.conf       # SPA 托管 + /proxy-default 反代 + /oss/ + /llm/ 反代
├── config/                # 后端 config.yaml（含注入的 credential-key）/ PG / Redis / LiteLLM 配置
├── install.sh             # 一键部署脚本
└── README-DEPLOY.md       # 本文件
```

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

用新构建的部署包解压覆盖后重跑 `bash install.sh`：
`docker load` 覆盖同名镜像 → `up -d` 仅重建有变化的容器，数据在 bind mount 目录不受影响。

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
