# AI 网关·术语改名「模型 ID」+ 部署表单模态化重构

> 需求日期：2026-08-26。关联：[[ai-gateway-model-ux]]（同属新建模型链路优化）、[[ai-gateway-overview]]

## 背景

用户指出「路由名」叫法会与 P2/P3 规划的**路由配置**功能（策略/失败摘除/冷却/Fallback，即 AIHelms RouterSettings）产生术语冲突，采纳 AIHelms 的「模型 ID（用户请求时使用的标识）」；同时新增/编辑部署改为居中模态框 + 折叠分组（核心/计费与配额/路由配置/高级设置）。发布(visibility)确认为模型级属性，**不**进部署表单。

## 已实现（typecheck/lint 通过）

1. **术语改名（仅 i18n 文案与注释，后端字段 modelKey/API 契约不动）**：
   - `page.gateway.model.col.modelKey` 路由名→模型 ID；renameTip/modelKeyUnset/deployment.form.modelKey.* 同步改写
   - `gateway.api.d.ts`/`service/api/gateway/model.ts` 注释同步；`constants/module.ts`、`system.api.d.ts` 的前端路由(vue-router)概念"路由名"**不改**
2. **deployment-operate-drawer.vue → deployment-operate-dialog.vue**（跟随 credential-operate-dialog 命名与 NModal `preset="card" w-640px` 先例）：
   - NModal + NCollapse 三分组：计费与配额（`billingType !== 'token'` 才显示；月配额仅 `monthly_quota`）/ 路由配置 / 高级设置
   - 路由参数直绑 `litellmParams` 键：weight/order/timeout/stream_timeout/max_retries（NInputNumber）/tags（NSelect multiple tag）；高级：use_in_pass_through/drop_params（NSwitch）
   - 编辑回填 normalize（tags→[]、布尔开关→false）+ **已有值自动展开对应折叠组**（computeExpandedNames）
   - 提交：spread 保留掩码键仅覆盖 model + 剔除清空的可选路由键（`Reflect.deleteProperty`，eslint 禁动态 delete）
   - 后端零改动：路由参数经 litellmParams 透传（掩码 `MaskCredentialValues` 只掩凭证敏感值，weight 等明文回显）
3. 组件内表单 ref 命名 `formModel`（props.model 与 setup 变量同名会触发 vue/no-dupe-keys）
4. i18n 新增 deployment.group.{billing,routing,advanced} + col.{weight,order,timeout,streamTimeout,maxRetries,tags,useInPassThrough,dropParams} + form 提示键，三处同步

## 设计决策记录

- **容器**：用户拍板部署表单用模态框（居中聚焦），模型/供应商编辑仍为抽屉——网关模块内已有 credential modal 先例，不算开全站特例
- **「发布配置」不进部署表单**：发布是模型级属性（多部署共享），后续应在模型详情面板头部加入口（待做）
