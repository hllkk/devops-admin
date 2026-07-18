# 字典管理（Dict Management）

> 类型：业务模块需求 · 状态：前端已就绪，后端 Model + dict type 全套 + dict data 全套已落地；菜单+按钮权限种子 07-15 已存在（refreshCache 缓存层待补；apis→casbin 策略机制项目级未落地，见 [[menu-seed-routes-alignment]]）

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

## 后端（Model 已落地，dict type list 接口已落地，其余待补）

- **2026-07-15 model 落地**：新增 `SysDictType`（`sys_dict_type`）、`SysDictData`（`sys_dict_data`）两个模型，对齐前端 `Api.System.DictType/DictData`，已注册进 `initialize/gorm.go` 的 `RegisterTables`（AutoMigrate），`go build ./...` 通过。
- 字段决策（严格对齐前端类型）：
  - 两表**均无 status**——前端 `DictType/DictData` 及 operate/search params 均未含（区别于 RuoYi 标准）。
  - `dict_type`：类型表 `uniqueIndex:uk_dict_type`（`/data/type/{dictType}` 查询键），数据表普通 `index:idx_dict_data_type`。
  - `isDefault`→`string`(Y/N)；`listClass`→`string`(ThemeColor)；`isI18n`→`bool` 预留；`createDept` 不建（前端 CommonRecord 残留，同 SysPost 不建 tenantId）。
  - 基座 `OPS_AUDIT_MODEL`，主键 `dictId/dictCode` 雪花 `int64` + `json:",string"`。
- **2026-07-18 dict type list 接口落地**：打通 `GET /system/dict/type/list`（query 传参，对齐前端 `DictTypeSearchParams`/`DictTypeList`）：
  - request `DictTypeSearch`（`model/system/request/sys_dict.go`，内嵌 `PageInfo` + `dictName/dictType`，`form` tag 适配 GET query）。
  - service `DictTypeService.GetDictTypeList`（`service/system/sys_dict.go`，`dictName/dictType` 模糊过滤 + `LimitOffset` 分页 + `dict_id DESC` 排序），已注册进 `service/system/enter.go`。
  - api `DictApi.GetDictTypeList`（`api/v1/system/sys_dict.go`，`ShouldBindQuery` + `response.PageResult` 响应 + 完整 swag 注释），已注册进 `api/v1/system/enter.go`。
  - router `DictRouter.InitDictRouter`（`router/system/sys_dict.go`，挂 `system/dict/type` group），已注册进 `router/system/enter.go` 与 `initialize/router.go` 的 PrivateGroup（鉴权/操作日志由该组全局中间件统一处理，子 group 不重复挂）。
  - `go build ./...` 通过。
- **2026-07-18 type 增删改 + optionselect 落地**（在 list 基础上同四层文件扩展）：
  - request `DictTypeOperateParams`（dictId/dictName/dictType/remark，对齐前端 `DictTypeOperateParams`，create 时 dictId 空）。
  - service：`CreateDictType`（dictType 唯一校验 + 审计字段从 claims 注入）、`UpdateDictType`（唯一校验排除自身 + dictType 变更时同步 `dict_data.dict_type` 冗余列）、`DeleteDictType`（批量 + 级联清理对应 `dict_data`，对齐 RuoYi 避免孤儿数据）、`GetDictTypeOptionList`（全量，下拉框）。
  - 审计字段 `CreateBy/UpdateBy` 由 API 层 `utils.GetUserID(c)` 取 claims 后传 service（service 不依赖 gin.Context）；struct literal 因 Go 内嵌提升字段规则改用赋值写入。
  - api：`CreateDictType`(POST)、`UpdateDictType`(PUT)、`BatchDeleteDictType`(DELETE `/system/dict/type/:ids`，逗号分隔解析)、`GetDictTypeOption`(GET optionselect)；写操作返回 `data=true` 对齐前端 `request<boolean>`。
  - router 注册同 group 新增 POST/PUT/DELETE`:ids`/GET optionselect 四条。
  - `go build ./...` + `go vet ./...` 通过。
- **2026-07-18 dict data 全套落地**（新增 `DictDataService`，复用同一 `DictApi`/`DictRouter`，对齐前端 `dict-data.ts`）：
  - request `DictDataSearch`（dictLabel/dictType + PageInfo）、`DictDataOperateParams`（dictCode/dictSort/dictLabel/dictValue/dictType/cssClass/listClass/isDefault/remark，不含 isI18n/i18nKey，对齐前端 operate params）。
  - service：`GetDictDataList`（dictLabel 模糊、dictType 精确，`dict_sort ASC, dict_code ASC` 排序）、`CreateDictData`/`UpdateDictData`（审计字段从 claims 注入）、`DeleteDictData`（按 dictCode 批量）、`GetDictDataByType`（按 type 查全量，DictTag/DictRadio 渲染用）。已注册进 `service/system/enter.go`。
  - api：`GetDictDataList`(GET list)、`GetDictDataByType`(GET `/type/:dictType`)、`CreateDictData`(POST)、`UpdateDictData`(PUT)、`BatchDeleteDictData`(DELETE `/:dictCodes`)；写操作 `data=true`。`api/v1/system/enter.go` 加 `dictDataService`。
  - router `system/dict/data` group 注册 5 条。
  - 顺手将批量删除 ID 解析改为 `strings.SplitSeq`（Go 1.26 + 项目 `utils/sse` 已用先例，IDE lint 提示更高效）。
  - `go build ./...` + `go vet ./...` 通过。
- 仍未实现：type 的 `refreshCache`（依赖字典缓存层，当前 list/optionselect/`data/type` 直查 DB 无缓存，DictTag 每次落库；待缓存设计后统一实现并接 refreshCache）。
- **菜单/权限种子已存在（2026-07-18 核对纠正）**：`source/system/sys_menu.go` 早在 07-15 已 seed `route.system_dict` C 菜单 + 5 个 F 按钮（`system:dict:query/add/edit/remove/export`），供前端按钮显隐。**注意**：业务记忆曾记"菜单/权限种子待补"是基于 [[menu-seed-routes-alignment]] 的 apis 规划态误判——实际 `SysMenu` 无 `Apis` 字段、`service/system/sys_casbin.go` 不存在、`CasbinHandler` 在 `initialize/router.go:79` 被注释，apis→casbin 策略机制项目级未落地；当前 dict 接口仅过 JWT 鉴权即可访问，启用 casbin 是另一个独立功能。
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
- 后端 dict type list：`server/model/system/request/sys_dict.go`、`server/service/system/sys_dict.go`、`server/api/v1/system/sys_dict.go`、`server/router/system/sys_dict.go`（及对应 `enter.go` 注册、`initialize/router.go` 挂载）
- 关联：[[menu-management]]（菜单+权限种子规划参考）
