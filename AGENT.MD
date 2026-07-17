# AGENT.MD

## 目的

本文件是本仓库内 AI 协作规则的唯一真源。

`.codex/`、`.claude/`、`.cursor/`、`.trae/` 下的规则文件仅作为兼容适配层，不能再次演变成各自独立维护的 project rule 副本。

## 交互语言

- 所有 AI 协作过程（思考、回答、解释、代码注释、提交信息、文档、错误说明、计划）统一使用中文
- 代码标识符、命令、字段名、配置项、第三方库名等保持原文，不翻译

## 读取顺序

按下面顺序加载项目上下文：

1. `AGENT.MD`
2. `aiDoc/README.md`
3. 按任务读取以下目录中的相关文件：
   - `aiDoc/relations/`
   - `aiDoc/modules/`
   - `aiDoc/frontend-backend/`
   - `aiDoc/examples/`
   - `aiDoc/memory/`
4. 仅在当前工具确实依赖时，再读取工具目录下的适配文件。

若内容冲突，以 `AGENT.MD` 为准。

## 仓库概览

- `server/`：Go + Gin + GORM 后端
- `web/`：Vue3 + Vite + TS + NaiveUI + UnoCSS + Elegant Router 前端
- `deploy/`：部署资产
- `docs/`：项目文档与设计记录
- `aiDoc/`：AI 协作文档层

## 工程规则

### 架构

- 保持现有后端分层：`Router -> API -> Service -> Model`
- `enter.go` 继续作为分组注册与组合入口
- API 层处理 HTTP 相关逻辑，Service 层不要依赖 `gin.Context`
- 对外接口的 Swagger 注释必须和真实行为保持一致
- Swagger 的 `@Success` 响应要落到具体类型，让 swag 能生成真实返回结构：列表用 `response.PageResult{Rows=[]Model}`（`PageResult` 字段为 `Rows/Total/PageNum/PageSize`）、详情用具体 model，不要停留在空的 `response.PageResult` 或 `data=object`（仅动态/示例数据才用 `object`）；细则见 `aiDoc/modules/backend-layer-rules.md`
- 列表分页统一走 `request.PageInfo`：Service 层取 limit/offset 一律用 `info.LimitOffset()`（内置 `MaxPageSize=100` 截断），不要手写 `PageSize*(Page-1)` 换算
- `CreateBy`/`UpdateBy`/`DeptId` 公共操作字段（审计基座 `OPS_AUDIT_MODEL` 提供 `create_by`/`update_by`；`dept_id` 为数据权限的归属部门列；`deleted_by` 不是基座字段，由数据权限引擎在表含该列时软删除盖章）仅在业务表需要数据权限时才创建（对应代码生成器的 AutoCreateResource），手写时不要自造 `CreatorID`/`CreatedBy` 之类同义字段；细则见 `aiDoc/modules/backend-layer-rules.md`
- 优先沿用 devops-admin 现有模式，不做无关大改

### 前后端协作

- 明确请求与响应契约
- 保持统一响应结构：`{ code, data, msg }`
- 保持统一分页结构：`{ pageNum, pageSize, total, rows }`
- 前后端字段名与数据类型保持一致
- 优先复用 `web/` 的 `@sa/*` 包与 `src/{service,utils,hooks}`，**禁止重复造轮子**
- 涉及跨栈边界变更时，**同步更新 `aiDoc/frontend-backend/`**

### 前端上游同步

- 同步 soybean-admin 上游用 `scripts/sync-upstream.sh`：`--preview` 预览、`--run` 执行
- `web/` 是**直接复制导入**、未经 `git subtree add`，与上游无合并基点；脚本默认走 **apply 模式**：`git diff <track sha> <上游HEAD> | git apply --directory=web`，仅当检测到合并基点时才回退 `git subtree pull`
- 上游增量小且改动文件本地未动 → 0 冲突自动应用并提交；存在冲突则逐文件报告、**不改动工作区**，人工解决后重跑
- 细则与冲突处理见 `aiDoc/relations/development-workflow.md`

### 前端样式

