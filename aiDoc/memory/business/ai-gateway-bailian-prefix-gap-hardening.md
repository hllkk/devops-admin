# AI 网关·百炼切换个人版踩坑与投影链路加固

- 日期：2026-10-10
- 状态：代码已实现(build/vet/test+eslint+typecheck 通过)，生产已手工止血恢复，待发版
- 反向链接：[[ai-gateway-p3-health]]（健康检查/巡检）[[ai-gateway-implementation-notes]]

## 事故复盘（生产 172.21.96.171）

百炼从团队版切个人版（退订团队坐席订阅、新建个人 Pro 版凭证、换绑全部百炼部署）后三类故障：

1. **连通性测试 all_deployments_in_cooldown**：根因=前端供应商选项 `bailian`（gateway.ts PROVIDER_TYPE_OPTIONS）在前缀差异表种子（provider_prefix.go providerPrefixSeeds）**没有对应行**。绑凭证部署推送时 `resolvePrefix` 未命中静默返空前缀 → openai 格式推**裸模型名**被 LiteLLM Router 每 30s upsert `LLM Provider NOT provided` Dropping（**管理面显示保存成功，实际路由表无此部署——静默失败**）；anthropic 格式走兜底前缀错误地转 Anthropic 协议。叠加：旧部署删除后 LiteLLM 侧残留**孤儿记录**（管理面 litellm_model_id 已不存在，本次 7bb324e3），引用的旧凭证已删 → `Missing Anthropic API Key` 401 进 cooldown，成为组内唯一可路由节点 → 巡检/测试永远打到它。
2. **坐席信息空**：个人版无坐席概念。seat-detail OpenAPI（AK/SK ACS3 签名）仍通但返回团队版遗留订阅实例（`sfm_tokenplanteams_dp_*`，AssignedStatus=UNASSIGNED、SeatId/AccountName 空）。
3. **套餐余量 0**：个人版 Token Plan 余量不在坐席 EquityList（返回 CREDITS 全 0），要走百炼 **console 鉴权口径**（参照 `bl usage token-plan`），当前 Go 实现只有团队版 AK/SK OpenAPI 通道——**功能缺口未实现**。

## 生产止血记录（2026-10-10 已执行）

- 补 `gateway_provider_prefix` bailian 三行（openai/chat、openai/embedding、anthropic/chat，与 dashscope 对齐）+ 删 LiteLLM 孤儿 7bb324e3 + 重启 devops-litellm + 前端凭证「停用→启用」触发 `syncCredentialRouting` 级联全量重推（ResyncDeployments 当时未接线）。
- 第二层问题：凭证 api_key 无效（sk-sp- 前缀对百炼所有端点 401）+ api_base 误填 `/apps/anthropic`（404）。**实证探测确认 token-plan 域名真实路径 = `https://token-plan.cn-beijing.maas.aliyuncs.com/compatible-mode/v1`**（openai 兼容口径；anthropic 口径路径未探测到可用值）。用户换有效 key+正确 base 后 glm-5.3 全链路恢复。

## 代码加固（本次落地，发版后存量库自愈）

1. **种子一致性防线**：`model/gateway.ProviderOptionTypes` 镜像前端 19 个预置选项 value；`source/gateway/provider_prefix_test.go` 单测断言每个类型有种子的 chat 行（首跑即抓到 ollama 缺 openai format 行的真实缺口，已补种子）。种子经 SeedProviderPrefix OnConflict DoNothing 在 RegisterTables 每次启动补插——发版即自愈，与手工 SQL 幂等不冲突。
2. **裸名推送拦截**：`resolveDeploymentPrefix` 返回 error——绑凭证但差异表未命中时推送侧直接报「供应商类型 X 未命中前缀差异表」，保存时暴露而非 LiteLLM 日志静默 Dropping；内联部署（cred==nil）不拦（model 自带 provider/ 前缀合法）。5 个调用点：Create/Update 事务内报错回滚，Resync/凭证级联/模型级联记 Failed/warning 继续。
3. **ResyncDeployments 接线 + 孤儿对账**：新增 `POST /gateway/model/deployment/resync`（同 api_prefix 分组免新增菜单授权）+ 前端部署面板行内 sync 按钮（抄凭证面板模式，i18n 三处同步）。尾部 `cleanupOrphanDeployments`：ListModels 的 model_info.id 对照管理面 **Unscoped 含软删行** 的 litellm_model_id 集合（删除=禁用留痕设计，软删行锚点必须保留），远端无主记录 DeleteModel；db_model=false 的 config 静态模型跳过。ResyncResult 加 orphanCleaned/orphanFailed 字段（凭证 resync 共用结构，零值无碍）。
4. **冷却原因透传**：`cooldownRootCause` 查回流侧 LiteLLM_SpendLogs 该 model_group 最近失败行，解析 `metadata.error_information`(JSON 字符串二次编码)的 traceback 末行（异常类+消息），跳过 RouterRateLimitError 冷却包装自身；`appendCooldownCause` 在 TestDeployment 与 HealthService.probeRoute 的 429 all_deployments_in_cooldown 响应上追加「冷却前最近一次真实上游错误」，页面直接看到 401 InvalidApiKey 不用翻容器日志。

## 遗留

- 个人版余量采集（console 口径）未实现，坐席面板对个人版应显示「无坐席」而非团队遗留空壳——待另立需求。
- 孤儿成因（删除部署时 LiteLLM 侧禁用/留痕为何漏改名）未深挖，resync 对账已可兜底。
- swag init 存量解析错误（credential.go CredentialView 找不到类型）与本次无关，swagger 文档待发版流程处理。
