# 企业微信通讯录同步（部门/用户/岗位）

> 2026-09-01 提出，同日完成三方分析（SoyDisk / 远程版 devops-admin / 当前项目现状）并经用户拍板方向。承接 [[wecom-qrcode-login]] 遗留的"不含组织架构同步（后续再做）"，身份侧与其共用 `sys_social(source=wecom)`。

## 需求

从企业微信通讯录单向拉取（只拉不推），同步部门、用户、岗位到本地：

- 部门 → `sys_departments`，用户 → `sys_users` + 多部门/岗位关联，岗位由企微 `position` 派生
- 手动触发（部门管理页按钮，异步+轮询）+ 定时任务（默认禁用）

## 三方分析与方向（已拍板）

演化链：远程版 `/home/remote/devops-admin`（原始，坑多）→ SoyDisk（移植+修复一轮，残留 4 坑）→ 当前项目（空白但企微基建三者最好）。**方向：以 SoyDisk `server/service/system/sys_wecom_contact.go`（728 行）为蓝本整体移植，用当前项目更强基建替换薄弱环节，顺手修掉其残留坑**。分析详情见 2026-09-01 会话报告（三方对比表/资产清单/坑清单）。

## 关键决策

- 身份映射：**复用 `sys_social`**（不另起映射表），加 `in_sync_scope` 列做"在册标记"，划清离职停用作用域（SoyDisk 修过"误伤扫码用户"事故的正解）
- 部门映射：`sys_departments` 加 `wecom_dept_id int64`（0=手动建），AutoMigrate 自动加列
- 部门 diff：SoyDisk 两阶段（先建齐缺失+ancestors 占位 → 全部本地 ID 就绪后带 memo 递归重算 ancestors），不依赖企微返回顺序
- 用户 diff：批量 load 内存 map 比对（防 N+1）+ 每用户小事务 + 字段级防覆盖（空值/gender"0" 不回写，保护扫码登录写入的资料）
- 岗位：`(主部门, position)` 派生 `post_code="wecom_<deptId>_<hash>"`，`wecom_` 前缀来源隔离，`sys_user_post` 关联全量重建，手动岗位不受影响
- 离职：差集停用（在册绑定不在企微返回集 → `status='1'`，不删不解绑）；部门/岗位企微侧删除本地保留
- 建号：随机不可用口令 + `PasswordUpdatedAt=now` + `SysGeneralConfig.DefaultRoleId`（复用现有，不设企微专属配置）
- 触发：手动走异步 goroutine + 前端 3s 轮询（SoyDisk 模式）；定时走 `task.Register("SyncWecomContact")` + `sys_timed_tasks` 种子默认 `@daily` 禁用（防重叠/panic 兜底/执行日志/失败 SSE 告警全白拿）
- token：复用当前 `WecomClient`，补 `DepartmentList`/`DepartmentUsers` 走 `wecomGet`——40014/42001 失效重试自动获得（SoyDisk 恰好缺）
- 权限：PrivateGroup + 菜单 F 按钮 `system:wecom:sync`（ApiPrefix 沿用菜单模式；已有库需补丁 SQL）
- **相对 SoyDisk 的 4 项修复**：①重新入职自动复启（在返回集且停用 → 恢复 '0'）②同步状态快照落 Redis（多实例可见）③fail-fast 错误带部门/用户定位 ④绑定孤儿清理（本地硬删用户残留的 sys_social）
- 明确不做（过度设计）：企微回调增量、统一 IDP 抽象、`user/list_id` 游标替换、独立同步历史表

## 落点（实际）

- 模型：`sys_departments` 加 `wecom_dept_id`、`sys_social` 加 `in_sync_scope` + `disabled_by_sync`（复职恢复的作用域标记）、新增 `model/system/sys_wecom_contact.go`（WecomSyncResult 含 UserRestored / WecomSyncStatus）
- 后端：`server/utils/wecom.go` 补 `WecomDepartment`/`WecomContactUser` DTO + `DepartmentList`/`DepartmentUsers`（带 40014/42001 失效重试）；`server/service/system/sys_wecom_contact.go`（核心，StartSync 异步手动入口 + SyncStructure 定时入口 + syncStructureLocked 主流程）；`api/v1/system/sys_wecom_contact.go` + `router/system/sys_wecom_contact.go`（挂 PrivateGroup）；三处 enter 注册；`initialize/router.go` 挂载
- 定时：`initialize/timer.go` 注册 `SyncWecomContact`；`source/system/timed_task.go` 种子（@daily 默认禁用，仅新库生效）
- 菜单：`source/system/sys_menu.go` 部门管理下 F 按钮 `system:wecom:sync`，ApiPrefix 显式枚举 `/system/wecom/syncStructure, /system/wecom/syncStatus`（super/admin 种子自动全授权）；已有库补丁 `deploy/patches/2026-09-02-wecom-contact-sync-menu.sql`
- 前端：`service/api/system/wecom.ts`、`system.api.d.ts` 两类型、`dept/index.vue` 同步按钮（hasAuth）+3s 轮询+结果 toast（i18n 插值 `syncDone` 带 7 项统计）、i18n 三处（zh-cn/en-us/app.d.ts，6 key）
- 单测：`service/system/sys_wecom_contact_test.go`（性别转换/postCodeHex/集合比较/部门映射 4 纯函数）

## 状态

已实现（2026-09-02，go build/vet/test 全过 + vue-tsc/eslint 通过），待真实企微联调。
真实联调前置：企微自建应用配通讯录读取权限（secret 需含部门/成员可见范围），否则 department/list 报 60011/48002 权限错误。
