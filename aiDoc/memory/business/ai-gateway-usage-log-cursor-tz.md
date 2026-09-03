# AI 网关·用量回流游标时区错位修复

- 日期：2026-09-02
- 状态：已实现并运行时验证（游标推进到此前被卡的那笔调用）
- 关联：[[ai-gateway-mcp-usage-sync]]（MCP 回流）、[[ai-gateway-usage-log-page]]（调用日志页）、[[ai-gateway-implementation-notes]]

## 现象

用户经平台 AiKey 调 LLM 后：调用日志页打开/刷新无数据（查询驱动回流失效）、「立即回流」无数据、只有「漏单对账」能展示——增量路径整体盲区，对账兜底成为唯一通路。

## 根因

`LiteLLM_SpendLogs.startTime/endTime` 是 **naive UTC**（timestamp without time zone）；游标参数是 Go `time.Time`（timestamptz）。PG 比较 naive 列 vs timestamptz 参数时，把参数按**连接会话时区**渲染成 naive 再比较：

- spend-dsn 漏配 `TimeZone=UTC` → 会话时区落到 PG 服务器默认 `Asia/Shanghai` → 同一时刻两侧解释差 8h → **新调用 8 小时内对增量回流不可见**（游标值超前"假装"已同步过）。
- 读取侧（pgx 扫 naive 列）不受会话时区影响，按 UTC 解释——所以对账回灌进来的行时间显示是对的，隐蔽性强。
- 库上实测复现：同一行同一游标，会话 +08 → `visible=f`；会话 UTC → `visible=t`。
- 对账走 `request_id` 差集不看游标，所以能捞到——与用户观察完全吻合。

模型注释（llm_log.go）本约定"连接串配 TimeZone=UTC"，但 dev config 没配，正确性依赖配置纪律，一漏即复发。

## 修复（4 处 SQL + 1 处配置）

- `usage_sync.go` fetchSpendBatch：游标比较两分支改 `COALESCE("endTime","startTime") AT TIME ZONE 'UTC' > ?`（复合游标行同步）；ReconcileLLMLogs 窗口 `"startTime" AT TIME ZONE 'UTC' >= ?`。
- `mcp_usage_sync.go` fetchMcpSpendBatch + ReconcileMcpLogs：同款四处中的两处。
- dev `config.yaml` spend-dsn 补 `?TimeZone=UTC` 双保险（代码侧已不依赖它，但保持注释约定一致）。
- prod 共享库场景（spend-dsn 留空复用主库连接）：同样被 `AT TIME ZONE 'UTC'` 显式化覆盖，主库连接会话时区亦不再参与正确性。

## 验证

重启 dev server 后等 `*/5` 定时 tick：游标从 `12:10+08`（启动兜底值）推进到 `17:22:37.372+08 + 43102970-...`——正是此前被 8h 盲区卡住的那笔平台 Key 调用；`gateway_llm_log` 从对账回灌的 1 行不变（幂等 DO NOTHING），游标追平后查询驱动回流恢复正常（打开页面即可见）。

## 经验

- naive timestamp 列与 timestamptz 参数比较，PG 永远按会话时区折算参数——跨库回流/同步凡涉及 naive 列比较，一律显式 `AT TIME ZONE`，别依赖连接串时区配置纪律。
- "读取正确 + 比较错误"的组合最隐蔽：数据时间显示全对，仅增量窗口判断错位，症状是"新增不进、对账才进"。
