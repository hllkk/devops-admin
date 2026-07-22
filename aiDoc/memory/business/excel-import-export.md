# 导入导出接口完善（按模块专用 handler + excelize）

> 2026-07-21｜承接 [[system-log-read-api]]「导出延后」、各模块 management 的导出按钮缺后端

## 需求
前端各列表「导出」按钮（用户/角色/岗位/字典类型/字典数据/登录日志/操作日志）与用户「导入」弹窗的 UI + 调用代码早已接好，但后端**完全缺失**：无 Excel 库、无 model、无 service、无 handler、无路由，`InitSysExportTemplateRouter` 仅注释占位（函数不存在），点击全部 404。需补齐后端导入导出。

## 方案选型（借鉴 gva 3.0-beta，去其过度设计）
- **路线对比**：gva 3.0-beta 用 `sys_export_template` 动态导出模板系统（模板表 + JOIN + Conditions + 自定义 SQL + token 下载 + 管理页）；但 devops 前端是**按模块的专用接口**调用（`/system/user/export` 等），动态模板对固定模块属过度设计
- 选「**按模块专用 handler**」：每模块 export/import 走现有 Router→API→Service 分层，前端调用方式零改动（贴合 AGENT.MD「沿用现有模式/不过度设计」）
- **借鉴 gva 的点**：① excelize 库（`xuri/excelize/v2`）；② 类型化 `SetCellValue`（数字存数字、`time.Time` 格式化）；③ 导入表头↔字段映射解析；④ 导出空模板
- **不借鉴**：动态模板表/JOIN/自定义 SQL（过度设计）、一次性 token 下载（devops 前端 POST + blob 直下载，不需）

## 实现
- 新增 `server/utils/excel/excel.go`：通用 `Export(rows, headers, sheet)`（反射读 struct 字段，可命中嵌入基座 `OPS_AUDIT_MODEL` 的提升字段如 `CreatedAt`；数字/time.Time/指针 nil 处理）、`ExportTemplate`（只表头）、`Parse(file, headers)`（读 xlsx **第一个工作表** → 按表头标题映射成 `[]map[string]string`）；Export/ExportTemplate 用 `SetSheetName` 重命名默认 sheet 保持**单 sheet**——曾因 `excelize.NewFile()` 默认带空 `Sheet1` + 又 `NewSheet(业务名)` 产生两 sheet、而 Parse 写死读 `Sheet1`，致导入报「Excel 无有效数据」（读到空 sheet），已修正为单 sheet + 读第一个工作表
- 新增 `server/api/v1/system/sys_export.go`：`writeXlsx`（Content-Disposition + Download-Filename + success 头，适配前端 `useDownload`）+ 全模块列定义（userHeaders / userImportHeaders / roleHeaders / postHeaders / dictTypeHeaders / dictDataHeaders / loginLogHeaders / operLogHeaders）
- **9 接口**：
  - `/system/user/export`（复用 list 条件去分页）、`/system/user/importTemplate`、`/system/user/importData`（表单字段 `updateSupport`；导入用 `Unscoped` 查询三分支——**活用户** updateSupport 更新/跳过、**软删除用户复活**(清 `deleted_at`+覆盖字段+重置密码,计入 update)、都没命中新建；默认密码取 `SysGeneralConfig.DefaultPassword`(常规配置可配,空回退 `User@1234`)，`PasswordUpdatedAt` 留空触发 `MustChangePwdGuard` 强制首登改密）
  - `/system/{role,post}/export`、`/system/dict/{type,data}/export`、`/log/{loginlog,operlog}/export`
- 各 service 加 `ExportList`（复用 `GetList` 的 where，去分页，加 `ExportMaxRows=10000` 上限防过大）
- 各 router 注册 `POST export`（+ user 的 importTemplate/importData）挂 PrivateGroup，操作日志中间件按 URL 自动分类 bizExport/bizImport（基础设施早已就绪，无需改）
- operlog 导出时间范围：`params[beginTime/endTime]` 经表单体传输，handler 用 `c.PostForm("params[beginTime]")` 显式取（与 GET list 的 `c.Query` 同构，前端 `transformToURLSearchParams` 对嵌套对象序列化为 bracket 形式）

## 前端配套修复（对齐 cookie 鉴权模式）
- `hooks/business/download.ts`：fetch 加 `credentials:'include'`、去掉 `Authorization: Bearer`（cookie 模式 token 走 httpOnly cookie，原代码发 Authorization 且无 credentials → 后端 `GetToken` 取不到 → 401）
- `user-import-modal.vue`：`NUpload` 加 `:with-credentials="true"`、去掉 Authorization
- operlog/loginlog 导出路径 `/monitor/*`→`/log/*`（对齐后端 `log/operlog`、`log/loginlog` 路由前缀；权限码 `monitor:operlog:export` 等保留不变，与路由独立）

## 导入 CORS 根因与修复（DEV）
- **现象**：导入用户报 CORS 错误（导出正常）。
- **根因**：`user-import-modal.vue` 写死 `getServiceBaseURL(env, false)` → isProxy=false → DEV 直连 `http://localhost:8888`（跨域）；后端 CORS 未启用（`config.yaml` `cors.mode=""` + `router.go:79-80` 注释）→ 浏览器拦截。导出走动态 isProxy（`/proxy-default` 同源）所以无事。**只有导入 CORS、导出没事**正印证根因。
- **修复**：import-modal 的 isProxy 改为动态 `import.meta.env.DEV && import.meta.env.VITE_HTTP_PROXY === 'Y'`（与 `useDownload` 一致）→ DEV 走 `/proxy-default` 同源，无 CORS。
- **生产不会出现**：cookie `SameSite=Lax`（`utils/claims.go:25,35`）决定跨站 XHR 不发 `x-token` → **部署必须同源**（Nginx 反代），故生产同源天然无 CORS；后端 CORS **无需启用**（同源不需要，跨域 cookie 也不发，启用是过度设计）。部署提醒：生产 docker 必须用 Nginx/同类反代使前后端同源，否则 cookie 鉴权整体失效（不只导入）。

## 验证
- 后端 `go build ./...`、`go vet` 通过
- 前端 eslint 改动文件 0 errors
- vue-tsc 全量被**预存**的 `src/typings/components.d.ts:183`（`const 'IconAntDesign:dingtalkCircleFilled':...` 字符串字面量作变量名，cbd158a 引入 dingtalk 图标时自动生成 bug）阻塞，**与本次无关**

## 待办 / 可选（当前不做）
- 导入用户部门支持按名称反查（当前填数字 ID）
- 其他模块导入（当前前端只有用户导入弹窗）
- 字典值字段（sex/status/businessType）导出转中文（当前导出原始码值，与 gva 一致）

## 关联
- [[system-log-read-api]]（导出延后）、[[httponly-cookie-auth]]（cookie 鉴权对齐）、各模块 management
