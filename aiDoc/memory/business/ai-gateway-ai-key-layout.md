# AI 网关·AI密钥管理页改左右布局（左侧菜单切换）

- 日期：2026-08-26
- 状态：已实现（typecheck/eslint 通过）
- 反向链接：[[ai-gateway-key-scenario]]、[[ai-gateway-overview]]

## 需求

为保证前端展示统一性（对齐供应商管理等网关页面的左右布局），AI 密钥管理页从顶部 NTabs 双页签改为左右展示：左侧"密钥列表/场景管理"两个菜单，右侧展示对应页面内容。

## 设计决策

- **复用 `TableSiderLayout`**（`web/src/components/advanced/table-sider-layout.vue`，与 provider/model/user 等页面同一布局组件），左侧 sider 从"数据列表"换成"菜单"——保证页面骨架与供应商管理一致；菜单选中态样式沿用 provider-item 的 scoped `rgb(var(--primary-color))` 模式。
- **密钥列表抽成面板组件** `modules/ai-key-list-panel.vue`（index.vue 只承担布局+菜单切换），与已有 `key-scenario-panel.vue` 对等；右侧 `v-show` 切换保持两面板状态（等价原 NTabs `display-directive="show"`）。
- **场景→密钥联动刷新**改为 ref 暴露：AiKeyListPanel `defineExpose({ refresh: getData })`，index.vue 监听 KeyScenarioPanel `@changed` 调用。
- 面板根节点统一 `h-full flex-col-stretch overflow-hidden lt-sm:overflow-auto` + 表格卡片 `sm:flex-1-hidden`，在 sider 布局右侧容器里撑满高度。
- i18n 复用既有 `aiKey.tabKeys/tabScenario` 作菜单标题，新增 `tabKeysDesc/tabScenarioDesc` 描述（zh-cn/en-us/app.d.ts 三处同步）。

## 落地

- `web/src/views/_gateway/ai-key/index.vue` 重写：TableSiderLayout + 左侧菜单（lucide:key-round / lucide:shapes）+ 右侧面板切换。
- 新建 `modules/ai-key-list-panel.vue`（迁移原 index.vue 全部密钥列表逻辑）。
- `key-scenario-panel.vue` 根节点补 h-full 撑高；顺带清理存量未用的 `jsonClone` import（eslint 报错）。
