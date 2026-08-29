# AI 网关 · 「AI能力」目录收编 MCP/Skill 菜单

日期：2026-08-28
状态：已实现(待用户运行时验证)
反向链接：[[ai-gateway-mcp-server]]、[[ai-gateway-skill]]

## 需求

用户需求：①「MCP 服务器」菜单改名「MCP 管理」；②MCP 管理与 Skill 管理从顶层单页
变更为「AI能力」目录下的两个子菜单。

## 实现

- 后端 `source/system/sys_menu.go`：rootMenus 删 `route.mcp`/`route.skill` 两条顶层 C，
  新增目录 `route.ai-capability`(M,path=ai-capability,Layout,icon=lucide:sparkles,
  OrderNum 13)；childMenus 增 `route.ai-capability_mcp`(OrderNum 1)/
  `route.ai-capability_skill`(OrderNum 2)，ApiPrefix 不变(casbin 策略零改动)。
  sys_role_menu 对 super/admin 全量授权，无需动。
- 前端 views：`_gateway/{mcp,skill}` → `_gateway/ai-capability/{mcp,skill}`
  (git mv 保历史)；elegant 四件(imports/routes/transform/typings)由运行中 vite dev
  自动重生成，routes.ts 目录 meta 手补 icon/order/module 与后端对齐。
- i18n：route 段删 `mcp`/`skill`，新增 `'ai-capability'`(AI能力)/
  `'ai-capability_mcp'`(MCP 管理)/`'ai-capability_skill'`(Skill 管理)；
  app.d.ts 由 `Record<I18nRouteKey,string>` 派生自动联动。
- 已有库补丁：`deploy/patches/2026-08-28-ai-capability-menus.sql`
  (INSERT 目录+UPDATE 两菜单挂目录改名+super/admin 补授权目录；幂等；
  取代同日早前被删除的 gateway-skill-mcp-menus 补丁)。

## 顺手修复(本次发现的隐藏问题)

- **`.gitignore` 的 `skill*` 规则误伤源码**：monorepo 初始化(soybean 模板)带入的
  规则匹配任意目录下 skill 开头的文件/目录，导致 Skill 全部核心源码从未入库——
  前端 `views/_gateway/skill/` 整目录、后端 7 个 skill.go 系列
  (api/model×3/router/service×2+测试)、前端 API `service/api/gateway/skill.ts`
  (提交 9433601 声称完成但上述文件全被忽略，只进了非 skill 开头的文件)。
  修法：规则锚定根路径 `skill*` → `/skill*`(根目录 skills-lock.json 仍被忽略)，
  全部受误伤源码随本次首次 add 入库。

## 验证

- go build/vet 通过；pnpm typecheck 通过。
- 待用户运行时验证：新库走种子；已有库执行新补丁后重启服务。
