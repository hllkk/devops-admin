# 业务模块 (business-modules)

> devops-admin 的业务按模块组织，每个模块一节，记录 model/接口/边界。随业务开发补充。
>
> **实现状态总览（2026-07-16 校准）**
> - **当前已落地主线：system 核心基座（`1d632d9` 重构后）**——用户/角色/部门/岗位模型 + 安全配置/数据权限/JWT 黑名单/错误日志/定时任务 Service 骨架（详见下方「系统模块（重构后现状）」节）。**注意**：重构前的菜单/字典模型、初始化向导、`/init/*`、`/auth/*`（httpOnly cookie + go-captcha）、`sys_setting` 等实现**在重构后未保留**，对应 API/Router 待重建（历史轨迹见 `aiDoc/memory/business/` 与 `demand-index.md`）；`api/` 层当前为空。
> - **网盘模块：规划中、尚未启动**（见下方「网盘模块（规划中，未启动）」节）。前端 i18n 已预留 `disk` 等模块文案（`src/locales` 的 `module` 命名空间），后端无对应代码、`demand-index.md` 无相关记录。原需求骨架 `SoyDisk-Product-Spec.md` **当前不在仓库内**，启动前需重新确认去向。

## 网盘模块（规划中，未启动）

> 下方为企业内部网盘的**规划蓝图**，仅作设计参考，**尚未实现**。后端无 `server/` 对应代码，`demand-index.md` 无相关业务记录；启动前需先找回或重建 `SoyDisk-Product-Spec.md`。

企业内部网盘：管理员上传大型安装包/交付物，对外用「链接 + 提取码」匿名分享给客户下载，对内按账号/部门共享并支持 OnlyOffice 多人协同编辑；含版本历史、回收站、配额管控、企业微信扫码登录。存储用 RustFS，协同用 OnlyOffice。

### 文件管理
- 上传：分片 + 断点续传 + hash 秒传，单文件上限 5GB，落 RustFS
- 文件/文件夹 CRUD：新建/重命名/移动/删除（进回收站）
- 预览：txt/pdf/Office（OnlyOffice 渲染），统一新页面打开

### 对外分享
- 链接 + 提取码 + 有效期（永久/7天/1天/自定义）+ 可选下载次数限制
- 外部客户匿名下载，不算系统用户，权限档为"可下载"

### 内部共享
- 按账号/部门授权，无需提取码
- 权限档：只读 / 编辑（编辑 = 文件操作层 rename/move/delete/overwrite + OnlyOffice 内容层协同）

### 配额
- User 默认 100M（上传门槛）；管理员可调；公司总量上限 2T（兜底）
- 按文件实际大小计，**不含版本历史占用**；超额上传拦截并提示

### 版本历史
- 覆盖上传 → 旧版进历史；默认留 10 份；可一键回滚

### 回收站
- 删除进回收站；可恢复；过期清理（`server/task/`）

### OnlyOffice 协同
- 有编辑权限者多人实时编辑（光标/改动同步）；自动保存 + 版本记录
- 权限与文件操作层共用一个"编辑权限"开关

### 企业微信登录
- 扫码登录，无需独立账号密码；部门/员工组织架构自动同步
- 离职账号随企微停用，其共享权限同步失效

### 角色与配额管理
- 三级角色：超管 / 管理员 / User
- **角色·用户·部门管理复用 SoybeanAdmin 基座，不重复实现**
- 配额在基座用户管理中调整；总量上限在系统设置-网盘配置

### 公共资料库
- 一键覆盖全员可见可下载，管理员不必逐人共享

## 系统模块（重构后现状）

`1d632d9`「完成项目基础架构重构与模块初始化」起，系统模块以**核心基座**形式重建，命名统一 `OPS_` 前缀。当前为 **Model + 部分 Service 骨架**，业务 API/Router 尚未补齐。

### 已建模型（`model/system/`）

- 业务实体：`SysUser`（`OPS_AUDIT_MODEL` + 自定义 `UserId`，字段对齐前端 `Api.System.User`/RuoYi）、`SysRole`（旧形态：手写时间戳、主键 `RoleId` json `id`）、`SysDepartment`、`SysPosition`。
- 系统表：`JwtBlacklist`、`SysSecurityConfig`（安全配置，替代旧 `sys_setting`）、`SysError`（错误日志）、`SysDataAccessLog`（数据访问审计）、`SysTimedTask`/`SysTimedTaskLog`（定时任务）。
- 关联表（显式 struct）：`SysUserRole`、`SysRoleDepartment`、`SysUserDepartment`。
- **未保留**：`SysMenu`/`SysDictType`/`SysDictData`/`SysRoleMenu`（重构前曾建，重构后待重建）。

### 已建 Service 骨架（`service/system/`）

- `data_scope`：数据权限引擎（按 `dept_id`/`create_by` 构建身份与可见范围，实现见 `utils/datascope`）。
- `jwt_black_list`：JWT 黑名单（refresh 轮换失效）。
- `sys_security_config`：安全配置（密码策略/登录失败锁定/IP 黑白名单，存 `SysSecurityConfig` 表）。
- `sys_error`：错误日志。
- `sys_timed_task`（+ `http` + `runner`）：定时任务调度与 HTTP 触发。
- `auto_code`：代码生成。

### 待补

- 业务 API/Router：`api/` 当前**为空**，user/role/dept/post/menu/dict 的 API+Router 待按 `Api.System.*` 反推实现（前端页面已齐备）。
- 菜单/字典：模型与权限 seed 待重建（前端 `page.system.menu/dict` 已就绪）。
- 认证与初始化链路：`/auth/*`（httpOnly cookie 登录 + go-captcha 验证码）、`/init/*`（初始化向导）等 HTTP 接口重构后未保留，待重建（历史设计见 `aiDoc/memory/business/` 的 `httponly-cookie-auth`、`go-captcha-login`、`system-init-flow`、`init-wizard-redis`）。

### 系统设置（待重建）

重构前的全局配置中心 `sys_setting`（`name`=分类 + `value`=JSON，六类 `general`/`security`/`authentication`/`ldap`/`notify`/`disk`）**在重构后未保留**。当前安全相关配置由 `SysSecurityConfig` 承载，其余分类（通用/认证/LDAP/通知/网盘）的落地方式待重新设计，历史设计要点见 `demand-index.md`。

