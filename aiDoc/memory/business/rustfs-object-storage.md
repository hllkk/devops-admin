# RustFS 对象存储集成（drop-in 替换 MinIO）

- 日期：2026-07-23
- 状态：已落地并实跑验证（dev）—— rustfs healthy、bucket `devops-admin` 建好、匿名 `download`、mc 上传+宿主匿名 GET 200 全链路通；仅应用层 minio-go 未单独实跑（mc 同协议已证写入可行）。

## 需求

用户希望用 RustFS 替代 MinIO 作为对象存储后端，docker 集成、开箱即用、不改业务代码即自动启用（用户认为 MinIO 已不维护，实际 MinIO 仍活跃；RustFS 为国产 Rust 实现、Apache 2.0、性能不差，换是合理选择）。

## 选型决策（已与用户确认）

- **OSS 客户端走 `minio`（minio-go）**：drop-in 最自然，直接复用 `server/config.yaml` 的 `minio` 配置块，后端代码零改动。RustFS 是 S3 兼容、官方自称 drop-in MinIO 替换，端口同为 9000；项目只用 PutObject/RemoveObject/StatObject/RemoveObjects/ListObjects 标准 S3 操作，minio-go 连 RustFS 足够。
- **建桶走 docker init 容器**：RustFS 与 MinIO 官方镜像一样不在启动时自动建 bucket，项目代码也无建桶逻辑（`MakeBucket` grep 无结果）。在 compose 加一次性 `minio/mc` sidecar 建桶并设匿名下载策略，后端代码零改。

## 落地改动

1. `deploy/docker-dev/docker-compose.yml`：新增两个服务（对齐用户已验证的 rustfs 配置，**不引入 chown sidecar**）
   - `rustfs`（`rustfs/rustfs:latest`）：`user: root`（免去 bind mount 属主修正），端口 9000(S3)+9001(console)，env `RUSTFS_ROOT_USER/ROOT_PASSWORD/BROWSER=on`（minio 兼容的 root 凭据，比 `RUSTFS_ACCESS_KEY` 更标准），卷挂 `/data`，`command: ["server","/data","--console-address",":9001"]`，healthcheck 探 `curl /health/ready`
   - `rustfs_init`（`minio/mc`）：`depends_on rustfs: service_healthy` 后 `mc mb --ignore-existing` → `mc anonymous set download`
   - 全部参数 `${VAR:-default}` 内联，默认凭据 `rustfsadmin/rustfsadmin`、bucket `devops-admin`、数据卷 `/home/docker/docker-dev/rustfs`
2. `server/config.yaml`：`system.oss-type: local→minio`；`minio` 段占位符换实际值（endpoint `127.0.0.1:9000`、ak/sk `rustfsadmin`、bucket `devops-admin`、bucket-url `http://127.0.0.1:9000/devops-admin`）。注：server 读 `config.yaml`（`core/internal/constant.go ConfigDefaultFile`），`config.docker.yaml` 是 gva 遗留未用模板、不动。
3. `deploy/docker-dev/README.md`：顶部描述/镜像拉取/连接信息表/与 server 对接/.env/数据目录/重置/验证 八处同步。

## 关键约束（踩坑提醒）

- **server 跑宿主机、依赖服务跑 docker**，故 endpoint/bucket-url 用 `127.0.0.1:9000`（端口映射直连）；compose 定义了显式 `dev-net` bridge 网络供服务间互访（如 `rustfs_init`→`rustfs:9000`），未来 server 容器化 join `dev-net` 即可改用服务名。
- 用 `minio` 前必须先 `docker compose up -d` 起 RustFS，否则上传失败；切回本地磁盘改 `oss-type: local`。
- `mc anonymous set download` 让 bucket 匿名可读，前端可直接 GET 文件 URL；**生产应改预签名/鉴权 URL**。
- **RustFS 官方明确"快速迭代中，勿用于生产"**，当前配置面向 dev。
- **勿再为 rustfs 加 chown 10001 sidecar**：官方文档建议镜像以 uid 10001 运行 + chown 数据目录，但 `user: root` 一行即绕过所有属主权限问题（用户实测验证）。初版照搬文档加了 `rustfs_perms`（alpine chown），属过度设计，已移除。

## 关联

- 兑现 [[user-center-profile]] 里"头像刷新显示待切 rustfs/部署层"的待办。
- 对象存储抽象入口 `server/utils/upload/upload.go` `NewOss()`；minio 实现 `server/utils/upload/minio_oss.go`（每次现读 config 建 client，支持热重载）。
