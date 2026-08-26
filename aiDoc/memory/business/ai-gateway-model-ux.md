# AI 网关·新建模型操作逻辑优化（对标 AIHelms 交互）

> 需求日期：2026-08-26。关联：[[ai-gateway-overview]]、[[ai-gateway-p1-progress]]

## 背景

对比 devops-admin 与本地 AIHelms（`/home/remote/AIHelms`）的「新建模型」前端操作逻辑，结论：AIHelms 更友好（免填技术性模型 ID、能力标签预设点选、Logo 图标网格）；devops-admin 在表单校验/抽屉布局/i18n 上更规范。按优势互补吸收 AIHelms 四点，本次纯前端改动（后端 CreateModel 早已支持 modelKey 可空，`server/service/gateway/model.go`「先建壳后部署时设置」）。

## 已实现（typecheck 通过）

1. **modelKey 改选填**：`model-operate-drawer.vue` 去掉必填规则，placeholder 提示「选填，留空可在新增部署时设置」。
2. **壳模型部署闭环**：`deployment-operate-drawer.vue` 的 props 从 `modelId` 改为整个 `model`；新增部署且关联模型 modelKey 为空时，表单顶部 NAlert + 路由名输入，提交时先 `fetchUpdateModel` 回写路由名再创建部署（绕开后端「请先为模型设置路由名」硬拒绝，正好是该后端注释的设计意图）。
3. **能力标签预设**：`constants/business/gateway.ts` 新增 `MODEL_CAPABILITY_PRESETS`（chat: 图像/推理/工具调用/长上下文；embedding: 多语言/多模态/代码/长文本；rerank: 多语言/多模态；值=展示文本即存储值，与 PROVIDER_TYPE_OPTIONS 同风格不走 i18n）；表单 options 随类别联动，切换类别经 `@update:value` 清空（非 watch，避免编辑回填时误清）。
4. **Logo 下拉渲染品牌图标**：drawer 切 `lang="tsx"`，NSelect `render-label` 用 `<SvgIcon :local-icon="getProviderIcon(...)">` + 文本；tag 自定义类型兜底 custom 图标。
5. **创建后自动选中**：`submitted` 事件带出 modelId，`index.vue` 刷新列表后选中该行并 `scrollIntoView`（列表项加 `data-model-id`）；编辑提交同样保持选中。
6. **空路由名兜底展示**：列表项与详情面板显示「未设置路由名」（详情面板用 NTag type=warning）。

## i18n 三处同步（zh-cn / en-us / app.d.ts）

- `page.gateway.model.form.modelKey.placeholder`（替换原 `.required`，key 已删）
- `page.gateway.model.modelKeyUnset`
- `page.gateway.deployment.form.modelKey.{required,tip}`

## 未做（后续可选）

- Logo 图标网格选择器（AIHelms 的 5 列网格弹层）——当前 render-label 图标下拉已够用，未引入弹层
- 新建模型时一步关联凭证自动建部署（AIHelms 也是僵尸半成品，未做）
