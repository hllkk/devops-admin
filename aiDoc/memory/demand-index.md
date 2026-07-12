# 业务需求索引 (demand-index)

> 仅索引。每条业务需求的日期、标题、文件路径、状态。新增 business 记录时同步追加一行。

| 日期 | 需求 | 文件 | 状态 |
|---|---|---|---|
| 2026-07-11 | 借鉴 gin-vue-admin 实现系统初始化流程（checkdb→initdb，路由守卫自动跳转，后端响应码对齐 "0000"） | business/system-init-flow.md | 已实现 |
| 2026-07-11 | 后端引入雪花算法作为统一主键策略（自实现 + 字符串传输 + GORM Callback 集成） | business/snowflake-id-generator.md | 已实现 |
| 2026-07-12 | 清理基座多租户残留代码（前端 12 文件：类型/service/store/hook/登录与社交登录页面；后端无多租户代码） | business/remove-multi-tenant.md | 已实现 |
| 2026-07-12 | 后端 User/Role 模型建模（贴合前端 RuoYi 契约：SysUser/SysRole + sys_user_role/sys_role_menu + request DTO + AutoMigrate，方案 A，AuditModel 放 system 包） | business/system-user-role-models.md | 设计中 |
