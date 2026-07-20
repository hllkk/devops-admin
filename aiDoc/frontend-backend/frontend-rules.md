# 前端规范 (frontend-rules)

> 适用范围：仅 `web/`。基座 = SoybeanAdmin 2.x（Vue3 + Vite + TS + NaiveUI + UnoCSS + Elegant Router + `@sa/axios` + vue-i18n，pnpm monorepo）+ **RuoYi 约定混合体**（`CommonRecord`/`PaginatingQueryRecord`/`EnableStatus` 审计与状态类型、`system:模块:动作` 权限码、字典体系、`/system/<m>/*` REST 约定）。
> `web/` **已 scaffold** 并有真实模块（`_admin/system/{user,role,menu,dept,post,dict,notice,setting}`，其中 `setting` 是标签页式系统配置页、非标准三件套，见 §9.9）。本文件为实况规范，新增模块前以 `dept` 模块为参照（见 §9）。

## 1. 路由（Elegant Router，最易错）
- 基于**文件**：`src/views/*` 文件 → 自动生成 `src/router/elegant/{routes,imports,transform}.ts` 与 `typings/elegant-router.d.ts`（`RouteKey` 联合类型）
- **严禁手改 `src/router/elegant/`**；新增/改页面后跑 `pnpm gen-route`（`sa gen-route`）重生成
- 导航守卫在 `src/router/guard/`，静态路由在 `src/router/routes/builtin.ts`
- 支持前端静态 + 后端动态路由

## 2. 请求（@sa/axios）
- 用 `@sa/axios` 的 `createRequest`（抛错式）或 `createFlatRequest`（返回 `{ data, error, response }`，推荐）
- 钩子：`onRequest`（注 token）、`isBackendSuccess`（**判 `String(code) === VITE_SERVICE_SUCCESS_CODE`，默认 `"0000"`，GVA 字符串码——不是 `code === 0`**）、`onBackendFail`（按 `web/.env` 的 logout/modal/expired code 分流登出与刷 token）、`onError`、`transform`
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

新增系统模块**以 `dept` / `post` / `menu` 为参照**，遵守下列强约定。

### 9.1 页面模块结构（三件套）
```
src/views/_admin/system/<m>/
  index.vue                            # 列表页 <script setup lang="tsx">（列 render 用 TSX）
  modules/<m>-operate-drawer.vue       # 新增/编辑抽屉（同表单复用）
  modules/<m>-search.vue               # 顶部搜索条件栏
src/service/api/system/<m>.ts          # CRUD：fetchGetXxx / fetchCreate / fetchUpdate / fetchBatchDelete
src/typings/api/system.api.d.ts        # Api.System.<Entity> + <Entity>SearchParams + <Entity>OperateParams + <Entity>List
```
- 树形模块（dept）：`index.vue` 用 `useNaiveTreeTable`，列表返回扁平数组，前端 `handleTree` 构树。
- 例外：`menu` 因交互形态特殊（树形 + 级联删除弹窗等），暂未接入 `useNaiveTreeTable`，改用 naive-ui `TreeInst` 自管；新增树形模块仍以 `dept` 为准、优先用 hook，不要把 `menu` 当模板。
- 分页模块（post/role/user）：`index.vue` 用 `useNaivePaginatedTable`，列表返回 `{ pageNum, pageSize, total, rows }`。

### 9.2 列表页 hook 体系（`src/hooks/common/table.ts`，禁止手搓表格）
- `useNaiveTable` / `useNaivePaginatedTable` / `useNaiveTreeTable`：产出 `columns/columnChecks/data/rows/getData/loading/scrollX`（树表再加 `expandedRowKeys/expandAll/collapseAll`）。
- `useTableOperate(rows, idKey, getData)` / `useTreeTableOperate`：产出 `drawerVisible/operateType/editingData/handleAdd/handleEdit/onDeleted/onBatchDeleted`，抽屉与表格增删改的标准联动。
- `defaultTransform` / `treeTransform`：响应 → 表格数据的标准转换。
- 详见 `frontend-utils.md`。

### 9.3 字典体系（RuoYi 风格）
- 取字典：`const { options } = useDict('sys_normal_disable')`（自动进 `useDictStore` 缓存）。
- 表单控件：`<DictSelect dict-code="..." />` / `DictRadio` / `DictCheckbox`。
- 列表回显：`<DictTag :value="row.status" dict-code="sys_normal_disable" />`。

### 9.4 权限控制（RuoYi 三段式权限码）
- 码格式：`<模块>:<资源>:<动作>`，如 `system:dept:add` / `system:dept:edit` / `system:dept:remove` / `system:post:export`。
- 判断：`const { hasAuth } = useAuth(); hasAuth('system:dept:add')`，用于按钮显隐与表格操作列渲染。

