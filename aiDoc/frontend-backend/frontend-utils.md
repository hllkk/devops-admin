# 前端工具复用 (frontend-utils)

> 适用范围：`web/`（SoybeanAdmin 2.x + RuoYi 约定混合体）。
> **新增功能前先查本表，禁止重复造轮子。** 范式与用法见 `frontend-rules.md` §9。

## Hooks

| 场景 | 用什么 | 位置 |
|---|---|---|
| 数据表格（分页 / 列设置 / 增删改抽屉联动 / 树表） | `useNaiveTable`、`useNaivePaginatedTable`、`useNaiveTreeTable`、`useTableOperate`、`useTreeTableOperate`、`treeTransform`、`defaultTransform` | `src/hooks/common/table.ts` |
| 表单校验 | `src/hooks/common/form.ts` 的校验规则工厂 | `src/hooks/common/form.ts` |
| 字典数据 | `useDict(dictType)` → `{ data, record, options, transformDictData }`（Pinia `useDictStore` 缓存） | `src/hooks/business/dict.ts` |
| 权限判断 | `useAuth()` → `hasAuth('system:dept:add')` | `src/hooks/business/auth.ts` |
| 图标 / 路由 / 图表 | `src/hooks/common/{icon,router,echarts}.ts` | `src/hooks/common/` |
| 验证码 / 下载 | `useCaptcha` / `useDownload` | `src/hooks/business/{captcha,download}.ts` |
| 通用基座 | `useBoolean`、`useTable`（+ 类型 `PaginationData`/`TableColumnCheck`/`UseTableOptions`） | `@sa/hooks` |

## 组件（`src/components/`）

| 层 | 组件 | 用途 |
|---|---|---|
| `advanced/` | `table-sider-layout` | 左树/左栏 + 右表的响应式主从布局（岗位/用户等左树过滤页用） |
| | `table-header-operation` | 列表头：新增 / 刷新 / 批量删 + 列设置入口（`TableHeaderOperation`） |
| | `table-column-setting` | 表格列显隐与排序 |
| | `table-row-check-alert` | 行勾选后的提示条 |
| `common/` | `app-provider`、`system-logo`、`lang-switch`、`theme-schema-switch`、`full-screen`、`reload-button`、`menu-toggler`、`pin-toggler`、`icon-tooltip`、`dark-mode-container`、`exception-base` | 基座通用件 |
| `custom/` | `button-icon` | 表格操作列统一按钮（icon + tooltip + popconfirm） |
| | `dict-select` / `dict-radio` / `dict-checkbox` / `dict-tag` | 字典驱动的表单控件 / 列表回显标签 |
| | `status-switch` | 启用/停用（`EnableStatus`）状态开关 |
| | `dept-tree` / `menu-tree` | 部门树 / 菜单树选择 |
| | `file-upload` / `svg-icon` / `soybean-avatar` / `count-to` / `wave-bg` / `better-scroll` / `look-forward` / `module-select` | 其他业务通用件 |

## 工具与常量

| 场景 | 用什么 |
|---|---|
| HTTP 请求 | `@sa/axios`（`createFlatRequest`），实例 `src/service/request/index.ts` |
| 颜色 / 主题 token | `@sa/color` |
| 通用工具 | `@sa/utils`（`jsonClone` 等）；`src/utils/common.ts`（`handleTree` 扁平构树、`isNull`、`transformRecordToOption`） |
| 深拷贝 / 加密 / 存储 | `src/utils/{copy,crypto,storage}.ts`（`localStg`） |
| 本地图标 | `src/utils/icon.ts` 的 `getLocalIcons()`；图标标签格式化 `icon-tag-format.ts` |
| 服务 baseURL | `src/utils/service.ts` 的 `getServiceBaseURL()` |
| 业务常量 | `src/constants/business.ts`（`enableStatusRecord/Options`、`yesOrNoStatusRecord/Options`、`menuTypeRecord`…）、`common.ts`、`reg.ts`、`env.ts`、`module.ts`、`app.ts` |

## 类型范式（`src/typings/api/api.d.ts`）

- `Api.Common.CommonRecord<T>`：对外业务实体的审计基座 `{ createBy, createDept?, createTime, updateBy, updateTime } & T`。业务模型一律 `= Common.CommonRecord<{…}>`，实体专属字段（含 `remark`）放进 `T`。
- `Api.Common.PaginatingCommonParams`：`{ pageNum, pageSize?, total }`；`Api.Common.PaginatingQueryRecord<T>` 在其上加 `rows: T[]` —— 即分页响应 `{ pageNum, pageSize, total, rows }`。
- `Api.Common.CommonSearchParams`：`Pick<pageNum,pageSize> & RecordNullable<{ orderByColumn, isAsc, params }>`（RuoYi 搜索片段，含排序与扩展参数袋）。
- `Api.Common.EnableStatus = '0'|'1'`、`VisibleStatus = '0'|'1'`、`YesOrNoStatus = 'Y'|'N'`、`CommonTreeRecord`。
- `CommonType.IdType`：雪花 ID 的字符串类型，**所有 ID 字段一律用它，禁当 number 运算**（见 `boundary.md` 主键契约）。
- `CommonType.RecordNullable<T>`：搜索 / 操作参数的 nullable 包装；`CommonType.Option`：下拉选项 `{ label, value }`。

## 禁止

- 裸用 axios / 手写 fetch 封装
- 硬编码颜色、手写日期格式化、手写命名转换、手写扁平构树
- 重复实现已有 hook / 组件 / 工具 / 常量
