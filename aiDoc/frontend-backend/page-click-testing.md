# 页面点触测试（AI 驱动浏览器验证）

> 适用范围：前端页面改动后，需要真实浏览器点触验证交互/视觉/数据流时，AI 用浏览器自动化驱动验证。
> 配套高层规则见 `AGENT.MD` 的「页面点触测试」一节；登录态机制见 `aiDoc/memory/business/httponly-cookie-auth.md`、验证码见 `aiDoc/memory/business/go-captcha-login.md`。

## 0. 何时用、何时不用

- **用**：改动涉及路由/菜单/布局/表单交互/权限显隐/表格联动等运行时行为，单靠代码阅读和类型检查无法确认效果时。
- **不用**：纯样式原子类调整、文案 i18n、可由 `pnpm typecheck` + 代码审读覆盖的改动——不要为小改一动就起浏览器。

## 1. 环境前置（浏览器自动化能力）

- **首选 Playwright**（`@playwright/test` 或 `playwright`）。备选仅在与现有工具链冲突时再考虑。
- 启动前先探测当前环境是否具备浏览器自动化能力（如 `npx playwright --version` 或尝试拉起一个 headless context）。
- **不具备时，主动建议用户安装**，标准话术：

  > 这次改动建议用真实浏览器点触验证。当前环境没有浏览器自动化能力，推荐装 Playwright（`pnpm dlx playwright install chromium`，约需下载浏览器内核）。是否安装？装好我再继续。

- **经用户确认后再装**，不要不问就装；安装走项目既有包管理器（pnpm）。
- **用户拒绝安装** → 回退为「给出人工目测清单」（见 §5），不要硬撑。

## 2. 登录态获取（httpOnly cookie 模式，关键）

本项目登录态是 **httpOnly cookie 模式**：access/refresh token 由后端写入 cookie，前端 JS **拿不到也不下发** Authorization，本地只存 `isAuthenticated`（bool）与 `tokenExpiresAt`（见 `auth/shared.ts`、`service/request/shared.ts`）。

> ⚠️ **禁止**走"向用户索取 token 字符串、注入 localStorage"的老办法——token 不在 JS 可见范围，这条路在本项目不成立。

维持登录态，按优先级三选一：

1. **`storageState` 复用（推荐，无需知道 cookie 名）**
   - 首次：在一个 Playwright context 里走真实登录（或让用户在弹出的浏览器里登录一次），保存：
     ```ts
     await context.storageState({ path: '.local/auth-state.json' });
     ```
   - 之后：复用，跳过登录：
     ```ts
     const context = await browser.newContext({ storageState: '.local/auth-state.json' });
     ```
   - `storageState` 同时保存 cookie（含 httpOnly）与 localStorage，正好覆盖 `isAuthenticated`/`tokenExpiresAt`。

2. **`context.addCookies` 注入**：从后端/运维拿到后端下发的 cookie 名与值，直接注入 context。仅当无法走完整登录、且能拿到 cookie 原始值时使用。

3. **真实登录流程**：在自动化浏览器里输入用户名+密码+验证码完成登录。受验证码影响，适合一次性验证、或配合 §3 的阈值下调。

凭证来源话术（向用户索取 `storageState` 或 cookie）：

> 我需要登录态来点触验证。能否请你在一个真实浏览器里登录系统，然后用 Playwright 把登录态导出到 `.local/auth-state.json` 给我（或提供后端下发的 cookie）？这是真实凭证，我只放在已 gitignore 的 `.local/` 下，不会进提交/截图/日志。

## 3. 覆盖登录链路本身（验证码 / 锁定）

- 登录链路用 go-captcha 行为验证码（见 `go-captcha-login.md`），自动化里直接过验证码成本高。
- 需要覆盖登录/锁定/验证码相关用例时：在「系统设置 → 安全配置」把**验证码阈值临时下调**（或临时关闭），用测试账号直登完成验证，**测完立即改回原值**。
- 改前记下原值，改后核对还原，不要把临时值留在配置里。

## 4. 凭证安全约束

- cookie / `storageState` / 账号密码都是**真实凭证**：
  - 缓存只能落 `.local/`（根 `.gitignore` 已忽略整个目录）；
  - **不写入任何会提交的文件**（代码、文档、示例、配置）；
  - **不出现在截图、控制台日志、commit 信息、PR 描述里**；
  - 截图前确认页面无敏感信息（顶栏用户名、token、真实业务数据按需打码）。
- 点触可能造成**破坏性数据操作**（删除/批量操作/状态翻转）时，**先征得用户同意**再执行，优先用可见的测试数据。

## 5. 人工目测回退清单（无法起浏览器时）

当用户拒绝安装自动化、或环境确实无法拉起浏览器，给出**结构化目测清单**，例如：

```
请按以下顺序在浏览器里核对本次改动：
1. 进入「系统管理 → <模块>」列表页，确认表格渲染、分页、列顺序与改动一致
2. 点「新增」→ 填表 → 提交，确认抽屉关闭、列表刷新、新增行出现
3. 点某行「编辑」→ 改字段 → 提交，确认列表对应行更新
4. 勾选行 → 「批量删除」→ 确认，确认删除生效、分页总数减少
5. 切换暗色主题 / 切换语言，确认样式与文案无误
预期：<每一步的预期结果>；若任一步不符，把现象/截图发我。
```

清单要可执行、可对照，给出每步预期，而不是泛泛"看看对不对"。
