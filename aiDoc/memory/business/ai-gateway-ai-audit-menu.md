# AI 网关 · 「AI审计」目录收编审批管理/调用日志菜单

日期：2026-08-28
状态：已实现(重新初始化数据库后生效)
反向链接：[[ai-gateway-resource-application]]、[[ai-gateway-usage-log-page]]、[[ai-gateway-ai-capability-menu]]

## 需求

用户需求：①「资源申请」菜单改名「审批管理」；②新建顶层目录「AI审计」，审批管理与
调用日志(名称不变)收编为其下两个子菜单，顺序：审批管理、调用日志；③不生成已有库
补丁，用户直接重新初始化项目生效。

## 实现

- 后端 `source/system/sys_menu.go`：rootMenus 删 `route.usage`/`route.application`
  两条顶层 C，新增目录 `route.ai-audit`(M,path=ai-audit,Layout,
  icon=lucide:file-search,OrderNum 11)；childMenus 增
  `route.ai-audit_approval`(OrderNum 1,原 route.application 更名,
  ApiPrefix 沿用 /gateway/application 管理接口枚举)与
  `route.ai-audit_usage`(OrderNum 2,原 route.usage 平移，
  ApiPrefix 沿用 /gateway/usage)——casbin 策略零改动。
  sys_role_menu 对 super/admin 全量授权，无需动；user 角色不授，行为不变。
- 前端 views：`_gateway/{application,usage}` →
  `_gateway/ai-audit/{approval,usage}`(git mv 保历史，modules 子文件名不变)；
  elegant 四件(imports/routes/transform/typings)由运行中 vite dev 自动重生成，
  routes.ts 目录 meta 手补 icon/order/module 与后端对齐(插件再生成的裸条目去重)。
- i18n：route 段删 `usage`/`application`，新增 `'ai-audit'`(AI审计)/
  `'ai-audit_approval'`(审批管理)/`'ai-audit_usage'`(调用日志)；
  app.d.ts 由 `Record<I18nRouteKey,string>` 派生自动联动。
- 注释同步：`router/gateway/resource_application.go` 头注释的菜单名引用改
  `route.ai-audit_approval`。
- 无已有库补丁：菜单 ID/父子关系变更走重新初始化(用户明确选择)。

## 验证

- go build 通过；pnpm typecheck 通过；全仓无旧路由名(`/application`、`/usage`、
  `route.application`、`route.usage`)残留引用。
- 生效方式：重新初始化数据库(新库走 sys_menu 种子)后重启前后端。
