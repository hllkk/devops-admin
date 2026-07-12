# 系统初始化流程（借鉴 gin-vue-admin）

- 日期：2026-07-11
- 状态：已实现（前后端 + 文档），构建/类型/lint 全绿
- 关联分支：feat/multi-module-isolation

## 背景

借鉴 gin-vue-admin 的"首次启动自动初始化数据库"能力。后端 `sys_init*.go` 此前已完整移植（`SubInitializer` 自注册 + 四种 DB handler + 免重启热接入 `global.OPS_DB`），但前端只做到一半（`views/_builtin/init/` 空目录、`pwd-login.vue` 里有断的 `fetchInitDB` 导入和 stub 按钮），且后端响应码与前端约定不一致（阻断项）。

## 契约决策（用户拍板）

1. 检测时机：**路由守卫自动 `checkDB` 强制跳转** `/init`（优于 gva 的 dev-only 按钮/手改 URL）。
2. 进度展示：提交时**全屏 loading 单文案**（后端单次 POST 同步阻塞，无分步进度）。
3. 成功后：跳登录页，**不 reload**（后端热接入 DB，免重启）。
4. 响应码：**后端对齐前端**，`Response.Code` 由 `int(0/7)` 改为 `string("0000"/"0001")`（前端 `App.Service.Response.code` 本就是 `string`，`VITE_SERVICE_SUCCESS_CODE=0000`）。

## 实现要点

- 后端：`model/common/response/response.go` 的 `Code` 改 `string`、`SUCCESS="0000"`、`ERROR="0001"`，`NoAuth` 跟随；`service/system/auto_code_llm.go` 的 `code=%d`→`%s`；`mcp/http_client.go` 的 `upstreamEnvelope` 是独立结构、未改。已重生成 swagger。
- 前端：
  - 类型 `typings/api/init.d.ts`（`Api.Init.InitDBForm` / `CheckDBResult` / `DBType`）
  - 接口 `service/api/init.ts`（`fetchCheckDB` / `fetchInitDB`）+ barrel 导出
  - 向导页 `views/_builtin/init/index.vue`（须知→表单两步，dbType 切默认值，sqlite 显 dbPath、pgsql 显 template，全屏 loading，成功后 `resetSystemInitCheck()` 再 `router.replace({name:'login'})`）
  - 守卫 `router/guard/route.ts`：`initRoute` 中加会话级缓存 `ensureInitChecked()`，未初始化强制 `/init`、已初始化却停在 `/init` 则回 `root`，探测失败降级不阻塞；导出 `resetSystemInitCheck()` 供初始化成功后放行
  - 路由：`build/plugins/router.ts` 的 `constantRoutes` 加 `init` → gen 产出 `constant:true`；`routes.ts` 里 init 的 `component` 设为 `layout.blank` + `hideInMenu`（gen 会保留已存在路由的 layout/meta）
  - i18n：`locales/langs/{zh-cn,en-us}.ts` 加 `route.init` + `page.init.*`，`typings/app.d.ts` 的 `Schema.page` 补 `init` 类型
  - 登录页：移除冗余的"前往系统初始化"按钮 + 断导入（自动守卫已覆盖）

## 接口

- `POST /init/checkdb`（PublicGroup）→ `{code:"0000", data:{needInit:bool}, msg}`
- `POST /init/initdb`（PublicGroup）→ body `request.InitDB`，成功 `{code:"0000"}`

## 注意

- `sa gen-route` 会弹交互式"新增路由"向导（非交互 stdin 自动取消，不产生脏路由）；`/init` 的 blank 布局是手改 `routes.ts` 后被 gen 保留，属该插件的单路由布局唯一正规入口。
- 后端 `auto_code_llm.go` 复用 `commonResp.Response` 解析上游大模型响应，Code 改 string 后若上游返回数字 code 会解析失败——此为既有耦合，使用该 LLM 能力前需确认上游约定。
