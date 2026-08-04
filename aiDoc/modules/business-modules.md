# 业务模块 (business-modules)

> devops-admin 的业务按模块组织，每个模块一节，记录 model/接口/边界。随业务开发补充。
>
> **实现状态总览（2026-07-16 校准）**
> - **当前已落地主线：system 核心基座（`1d632d9` 重构后）**——用户/角色/部门/岗位模型 + 安全配置/数据权限/JWT 黑名单/错误日志/定时任务 Service 骨架（详见下方「系统模块（重构后现状）」节）。**注意**：重构前的菜单/字典模型、初始化向导、`/init/*`、`/auth/*`（httpOnly cookie + go-captcha）、`sys_setting` 等实现**在重构后未保留**，对应 API/Router 待重建（历史轨迹见 `aiDoc/memory/business/` 与 `demand-index.md`）；`api/` 层当前为空。

## 系统模块（重构后现状）

`1d632d9`「完成项目基础架构重构与模块初始化」起，系统模块以**核心基座**形式重建，命名统一 `OPS_` 前缀。当前为 **Model + 部分 Service 骨架**，业务 API/Router 尚未补齐。

### 已建模型（`model/system/`）

- 业务实体：`SysUser`（`OPS_AUDIT_MODEL` + 自定义 `UserId`，字段对齐前端 `Api.System.User`/RuoYi）、`SysRole`（旧形态：手写时间戳、主键 `RoleId` json `id`）、`SysDepartment`、`SysPosition`。
- 系统表：`JwtBlacklist`、`SysSecurityConfig`（安全配置，替代旧 `sys_setting`）、`SysError`（错误日志）、`SysDataAccessLog`（数据访问审计）、`SysTimedTask`/`SysTimedTaskLog`（定时任务）、`SysLoginLog`（登录日志）、`SysOperLog`（操作日志）。
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

重构前的全局配置中心 `sys_setting`（`name`=分类 + `value`=JSON，五类 `general`/`security`/`authentication`/`ldap`/`notify`）**在重构后未保留**。当前安全相关配置由 `SysSecurityConfig` 承载，其余分类（通用/认证/LDAP/通知）的落地方式待重新设计，历史设计要点见 `demand-index.md`。

