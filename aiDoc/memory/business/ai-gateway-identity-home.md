# AI 网关·个人中心首页（home/index.vue，前端展示）

> 2026-07-29。参照本地 `/home/remote/AIHelms` 的 `MyIdentityView.vue`，在 devops-admin 用 Soybean 规范复刻「AI 个人中心」展示页，落位 `views/home/index.vue`（首页）。AI 网关后端尚未实现，AI 相关数据（API Key / 模型·MCP·Skill 授权 / 用量 KPI / 趋势 / 申请）用前端 mock 跑通展示；用户身份信息（昵称/邮箱/部门）取真实 auth store。这是 AI 网关整体功能的第一步——先做前端展示。

## 现状核对

- AIHelms 参照路径 `/home/remote/AIHelms/ui/packages/web/src/views/MyIdentityView.vue`（FastAPI+Vue3 蓝本，devops-admin 后端 Go 只能参照设计）
- devops-admin 的 AI 网关（`views/_gateway` / `service/api/gateway`）当前为占位页，后端 `/gateway/*` 接口完全不存在 → 前端走 mock（参照 disk `service/api/disk/file.ts` 的 `USE_MOCK` 硬编码模式）
- 复用项确认：echarts 6 + `useEcharts` hook（`hooks/common/echarts.ts`，`line-chart.vue` 样例）→ 不装 vue-echarts；图标走 Iconify `lucide:*` 经 `SvgIcon` → 不引 lucide-vue-next；复制用 `@vueuse/core` `useClipboard`
- 用户信息字段：`useAuthStore().userInfo.user`（`Api.System.User`：nickName / email / deptName / avatar）

## 展示区块（与 MyIdentityView 对齐）

1. 加载骨架屏（isLoading）
2. AI 身份证卡片：logo+标题+状态徽章、用户信息、API Key（显隐/复制）、meta（预算/模型/MCP/Skill 计数）
3. 我的资源：模型(NTag primary) / MCP(success) / Skill(warning)
4. 用量概览：月预算 / 已花费 / 请求数 / 日均 + 预算进度条 + 趋势折线图(useEcharts)
5. 我的申请：pending / approved / rejected 状态列表

## 改动（4 文件）

- `web/src/views/home/index.vue`：占位页 → **全屏门户页**（`min-h-screen` 渐变背景 + 自定义 sticky header，**不使用 global-header**——home 走 `layout.blank` 本就无全局 header/sider/tab）；header 三段式（左 `SystemLogo`+「我的主页」/ 中三导航「我的AI身份·AI市场·模型广场」绝对水平居中，后两者点 `comingSoon` 占位提示 / 右 `NAvatar`+昵称+`NDropdown` 个人中心·退出登录）；内容区 `max-w-5xl` 居中；mock 数据内联
- `web/src/locales/langs/{zh-cn,en-us}.ts`：新增 `page.home.identity.*`（41 key：身份/资源/用量/申请 + 导航 navIdentity·navMarket·navModels + comingSoon，中/英）；顺补 `route.home`（我的主页/Profile，兼作 header 左侧标题）——此为 home 新增路由遗留的 i18n 缺失，typecheck 暴露
- `web/src/typings/app.d.ts`：home schema 补 `identity` 子树

## 关键决策

- **数据策略**：AI 数据 mock 内联组件（不建 `service/api/gateway/`）——后端契约未定，建 service 层是空架子；真正做 AI 网关后端时再按 `frontend-rules §9.10` 建 gateway 四层并切换真实接口。home 是基座首页非业务模块，内联 mock 最轻量。
- **颜色**：AI 身份证卡保留品牌紫渐变（#7C3AED→#5B21B6）+ dark 适配，作为该区块固定品牌标识（呼应 AIHelms「ACCESS PASS」视觉）；其余区块用 UnoCSS 语义色（slate/emerald/amber/red）+ NTag 内置色 + dark: 变体。规范 §4 禁硬编码颜色——身份证品牌色作为 logo 级固定色保留，如需跟随主题色可改。
- **趋势图配色**：echarts option 内固定紫 #8b5cf6（与身份证品牌色同系），dark 下轴线未做特殊处理（与 `line-chart.vue` 现状一致，不过度设计）。
- **未显示头像大图**：MyIdentityView 有 avatar 显示大图、无则紫色块；devops-admin 固定用品牌渐变块 +「AI」水印，避免 avatar 相对 URL 在 local 模式刷新 404 的既有问题（见 [[user-center-profile]]「已知点」）。

## 验证

`pnpm typecheck`（vue-tsc）+ `pnpm lint`（oxlint + eslint --fix）+ `pnpm fmt`（oxfmt）全通过。未跑 build / dev 目测。

## 待办（AI 网关整体）

- 后端 `/gateway/identity/*`（个人 AI Key、授权模型/MCP/Skill、用量 KPI、趋势、申请记录）
- 后端就绪后：删 home mock，按 §9.10 建 `service/api/gateway/identity.ts` + `typings/api/gateway.api.d.ts`，home 调真实接口
- AI 网关其它管理页（模型/MCP/Skill 市场、Key 管理、申请审批）落 `views/_gateway/`

## 关联

- 参照蓝本：`/home/remote/AIHelms/ui/packages/web/src/views/MyIdentityView.vue`
- mock 模式参照：disk `service/api/disk/file.ts` 的 `USE_MOCK`
- 个人中心（用户资料/头像后端）：[[user-center-profile]]
