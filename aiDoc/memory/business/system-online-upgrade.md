# 系统·在线升级（发版系统）

- 日期：2026-09-03
- 状态：已实现（build/vet/test/typecheck 通过，待真机升级/回滚演练；swag 重生成欠着——
  同 gateway.MCPServer datatypes.JSON 解析错误阻塞，新接口 @Tags SysUpgrade 注释已按规范书写）
- 关联：SoyDisk 同构组件（/home/SoyDisk/deploy/updater 与 deploy/release），两项目共用发布服务器

## 需求

两个项目（devops-admin、SoyDisk）生产均为 docker compose 离线全量部署，升级原先靠
scp 全量包→解压→install.sh。要在产品内集成版本发布能力：检查更新→前端提示有新版→
在线升级。发布服务器当时未定，地址留配置项。

## 方案拍板

- **对齐 SoyDisk 模式**（评估过 self-update/Watchtower/k8s GitOps/CI-CD 四路线：
  Docker 离线环境只有「发布服务器 manifest + updater sidecar」成立）。updater 是
  SoyDisk 同一套 Go 代码直接搬运（module 名/容器名/镜像名/网络名适配），零第三方依赖。
- **发布服务器地址走 config**：`upgrade.update-server-url` + `upgrade.updater-token`
  （server/config/upgrade.go），生产由 .env 的 `UPDATE_SERVER_URL`/`UPDATER_TOKEN`
  经 initialize/other.go 覆盖。devops-admin 不做 SoyDisk 的系统设置表/定时自动升级/
  维护窗口（范围内裁剪，需要时另立功能点）。
- **权限划分**：`/system/upgrade/version|check|status` 进 casbin_rbac.go 的
  rbacWhitelistPrivate（登录即可，关于弹窗人人可见）；`/system/upgrade/start` 走
  casbin——系统设置菜单 ApiPrefix 追加 `/system/upgrade/start` + F 按钮
  `system:setting:upgrade`（新库生效；已有库超管 SuperAdmin 绕过 casbin 直接可用）。
- **镜像版本化**：web/server/updater tag = `${APP_VERSION:-prod}`，Dockerfile.server
  ldflags 注入 `global.Version/BuildTime`（Version 由 const 改 var 才可注入）；回滚 =
  .env 的 APP_VERSION 改回旧版本 + up -d。

## 关键坑（搬运时必须保持的防线）

1. **credential-key 保护（devops-admin 特有）**：config/config.yaml 的
   credential-key 是构建期注入的 AI 网关凭证 AES 密钥，轮换即历史凭证不可解。
   三道防线：增量包打包排除 config/config.yaml；updater install job 替换资产前
   moveIfPresent 暂存/还原；手工 upgrade.sh 先 mv 备份再合并式拷贝。
2. **安装必须 job 化**：up -d 会重建 updater 自身容器，进程内跑 compose 会被杀中断；
   install job 由 docker daemon 直管能完整写终态（install.go 的 composeBaseArgs 还要
   --project-directory 传宿主真实路径，docker inspect 反查，否则 bind mount 挂空目录）。
3. **升级只动自研三件**（web/server/updater）：pg/redis/rustfs/litellm 不重建——
   数据面与 AI 网关转发面持续可用，也不触发 litellm 建表时序问题。
4. **网络名**：不给 compose 网络加显式 name（会改已部署环境的实际网络名导致全容器
   迁移），updater 的 UPDATER_NETWORK 写推导出的实际名 `devops-admin-prod_prod-net`。

## 产物链

`build-release.sh [版本]`（默认 global.Version-gitsha）→ dist/ 三件套：全量包
（8 镜像+.env+install.sh+upgrade.sh）、增量包（自研 3 镜像+编排资产+upgrade.sh）、
manifest-<版本>.json（+sha256）→ `publish/publish.sh` 推发布服务器（人工填 changelog
后 mv 名 manifest.json 原子生效）→ 生产「关于」检查更新 → updater 下载(断点续传)/
校验/解压/job 安装 → 前端进度弹窗轮询 /system/upgrade/status。
