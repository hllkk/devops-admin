# 字典管理（Dict Management）

> 类型：业务模块需求 · 状态：前端已就绪，后端整体缺失（用户于 2026-07-13 决定后端延后单独规划）

## 需求

系统级「字典管理」：左侧字典类型树 + 右侧字典数据表，支持字典类型/字典数据的增删改查、刷新缓存、导出。
对齐 RuoYi 字典契约，前端由用户先行接入路由与页面。

## 前端（已就绪）

- 路由 `system_dict`（`/system/dict`，Elegant Router 静态生成），父级 `system`。i18nKey `route.system_dict`。
- 页面 `web/src/views/_admin/system/dict/`：`index.vue` + 3 个 module（`dict-data-search`、`dict-data-operate-drawer`、`dict-type-operate-drawer`）。
- service：`web/src/service/api/system/dict.ts`（类型 CRUD/optionselect/refreshCache）、`dict-data.ts`（数据 CRUD/export），均由 `system/index.ts` re-export。
- 类型 `Api.System.DictType/DictData/*SearchParams/*OperateParams/*List`（`system.api.d.ts`）齐全。
- 复用组件：`TableSiderLayout`、`TableRowCheckAlert`、`TableHeaderOperation`、`DictTag`、`DictRadio`、`ButtonIcon`；hooks `useDict`、`useDownload`、`useAuth`；util `handleCopy`。
- i18n：`page.system.dict` 全量（zh/en + `app.d.ts` 声明）齐全。权限码 `system:dict:add/edit/remove/export`（list 隐含）。

## 后端（缺失，延后）

> 用户决定：后端先跳过，后续再单独规划补齐。

- 目前后端**无任何字典代码**：无 `SysDictType/SysDictData` 模型、无 service/api/router。`/system/dict/type/*`、`/system/dict/data/*` 接口均未实现。
- 菜单/权限种子亦未做：`server/source/` 为空，`RegisterInit` 从未被调用，`sys_menu` 无 `system:dict:*` 记录。
- 用户原话「后端的菜单模型定义」经澄清 = 延后；后续可参照 [[menu-management]] 文档里 `menu.go 初始化种子：默认菜单与权限` 的规划，补字典模块的菜单+按钮权限种子。

## 本次（2026-07-13）前端补齐

- `index.vue` 导出文件名硬编码 `字典数据_` → `$t('page.system.dict.dictData')`（新增 `dictData` key，zh/en/`app.d.ts` 三处同步）。
- `index.vue` 表头操作权限码从 `system:user:*` 修正为 `system:dict:*`（add/remove/export）——复制 user 页面遗留的错码。
- `dict-type-operate-drawer.vue` 字典类型输入框 placeholder 误用 `form.dictValue.required` → 改为 `form.dictType.required`。
- 补 `common.selected/anyRecords/clear/noSelectRecord`（`TableRowCheckAlert` 使用，`app.d.ts` 已声明但两个语言文件缺失）。
- `web/src/utils/copy.ts` 硬编码中文（复制成功 / 不支持 Clipboard API）→ `common.copySuccess/copyNotSupported`（共享 util，字典页复制字典类型时触发）。

## 备注

- `web/src/typings/components.d.ts` 为自动生成、当前**陈旧**（未含 `TableSiderLayout/TableRowCheckAlert/DictRadio` 等），`pnpm dev`/`build` 时由 `unplugin-vue-components` 自动重生成，pre-commit typecheck 前需先跑一次 dev 或 build。

## 相关文件

- 前端：`web/src/views/_admin/system/dict/`、`web/src/service/api/system/{dict,dict-data}.ts`、`web/src/typings/api/system.api.d.ts`、`web/src/locales/langs/{zh-cn,en-us}.ts`、`web/src/typings/app.d.ts`
- 关联：[[menu-management]]（菜单+权限种子规划参考）
