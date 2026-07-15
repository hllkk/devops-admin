# 字典管理（Dict Management）

> 类型：业务模块需求 · 状态：前端已就绪，后端 Model 已落地（service/api/router/seed 待补）

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

## 后端（Model 已落地，接口待补）

- **2026-07-15 model 落地**：新增 `SysDictType`（`sys_dict_type`）、`SysDictData`（`sys_dict_data`）两个模型，对齐前端 `Api.System.DictType/DictData`，已注册进 `initialize/gorm.go` 的 `RegisterTables`（AutoMigrate），`go build ./...` 通过。
- 字段决策（严格对齐前端类型）：
  - 两表**均无 status**——前端 `DictType/DictData` 及 operate/search params 均未含（区别于 RuoYi 标准）。
  - `dict_type`：类型表 `uniqueIndex:uk_dict_type`（`/data/type/{dictType}` 查询键），数据表普通 `index:idx_dict_data_type`。
  - `isDefault`→`string`(Y/N)；`listClass`→`string`(ThemeColor)；`isI18n`→`bool` 预留；`createDept` 不建（前端 CommonRecord 残留，同 SysPost 不建 tenantId）。
  - 基座 `OPS_AUDIT_MODEL`，主键 `dictId/dictCode` 雪花 `int64` + `json:",string"`。
- 仍未实现：service/api/router（`/system/dict/type/*`、`/system/dict/data/*`）、菜单/权限种子（`server/source/` 无 `system:dict:*`）。
- 后续可参照 [[menu-management]] 文档里 `menu.go 初始化种子：默认菜单与权限` 的规划，补字典模块的菜单+按钮权限种子。

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
- 后端 model：`server/model/system/sys_dict_type.go`、`server/model/system/sys_dict_data.go`、`server/initialize/gorm.go`（RegisterTables）
- 关联：[[menu-management]]（菜单+权限种子规划参考）
