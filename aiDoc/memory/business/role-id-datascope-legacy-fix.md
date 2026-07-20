# 角色ID遗留清理 + data_scope 默认权限收紧

- 日期：2026-07-20
- 状态：已实现（build 通过）
- 触发任务：通知公告 SSE 实时推送（补 timed_task 告警出口）

## 背景

补定时任务告警 SSE 出口时，发现 `sys_timed_task_runner.go` 的 `alertRoleID = 888` 是 GVA 搬迁遗留：devops-admin 角色主键已改雪花 int64（见 [[snowflake-id-generator]]、[[authority-to-role-rename]]），不存在 888 角色，`alertFailure` 查 `sys_role_id=888` 恒空、告警永远无接收人。顺藤摸出两处同类遗留 + 一个更实质的数据权限越权隐患。

## 修复（三处）

1. **告警接收人**（`service/system/sys_timed_task_runner.go`）：删 `alertRoleID=888` 常量，`alertFailure` 改子查询超管角色（`SysRole.SuperAdmin=true`）用户 → `sys_user_role` 取 userIDs；payload 加 `type` 字段、`Event.Name` 留空走 message 通道（前端 `useEventSource(url,[])` 只监听 message，统一经 `/resource/sse` 送达，按 `payload.type` 分流通知/告警）。
2. **用户主角色默认值**（`model/system/sys_user.go`）：`RoleId gorm:"default:888"` → `default:0`（0=未指定）。Create 已在 `sys_user_manage.go:105` 取 `roleIds[0]` 回填，888/0 仅在"未选角色"时 DB 兜底。
3. **data_scope 默认权限**（`service/system/data_scope.go`）：`BuildIdentity` 查不到角色（RoleId 无效）或角色 `data_scope=0` 时，else 分支由 `ScopeAll`（全部）→ `ScopeSelf`（仅本人）——堵住"无效主角色 → 全量数据权限"越权。超管不受影响（正常 RoleId=超管角色走 if 分支拿 `DataScope=1=ScopeAll`，不进 else）。

## ⚠️ 部署副作用（重要）

data_scope 改动是**收紧权限**：上线后 `role_id` 异常（0/历史 888/已删角色）的用户，数据权限从"全部"变"仅本人"。部署前排查并修正：

```sql
SELECT id, user_name, role_id FROM sys_users
WHERE role_id NOT IN (SELECT role_id FROM sys_role) OR role_id = 0;
```

老库 `sys_users.role_id` 的 DB default 仍是 888（AutoMigrate 不改 default 约束），需 `ALTER TABLE sys_users ALTER COLUMN role_id SET DEFAULT 0` 或 `/initdb` 重建才生效；新库直接是 0。

## 关联

- [[snowflake-id-generator]]：角色主键改雪花 int64 是 888 失效的根因
- [[authority-to-role-rename]]：Authority→Role 统一，RoleId 体系定型
- [[notice-management]]：告警链路经通知 SSE 端点 `/resource/sse` 送达超管
