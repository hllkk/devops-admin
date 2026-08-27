# AI 网关·模型发布配置前端落地（打通既有后端发布 API）

- 日期：2026-08-27
- 状态：已实现（vue-tsc typecheck + eslint 通过；后端零改动）
- 反向链接：[[ai-gateway-overview]]、[[ai-gateway-deployment-dialog-terminology]]、[[ai-gateway-model-rename-cascade]]

## 需求

模型录入与用户绑 Key 已就绪后的下一环：模型发布。后端此前已随模型 CRUD 全量落地（`GET /gateway/model/publish/:id` → `ModelPublishView`、`PUT /gateway/model/publish` ← `ModelPublishParams`、`gateway_model_visibility` 投影表、发布公开模型自动授权 `syncPublicModelToMainKeys`），前端完全缺入口——按既定设计决策「发布是模型级属性（多渠道共享），入口放模型详情面板头部，不进部署表单」补齐纯前端闭环。

## 已实现（纯前端）

1. **契约**：`gateway.api.d.ts` 新增 `ModelPublishView`/`ModelPublishParams`（departmentIds 为 IdType[]，后端 `[]int64` 无 `,string` 序列化为数字，与部门树数字 id 天然对齐）。
2. **service**：`model.ts` 新增 `fetchGetModelPublish`/`fetchPublishModel`。
3. **model-publish-dialog.vue**（新组件，居中模态框 w-560px 跟随 credential/deployment dialog 先例）：
   - 打开时 `GET publish/:id` 拉权威设置（可见部门投影行只在视图返回，本地行字段只作兜底回显）；部门树懒加载一次缓存复用
   - 表单：是否发布 NSwitch（旁挂自动授权提示文案）+ 可见范围 NRadio（all/selected，切换非 selected 清空已选部门）+ selected 时 NTreeSelect 多选可勾选（复用 `/system/user/deptTree`，key-field=id/label-field=label，先例 dept-tree-select）+ 订阅需审批 NSwitch
   - 校验：发布 + selected 且部门为空 → 前端 validator 报错（与后端「指定部门可见时必须选择至少一个部门」同口径）；后端另有部门存在性校验兜底
   - 顶部 NAlert：发布 + 免审批 + 模型未设模型 ID 时提示「发布后不会自动授权到主 Key」（对齐后端 `m.ModelKey != ""` 的自动授权条件）
4. **model-detail-panel.vue**：`#header-extra` 加「发布配置」按钮（material-symbols:publish 图标）；提交成功 emit changed → index.vue 重拉列表，列表徽标与面板头部发布/激活标签同步刷新（selectedModel 按 modelId 重找）。
5. **i18n**：`page.gateway.model.publish.*` 10 键三处同步（zh-cn/en-us/app.d.ts）。

## 设计决策记录

- **回填以 GET publish/:id 为准**而非列表行：可见部门投影行只在视图中返回，列表 Model 行没有该信息。
- **部门树 NTreeSelect 直用**而非复用 `dept-tree-select` 组件：该封装 value 模型是单选 IdType，多选勾选语义下直接用 NTreeSelect（multiple+checkable+filterable，默认 cascade 级联勾选父带子）。
- **取消发布不回收已授权 Key**：沿用既有边界（loadMainKey 自愈按 publicModelKeys 差集不会加回），弹窗不做取消发布的二次确认——发布设置可随时改回。
- **不改后端**：`PublishModel` 的事务（更新三字段 + 物理删插投影行 + syncPublicModelToMainKeys 尽力而为）已满足前端交互。

## 关联

- 术语与容器决策来源：[[ai-gateway-deployment-dialog-terminology]]
- 自动授权级联边界：[[ai-gateway-model-rename-cascade]]