- 前端已接入 UnoCSS（`web/uno.config.ts`，presetWind3）。**写样式优先用 UnoCSS 原子类**：布局、间距、排版、尺寸、颜色等能用原子类表达的，一律用原子类，不再新增自定义 class
- 只有以下情况才写 `<style scoped>`：需要 `:deep()` 覆盖 NaiveUI 内部样式、伪类/伪元素、复杂选择器，或原子类确实难以表达的样式
- 避免内联 `style`；动态样式用绑定的原子 class 或 CSS 变量，主题相关能力优先走 CSS 变量
- 细则见 `aiDoc/frontend-backend/frontend-rules.md`

### 图标

- 双体系:本地 SVG(`web/src/assets/svg-icon/`,经 `vite-plugin-svg-icons` + UnoCSS `FileSystemIconLoader` 在**构建期**注册)与 Iconify(`@iconify/vue`,离线走 `setupIconifyOffline()` / `VITE_ICONIFY_URL`);菜单/路由图标用**图标名字符串**引用,渲染统一走 `web/src/components/custom/svg-icon.vue`
- **菜单图标**:本地款放 `web/src/assets/svg-icon/menu/<name>.svg`,路由里以 `localIcon: 'menu-<name>'` 引用(渲染为 `#icon-local-menu-<name>`);也可用 Iconify 名(如 `icon: 'lucide:users'`)。优先线框/空心风格,避免视觉实心款;**找不到合适的空心款时,去 Iconify(优先 `lucide`)挑一个合适的空心 svg、规整后存为本地 `web/src/assets/svg-icon/menu/<name>.svg`(不要手画自己发挥,也不要将就用实心款)**;新增本地 svg 后由构建期扫描注册,dev 按需热更、生产构建全量进 sprite,**根 `<svg>` 不要写固定 `stroke-width`/`fill`,颜色用 `currentColor` 由外层控制**
- **其它系统 / 业务开发的图标**:只要语义合适即可,**不必在意空心还是实心**;优先复用现有图标集(lucide 等),确实没有合适的再往 `svg-icon/` 加本地 svg
- 细则见 `aiDoc/frontend-backend/frontend-rules.md` 的「图标」一节

### 页面点触测试(AI 驱动浏览器验证)

- 前端页面改动需要真实浏览器点触验证时,按 `aiDoc/frontend-backend/page-click-testing.md` 执行
- **登录态契约(目标:httpOnly cookie 模式,重构后待重建,详见 `aiDoc/frontend-backend/boundary.md`「认证」一节)**:当前后端 `/auth/*` 路由尚未重建(`server/api/` 为空)、登录链路暂不可用;目标态为 token 不进 JS、前端不下发 Authorization。浏览器自动化**不能**用"向用户索取 token 注入 localStorage"的老办法,维持登录态走 Playwright 的 `storageState` 复用 / `context.addCookies` 注入后端下发的 cookie,或在自动化浏览器里走真实登录(用户名+密码+验证码);详见该文档
- 浏览器自动化能力当前环境不具备时,**主动建议用户安装**(按该文档「环境前置」一节的话术给出推荐项,首选 Playwright),经用户确认后再装,不要不问就装;用户拒绝则回退为"给出人工目测清单"
- 需要覆盖登录链路本身(验证码/锁定)时,用「系统设置 → 安全配置」的验证码阈值临时直登,测完改回
- cookie / storageState 是真实凭证:缓存只能落 `.local/`(已 gitignore),不写入任何会提交的文件、不出现在截图/日志/commit 信息里;点触造成的破坏性数据操作先征得用户同意

### 插件与模块

- 后端插件放 `server/plugin/<name>/`
- 前端插件按 `web/src/plugins/<name>/` 
- 稳定的模块职责、入口与边界说明放 `aiDoc/modules/`

### 示例文档

- `aiDoc/examples/` 是讲解型示例层
- 示例**不是逐字复制**，而是示范写法
- 新增某层文件前，先读对应示例再实现

### 记忆规则

