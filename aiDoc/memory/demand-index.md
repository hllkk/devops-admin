# 业务需求索引 (demand-index)

> 仅索引。每条业务需求的日期、标题、文件路径、状态。新增 business 记录时同步追加一行。

| 日期 | 需求 | 文件 | 状态 |
|---|---|---|---|
| 2026-07-11 | 借鉴 gin-vue-admin 实现系统初始化流程（checkdb→initdb，路由守卫自动跳转，后端响应码对齐 "0000"） | business/system-init-flow.md | 已实现 |
| 2026-07-11 | 后端引入雪花算法作为统一主键策略（自实现 + 字符串传输 + GORM Callback 集成） | business/snowflake-id-generator.md | 已实现 |
