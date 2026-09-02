# AI 网关·模型发布（含用户级可见档）

- 日期：2026-08-27
- 状态：已实现（go build/vet + vue-tsc + eslint 通过）
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
- **用户级可见档(user)**：对齐 AIHelms `model_user_visibility` 蓝本。新增 `gateway.ModelVisibilityUser`（`gateway_model_visibility_user`，唯一索引 model_id+user_id，物理删插同部门投影表口径）；`VisibilityTypeUser` 常量；请求/视图加 `UserIds`；`PublishModel` 校验「user+发布时 userIds 必填 + 存在性校验(sys_users)」，事务内重建用户可见行；`DeleteModels` 级联清空。
- **请求 ID 列表必须用 `common.Int64StringSlice`**（踩坑：`[]int64` 遇前端 NSelect 用户 id 为 string 时报 `json: cannot unmarshal string into Go struct field ... of type int64`；用户下拉 value 来自 `fetchGetUserSelect` 的 IdType=string，而部门树 id 是 number 所以之前没炸）。发布参数 DepartmentIds/UserIds 均已改 Int64StringSlice（元素级 string/number 兼容，先例 ai_key 批量开通）。
- **SysUser 的 DB 列陷阱**（踩坑：`column "user_id" does not exist`）：`system.SysUser` 主键 gorm `column:id` 复用 id 列，手写 SQL 校验存在性要用 `id IN ?` 而非 `user_id IN ?`（部门侧 SysDepartment 是正常 `dept_id`，故 selected 档没炸）。
- **自动授权收窄为仅 all 档**：`PublishModel` 的 `syncPublicModelToMainKeys` 与 `publicModelKeys`（新主 Key 默认授权 + 自愈差集来源）都加 `visibility_type = all` 条件——定向发布（selected/user）不应全量授权到主 Key，语义与可见性投影一致。这是行为变更：此前 selected 档发布也会自动授权。
- **改名**：「发布配置」→「模型发布」（i18n title 一处，按钮/弹窗标题共用该 key）。

## 前端(第二轮：user 档 + 改名)

- `ModelPublishView/Params` 的 visibilityType 扩为 `'all' | 'selected' | 'user'`，加 `userIds: CommonType.IdType[]`。
- 弹窗 radio 加「指定用户可见」；user 档下 NSelect multiple+filterable 用户多选（`fetchGetUserSelect` 全量一次加载缓存，label `${nickName} ( ${userName} )`，先例 ai-key-batch-modal）；校验 user+发布时必选至少一个用户；切换非 user 档清空 userIds。
- i18n 新增 `visibilityUser/userIds/userRequired`，`autoGrantTip` 补「全员可见」限定，title 改「模型发布」(en: Model Publish)。

## 关联

- 术语与容器决策来源：[[ai-gateway-deployment-dialog-terminology]]
- 自动授权级联边界：[[ai-gateway-model-rename-cascade]]
- 第四档 mixed（部门+用户混合可见）：[[ai-gateway-publish-mixed-visibility]]
