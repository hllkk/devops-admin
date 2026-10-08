# 服务器模块·资产录入即 SSH 验证（spug 借鉴改造）

> 日期：2026-10-08 ｜ 状态：已实现（go build/vet/test + typecheck/lint 全过，待用户真实目标机验证）

## 需求

用户反馈 slice2 初版交互不好（验证推迟到「安装 agent」按钮），指路 spug 的「新建主机时验证」。调研确认 spug（openspug/spug，`apps/host/views.py` 的 `_do_host_verify`）与本项目架构**完全同构**（密码一次性内存态→注入公钥→密码不落库），其精髓是**保存动作一体完成「密码验证+公钥注入+私钥闭环 ping」**，验证不过不落库。

## 落地设计

- **Asset 加列**：`ssh_username`（spug 模式：用户名是资产属性，存主机行）+ `ssh_verified`
- **保存流**（CreateAsset/UpdateAsset，physical/vm 强制）：有密码=密码连接→注入平台公钥（幂等，复用 `appendAuthorizedKey`）→私钥闭环 ping；无密码（编辑）=仅公钥复验（改字段无需重输密码，spug 同款每次保存复验保证资产仍可管理）；错误分诊对齐 spug E00（不支持密码认证）/E01（不支持公钥）/E02（公钥已注入仍失败=~/.ssh 权限/sshd_config）+密码错+超时
- **验证服务**：`service/server/asset_verify.go`（VerifyAssetSSH，超时 10s 独立于安装 120s）
- **SSH 凭据类型退役**：公钥架构下只剩 username 无共享价值——前端类型选项删 ssh（存量 SSH 凭据行仍可编辑，FORM_FIELDS 保留模板）；资产表单「SSH 凭据下拉」换「SSH 用户名+一次性密码」输入框；`credential_id` 列保留（BMC/DB 凭据 slice5 预留）
- **安装流改纯公钥模式**：`StartInstall(assetId)` 无密码参数——前置校验 `ssh_verified`（未验证提示先编辑资产验证）；连接用 `sshDialPublicKey`+资产行 ssh_username；安装弹窗删表单（只确认+进度）；公钥部署步骤前移到录入验证（安装流步骤精简为连接→探测架构→sftp→systemd→启动）

## 前端

- 表单：physical/vm 显示 SSH 区块（用户名必填+密码「一次性验证使用不保存」，编辑态 placeholder「留空=复验已有公钥」）；提交即验证+保存（含验证会数秒）；提交后前端密码即清空
- 表格：加「SSH验证」列（已验证/未验证 tag，非 physical/vm 显示 -）；安装按钮允许点击未验证资产（后端拦截提示先验证）

## 关键文件

`model/server/server_asset.go`（列）/`model/server/request/server_asset.go`（SshPassword 一次性）/`service/server/asset_verify.go`（验证+分诊）/`service/server/server_asset.go`（保存流集成+needSSHVerify/assetTypeOf/ipOf 辅助）/`service/server/agent_install.go`（公钥化改造）/`web/src/views/_server/asset/modules/asset-list-panel.vue`（表单）/`agent-install-modal.vue`（去表单）

## 相关

- [[server-module-plan]]（slice2 记录：本改造为其修正版）
- [[server-account-grant-plan]]（公钥部署函数「写任意公钥」设计仍在 appendAuthorizedKey）