### 9.5 数据类型范式
- 实体：`type Dept = Common.CommonRecord<{ deptId: CommonType.IdType; … }>`。
- 搜索参数：`type DeptSearchParams = CommonType.RecordNullable<Pick<Dept, …> & Api.Common.CommonSearchParams>`。
- 操作参数：`type DeptOperateParams = CommonType.RecordNullable<Pick<Dept, …>>`（多对多附带 `menuIds/roleIds/postIds: CommonType.IdType[]`）。
- ID 一律 `CommonType.IdType`（字符串），禁当 number。详见 `frontend-utils.md` 类型范式。

### 9.6 REST 接口约定（`/system/<m>/*`）
- 列表 `GET /system/<m>/list`；排除树/下拉等子资源 `GET /system/<m>/list/exclude/{id}`、`/optionselect`。
- 新增 `POST /system/<m>`；修改 `PUT /system/<m>`（同 path，按 body 是否带 id 区分）。
- 批量删除 `DELETE /system/<m>/{ids}`（路径逗号分隔，前端 `ids.join(',')`）。
- 接口封装统一 `request<T>(...)`（flat 模式，返回 `{ data, error }`）。

### 9.7 i18n key 约定
- 字段/文案：`page.system.<m>.<field>`；通用：`common.*`（`operate/add/edit/delete/confirmDelete`…）；路由：`route.system_<m>`。
- 表单：`form.<m>.<field>.required`（占位提示）、`form.<m>.<field>.invalid`（校验失败文案）。
- 新增路由必须同步补 `route.system_<m>` 中英文案，否则 `Record<I18nRouteKey, string>` 类型不闭合。

### 9.8 通用组件分层
- `src/components/advanced/`：表格相关复合件（`table-sider-layout`/`table-header-operation`/`table-column-setting`/`table-row-check-alert`）。
- `src/components/common/`：基座通用件（布局/主题/语言/全屏等）。
- `src/components/custom/`：业务通用件（`button-icon`/`dict-*`/`status-switch`/`dept-tree`/`menu-tree`/`file-upload`…）。
- 可复用 UI 必须抽组件，单一职责，完整 props/emit；优先 NaiveUI + UnoCSS 类名，**禁内联样式**。完整清单见 `frontend-utils.md`。

### 9.9 系统配置页（`setting`，非标准三件套）
- `setting` 是标签页式配置页，不走列表/抽屉三件套：`index.vue` 承载 `n-tabs`，每个 tab 对应 `modules/<area>-setting.vue`（`general-setting` 通用配置、`security-setting` 安全配置、`setting-menu` 菜单设置）。
- 新增「系统设置」分区时，沿用「加一个 `<area>-setting.vue` tab + 对应接口封装」的模式，不要套用 §9.1 的列表三件套。

## 10. 工具复用
- 先查 `@sa/*` workspace 包、`src/service/`、`src/utils/`、`src/hooks/`，**禁止重复造轮子**
- 详见 `frontend-utils.md`

## 11. 时间展示格式化（列表/详情通用）

后端时间字段一律以 ISO（RFC3339Nano，如 `2026-07-19T21:25:53.071037-04:00`）返回，**前端禁止直接渲染原始字符串**——不要 `{{ row.createTime }}`，列表列也不要只给 `key` 不给 `render`。统一用 NaiveUI `NTime` 在展示层格式化。

- 列表列（`<script setup lang="tsx">`）：补 `import { ..., NTime } from 'naive-ui'`，列加
  ```tsx
  render: row => <NTime time={Date.parse(row.xxxTime)} format="yyyy-MM-dd HH:mm:ss" />
  ```
- 详情/抽屉（模板）：
  ```html
  <NTime :time="Date.parse(data.xxxTime ?? '')" format="yyyy-MM-dd HH:mm:ss" />
  ```
  抽屉数据可能为 null，`?? ''` 兜底；`Date.parse('')` 返回 `NaN`，`NTime` 渲染为空。
- **关键坑**：`NTime` 的 `time` 只收 `number | Date`，**不收 string**。后端 ISO 是 string，必须 `Date.parse(iso)` 转毫秒时间戳再传，否则 `vue-tsc` 报错。（在线用户表 `loginTime` 本身是 `number`，可直接传，是例外。）
- format 统一 `yyyy-MM-dd HH:mm:ss`（NaiveUI token，不是 dayjs 的 `YYYY-MM-DD`）。
- 列宽给够：`minWidth`/`width` ≥ 160，否则 `2026-07-19 21:25:53` 被截断；空间紧张时配 `ellipsis: { tooltip: true }` 兜底。
- 搜索表单里的 `createTime` 是 `NDatePicker`（日期范围**筛选**），不是展示，无需此处理。
- 已落地参照：`loginlog`/`operlog`（loginTime/operTime）、`user`/`role`/`dept`/`post`/`menu`/`notice`/`dict`/`role-auth-user-drawer`（createTime）。
- 契约面（后端为何保持 ISO、禁止改全局序列化）见 `boundary.md`「时间字段契约」。
