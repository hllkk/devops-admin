# AI 网关 · Skill 管理(P2 收尾)

日期：2026-08-28
状态：已实现(后端+前端，待用户运行时验证)
反向链接：[[ai-gateway-mcp-server]]、[[ai-gateway-resource-application]]、[[ai-gateway-overview]]
（菜单结构后续已变更：本页挂入「AI能力」目录，见 [[ai-gateway-ai-capability-menu]]）

## 需求

P2「AI 市场」最后一块：Skill(企业内 AI 技能包)统一注册/zip 上传/发布/授权/分发/使用日志，
打通 `resource_type=skill` 审批公共底座与 `gateway_ai_key.skills` P1 预留字段。
与 MCP 同套模式(发布三档可见性+需审批+主 Key 授权双向对齐+广场)。
参照 AIHelms(`/home/remote/AIHelms` apps/api/v1/skills.py + services/skill_service.py)。

## 关键设计决策

- **分发方式=本地 zip 上传分发**(用户决策，AIHelms 同款；设计文档原写"下载地址"已作废)：
  管理端 multipart 上传 zip → 服务端存 `uploads/skills/{skillId}_{时间戳}.zip` →
  用户侧经平台端点鉴权下载。**该目录独立于 StaticFS 匿名公开的 `uploads/file`**
  (router.go:78 StaticFS 把 uploads/file 整目录公开)，防匿名直连绕过审批
  (同 [[rustfs-object-storage]] 的匿名下载教训)。prod 挂卷覆盖 /app/uploads 持久化天然兼容。
- **Skill 是平台自有资源，不经 LiteLLM**：无路由投影/无远端同步(与 MCP 的本质差异)，
  授权锚点=AiKey.skills JSONB 存 **skill ID 字符串数组**(models=modelKey/mcps=serverName
  都是业务键字符串，skill 无业务键；ID 不可变故无改名级联；前端 IdType 字符串闭环)。
  **改 skills 只更新本地 JSONB，不调 syncKeyToLitellm**——skills 不在 LiteLLM Key 请求体里。
- 上传与元数据 CRUD 解耦：`POST /gateway/skill/:id/package`(multipart ≤100MB)；
  发布前置校验 zip 已上传。发布中的 Skill 允许换包(版本升级，不动授权)。
- 下载 `GET /gateway/skill/download/:id`(**静态段在前**，casbin 白名单前缀匹配需要；
  `:id/download` 形态前缀不命中)：校验启用+已发布+有包；requiresApproval 须主 Key skills
  含锚点(loadMainKey 顺带自愈)；install_count+1(原子 UPDATE)+usage log 尽力而为不阻塞下载。
- 下载触发前端 blob(`request<Blob,'blob'>`，transform 对 blob 原样返回；错误 JSON 响应
  按 blob.type=application/json 解析 msg 抛出)。

## 后端要点(全部与 MCP 同构，差异处标注)

- 四表：`gateway_skill`(name 唯一禁重/version/author/category/tags JSONB/agent_install_prompt/
  usage_instructions/zip 三字段/install_count/isActive/isPublished/visibilityType/requiresApproval)
  + `gateway_skill_visibility`/`_user`(三档投影，物理删重建) + `gateway_skill_usage_log`
  (OPS_BASE+自增 id，时间用基座 createTime)。注册两处：initialize/gorm.go RegisterTables
  + source/gateway MigrateTable(**顺手补了 MCP 上次漏注册的 MCPTool/两投影表**——上次只注册
  了 MCPServer 主表，/initdb 路径会漏建 3 表)。
- `service/gateway/skill.go`：CRUD/发布对齐 alignSkillAuthorization(syncSkillToMainKeys/
  revokeSkillFromMainKeys **无 LiteLLM 推送**)/启停翻转联动(停用全量回收/恢复按档重授)/
  删除(全量回收+物理删投影+软删主行+尽力删 zip；usage log 保留审计)/usage 分页
  (userName/skillName 两次 IN 回填)。纯函数层 `skill_payload.go`(6 单测)：
  NormalizeSkillVersion(空→1.0.0,`\d{1,5}(\.\d{1,5}){0,2}`)/CleanSkillTags/Marshal|Unmarshal/
  SkillZipFilename/ValidSkillUploadFilename(.zip+防穿越)/SkillIdentityOf。
