# 前端规范 (frontend-rules)

> 适用范围：仅 `web/`。基座 = SoybeanAdmin 2.x（Vue3 + Vite + TS + NaiveUI + UnoCSS + Elegant Router + `@sa/axios` + vue-i18n，pnpm monorepo）。
> ⚠️ `web/` 尚未 scaffold，本文件为 SoybeanAdmin 2.x **前瞻性规范**，脚手架生成后据实校准。

## 1. 路由（Elegant Router，最易错）
- 基于**文件**：`src/views/*` 文件 → 自动生成 `src/router/elegant/{routes,imports,transform}.ts` 与 `typings/elegant-router.d.ts`（`RouteKey` 联合类型）
- **严禁手改 `src/router/elegant/`**；新增/改页面后跑 `pnpm gen-route`（`sa gen-route`）重生成
- 导航守卫在 `src/router/guard/`，静态路由在 `src/router/routes/builtin.ts`
- 支持前端静态 + 后端动态路由

## 2. 请求（@sa/axios）
- 用 `@sa/axios` 的 `createRequest`（抛错式）或 `createFlatRequest`（返回 `{ data, error, response }`，推荐）
- 钩子：`onRequest`（注 token）、`isBackendSuccess`（**判 `code === 0`**）、`onBackendFail`、`onError`、`transform`
- 实例配在 `src/service/request/`，API 函数放 `src/service/api/`，类型放 `src/typings/api/`
- 自带 `REQUEST_ID_KEY` 头、`AbortController` 取消、`axios-retry`
- **禁止裸用 axios**；**禁止重复封装** HTTP

## 3. 状态（Pinia，setup 风格）
- 全局状态用 Pinia `defineStore` + setup 写法（`ref` / `computed` / 函数）
- 模块按业务划分，放 `src/store/modules/`
- **严禁组件内直接改全局状态**，通过 actions

## 4. 主题
- 主题配置在 `src/theme/settings.ts`（`themeScheme` light/dark/auto、`themeColor`、`themeRadius`、`otherColor`、`layout.mode`、`tokens.light/dark`）
- 改主题只改此文件 + `overrideThemeSettings`，经 `@sa/color` + UnoCSS 生效
- **禁止硬编码颜色**；用 UnoCSS 原子类 / CSS 变量 / 设计 token

## 5. 图标
- 双体系：
  - 本地 SVG 放 `src/assets/svg-icon/*.svg`，经 `vite-plugin-svg-icons` 注册，`getLocalIcons()` 枚举
  - Iconify（`@iconify/vue` + `@iconify/json`），`setupIconifyOffline()` 走离线 / `VITE_ICONIFY_URL`
- 路由/菜单图标用**图标名字符串**（如 `"mdi:folder"`）

## 6. 国际化（vue-i18n）
- 语言文件 `src/locales/langs/{zh-cn,en-us}.ts`
- 所有用户可见文案走 i18n key，**禁止硬编码中文**
- dayjs locale 在 `src/locales/dayjs.ts` 同步

## 7. 命令行与脚本
| 命令 | 作用 |
|---|---|
| `pnpm dev` / `pnpm dev:prod` | 开发（test/prod 模式） |
| `pnpm build` / `pnpm build:test` | 构建 |
| `pnpm gen-route` | 重新生成 Elegant 路由 |
| `pnpm commit` | `sa git-commit` 生成规范提交（`-l=zh-cn` 中文） |
| `pnpm release` | `sa release` 发版 |
| `pnpm update-pkg` | `sa update-pkg` 升级依赖 |
| `pnpm cleanup` | `sa cleanup` 清理 |
| `pnpm lint` | `oxlint --fix && eslint --fix .` |
| `pnpm fmt` | `oxfmt` 格式化 |
| `pnpm typecheck` | `vue-tsc --noEmit --skipLibCheck` |
- `simple-git-hooks`：pre-commit 强制 `typecheck && lint && fmt`，commit-msg 校验提交信息——**提交前必须过**

## 8. 规范
- ESLint（`@soybeanjs/eslint-config-vue`）+ `oxlint`（快）+ `oxfmt`（格式）+ `vue-tsc` 严格类型
- 命名：文件 `kebab-case`；组件 `PascalCase`；变量/函数 `camelCase`；常量 `UPPER_SNAKE_CASE`
- TS 严格模式，禁 `any`（必要时 `unknown` + 类型守卫）
- Vue SFC：`<script setup lang="ts">` 在前，`<template>`，`<style>`；优先 Composition API
- 组件分层：`src/components/{advanced,common,custom}`

## 9. 组件与页面
- 可复用 UI 必须抽组件，单一职责，完整 props/emit 定义
- 页面放 `src/views/`，按业务模块组织
- 优先 NaiveUI 组件 + UnoCSS 类名；**禁内联样式**

## 10. 工具复用
- 先查 `@sa/*` workspace 包、`src/service/`、`src/utils/`、`src/hooks/`，**禁止重复造轮子**
- 详见 `frontend-utils.md`