- `aiDoc/memory/long-term/` 用于记录长期稳定的用户偏好、协作方式和跨任务约束
- `aiDoc/memory/business/` 用于记录每次用户提出的业务需求
- 这里的“业务需求”指新增或修改模块、接口、页面、流程、业务规则、插件能力等项目需求
- **一条记忆的粒度是「一个功能点 / 一次独立的变更意图」，不是「一个大模块」**：同一大模块下的不同功能点必须各自独立成一个文件，禁止用“新增子需求”的方式把新功能追加进已有文件
- 用户提出新的功能点时，AI 必须新建一个 `business` 记忆文件，并在 `demand-index.md` 中登记
- 只有更新**同一条记忆**（同一个功能点）的状态、进展或已确认约束时，才编辑那个已有文件
- 同一大模块下的多个功能文件，用统一文件名前缀归组（如 `cli-xxx.md`），并在文件之间用反向链接关联上下文
- 当某条业务需求沉淀成长期稳定偏好时，再提炼到 `long-term`
- 长期记忆按主题一条一文件维护，业务记忆按功能点一条一文件维护
- `README.md`、`project-memory.md`、`demand-index.md` 这类文件只承担说明或索引职责，不承载多条记忆正文
- 更新记忆时，只修改对应那条记忆文件与必要索引，不要通过重写汇总正文来覆盖其他历史记忆

### 文档维护

- `AGENT.MD` 只保留高层、稳定、跨任务的规则
- 细节上下文写入 `aiDoc/`
- 结构关系放在 `aiDoc/relations/`
- 示例写法放在 `aiDoc/examples/`
- 长期记忆与业务记忆放在 `aiDoc/memory/`
- 若项目级 AI 规则发生变化，先改 `AGENT.MD`，再按需更新适配层

### 代码读取约束

- **无论什么情况，都不要直接读取 `node_modules/` 中的代码**
- 了解第三方库行为时，优先看项目源码中的调用方式、锁文件、配置、官方文档或包的公开类型/说明，而不是进 `node_modules/` 逐文件读实现
- 后端同理：优先看 `go.mod` 与项目内调用，不深挖 vendor 实现

### 工作流与 superpowers

- **不要自动套用 superpowers 重流程**（brainstorming / writing-plans / subagent-driven-development / requesting-code-review 等）。superpowers 通过 SessionStart 注入的 `using-superpowers` skill 会强烈倾向自动调用，本规则覆盖该倾向
- **进入规划/计划类任务前**（新功能设计、跨端改动、有设计不确定性的重构、多步骤实现），先用 `AskUserQuestion` 让用户选择：启用 superpowers 重流程，还是走项目自身轻量流程
- **简单改动**（bug 修复、单点调整、已有明确设计的实现）不问，直接按本文件与 `aiDoc/` 规范实现
- 用户指令优先级高于 skill：用户明确要求轻量或指定流程时，以用户为准
- 仅当用户同意后，才调用对应的 superpowers skill
- **superpowers 产物统一存放于本地 `.superpowers/`，不提交仓库**：spec / plan / sdd 等过程性文档一律放入 `.superpowers/`（根 `.gitignore` 已忽略整个目录）。确有长期价值的设计决策，应剥离 superpowers 标签后写入 `aiDoc/`（稳定约束）或 `docs/`（设计记录），不要以 superpowers 产物形式入库

## AI 文档索引

- `aiDoc/README.md`
- `aiDoc/relations/repo-profile.md`
- `aiDoc/relations/development-workflow.md`
- `aiDoc/relations/system-map.md`
- `aiDoc/modules/backend-layer-rules.md`
- `aiDoc/modules/business-modules.md`
- `aiDoc/modules/plugin-development.md`
- `aiDoc/frontend-backend/boundary.md`
- `aiDoc/frontend-backend/frontend-rules.md`
- `aiDoc/frontend-backend/page-click-testing.md`
- `aiDoc/frontend-backend/frontend-utils.md`
- `aiDoc/examples/README.md`
- `aiDoc/examples/backend/`
- `aiDoc/examples/frontend/`
- `aiDoc/examples/plugin/`
- `aiDoc/memory/README.md`
- `aiDoc/memory/project-memory.md`
- `aiDoc/memory/long-term/`
- `aiDoc/memory/business/`