- AiKey 闭环：AiKeyOperateParams.Skills(nil=不改空=清空)；CreateSceneKey 主 Key 默认授权
  可见免审批 Skill+已批准申请(与 models/mcps 三口径同源)；UpdateAiKey 支持 skills；
  loadMainKey 自愈补 skills 差集；MyIdentityView 加 skills/availableSkills。
- 审批接入：ApplicationResourceSkill 常量早已预留；validateResourceVisible 三档校验/review
  通过分派 syncSkillToMainKeys/审批侧校验仍启用+已发布/fillApplicationViews 回填。
- 菜单：sys_menu 顶层 C `route.skill`(OrderNum 14,ApiPrefix /gateway/skill, /gateway/skill/*)；
  casbin 登录白名单加 `/gateway/skill/active`、`/gateway/skill/download`。

## 前端要点

- 管理页 `views/_gateway/skill/`(index+search+operate-drawer[元数据+zip NUpload
  custom-request，编辑态才可传]+publish-dialog[克隆 MCP]+usage-drawer 分页日志)。
- ai-key 抽屉加 Skill 授权多选(value=skillId 字符串)；密钥列表「资源」列升级
  模型/MCP/Skill 三计数(非零项)。
- 审批页 resourceType 列/筛选项支持 skill。
- home 身份页 Skill 区切真实(可见+授权态 tag)；广场面板 模型/MCP/Skill 三选单选，
  Skill 卡片(版本/作者/标签/下载次数)+下载按钮(blob 触发保存,未传包置灰,需审批未授权
  前置拦截提示申请)+申请弹窗支持 skill；appliedKeys `${type}:${id}` 防三类资源撞车。
- i18n 三处同步：route.skill、page.gateway.skill.*、page.home.square.skill 相关、
  page.gateway.aiKey.{col.skills,form.skillsPlaceholder}、application.typeSkill。
- AiKey.skills typings 修正 string[](P1 误标 number[]，上次只修了 mcps 漏了 skills)。

## 验证

- go build/vet/test 全过(skill_payload 6 单测+路由冒烟 skill_test.go+批量绑定测试)。
- pnpm typecheck/eslint 0 错误；elegant 路由四件由运行中的 vite dev 自动生成。
- 已有库补丁：`deploy/patches/2026-08-28-gateway-skill-mcp-menus.sql`
  (route.mcp+route.skill 菜单+super/admin 授权+casbin 幂等；MCP 上次未写补丁一并覆盖)。

## 待办(关联)

- 用户运行时点触验证(已有库先执行补丁或重置菜单种子)。
- swag 注释已写 @Tags GatewaySkill，待 swag 重生成(与 MCP 一并欠着)。
- ~~Agent 经 AI Key 认证直连下载端点(AIHelms `/{id}/zip` 同款)未做~~ → 已实现(2026-09-05)：
  `GET /gateway/skill/agent/{id}/zip` 挂 PublicGroup(AiKey Bearer/?token 自鉴权)；
  AiKey 加 key_hash(sha256 明文索引，syncKeyToLitellm 同事务写入+启动期 BackfillAiKeyHashes
  幂等回填存量)；授权锚点=当前 Key 自己的 skills(与登录态校验主 Key 区分)；usage log 加
  ai_key_id 归因+action=agent_download；登录态 `GET /gateway/skill/install-info/{id}` 下发
  服务端拼好的 curl 命令(主 Key 明文，前端复制不回显)；广场 Skill 卡片加「Agent 接入」弹窗。
- Skill resync/巡检兜底未做(纯平台资源无远端投影，漂移面小，暂不建)。
