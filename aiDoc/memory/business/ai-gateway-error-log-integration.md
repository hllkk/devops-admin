# AI 网关·错误日志接入补齐 + 操作日志归属决策

- 日期：2026-08-30
- 状态：已实现（go build/vet/fmt/test 全过）
- 反向链接：[[ai-gateway-overview]]、[[ai-gateway-p1-hardening]]

## 需求

排查 AI 网关接口报错时错误日志页面查无记录。分析确认两个缺口并决策操作日志归属，经用户确认后修改。

## 分析结论

- **错误日志机制**：zap core（`core/internal/zap_core.go` Write）全局捕捉 Error 级以上日志自动落 `sys_error` 表（含 request_id/trace_id/调用栈/源码定位），任何模块 `logger.WithCtx(ctx).Err(err).Error()` 即自动接入，零注册成本；查询入口后台管理→日志管理→错误日志。
- **网关接入现状（改前）**：service 层已接入（9 文件约 33 处 `Mod("gateway")`，覆盖解密失败/投影推送失败/用量回流失败等内部路径）；但 API 层 14 文件全部无 logger，且 service 大量 DB 错误路径（列表 Count/Find 失败等）直接 return err 不记日志——错误直接丢失。
- **操作日志归属决策（用户确认）**：**维持集成后台管理操作日志，不另建**。OperationRecord 中间件挂 PrivateGroup，网关管理路由全在其上，写操作已自动落 sys_oper_log（title 自动推导 `gateway/xxx`，与 system 模块 `system/xxx` 格式一致）。理由：统一审计/检索/清理、避免重复造轮子；AIHelms 另建网关管理日志是因它无统一操作日志基座，不照搬；网关调用/用量日志属 usage 回流体系不属操作日志。

## 实现

- **范围**：`api/v1/gateway/` 全部 service err 分支补 `logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("<操作>失败")`，插在 FailWithMessage 前一行；对齐 system 模块 API 层模式。
- **明确不加**：ShouldBindJSON/Query 绑定失败、ParseInt 等 ID 解析、FormFile 缺失等用户输入校验分支（对齐 system 惯例，避免噪音）。
- **改动**：11 文件 +60 处新增（mcp/skill 各 14、model 6、credential 5、ai_key/deployment/provider 各 4、dashboard/key_scenario 各 3、provider_balance 1）；resource_application.go 4 处 `Warn("xx失败: "+err.Error())` 升级为 `.Err(err).Error()` 风格（Warn 不落 sys_error，原写法达不到目标）；217 行旁路通知失败保持 Warn 合理。router_settings/usage 原已完整零改动。
- **验证**：go build/vet/fmt 全过、service/gateway 测试过；awk 扫描确认剩余无 logger 的 FailWithMessage(err.Error()) 分支全为参数绑定，service err 分支零遗漏。
