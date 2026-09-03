# AI 网关·密钥归属对象用户选择支持搜索

- 日期：2026-09-02
- 状态：已实现（typecheck 通过）
- 反向链接：[[ai-gateway-ai-key-layout]]

## 需求

密钥管理新增密钥时，「归属对象」选用户类型后的用户下拉不支持搜索，用户多时只能滚动找。

## 设计决策

- **在公共组件 `UserSelect` 内默认 `filterable`，而不是逐使用点手动传**：该组件语义就是"从全量用户里挑一个"，可搜索是普适期望；现状是 `budget-rule-drawer.vue` 已手动传 `filterable`、其余 7 处（密钥新增抽屉/密钥搜索区/用量·成本·审批搜索/MCP 调用日志）都没传，说明是普遍遗漏而非个别刻意关闭。
- 组件保持 `inheritAttrs: false` + `filterable` 放在 `v-bind="$attrs"` **之前**，父级仍可用 attrs 覆盖（如 `:filterable="false"`）。
- 数据侧维持全量 `/system/user/optionselect` + 前端过滤，不引入远程搜索——内网运维平台用户量级不需要，符合"不做过度设计"。

## 落地

- `web/src/components/custom/user-select.vue`：`NSelect` 加 `filterable`（一处改动，全部使用点受益）。
- `web/src/views/_gateway/gateway/modules/budget-rule-drawer.vue`：删掉 UserSelect 上冗余的手动 `filterable`。
