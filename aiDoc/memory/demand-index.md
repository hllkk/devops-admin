# 业务需求索引 (demand-index)

> 仅索引。每条业务需求的日期、标题、文件路径、状态。新增 business 记录时同步追加一行。

| 日期 | 需求 | 文件 | 状态 |
|---|---|---|---|
| 2026-07-11 | 借鉴 gin-vue-admin 实现系统初始化流程（checkdb→initdb，路由守卫自动跳转，后端响应码对齐 "0000"） | business/system-init-flow.md | 已实现 |
| 2026-07-11 | 后端引入雪花算法作为统一主键策略（自实现 + 字符串传输 + GORM Callback 集成） | business/snowflake-id-generator.md | 已实现 |
| 2026-07-12 | 清理基座多租户残留代码（前端 12 文件：类型/service/store/hook/登录与社交登录页面；后端无多租户代码） | business/remove-multi-tenant.md | 已实现 |
| 2026-07-12 | 后端 User/Role 模型建模（贴合前端 RuoYi 契约：SysUser/SysRole + sys_user_role/sys_role_menu + request DTO + AutoMigrate，方案 A；审计基座 OPS_AUDIT_MODEL 上移 global，OPS_MODEL 去 ID） | business/system-user-role-models.md | Model 已落地 |
| 2026-07-12 | 菜单管理：后端 SysMenu 模型 + request DTO + AutoMigrate 注册（对齐前端 Api.System.Menu：树形 parentId、目录/菜单/按钮、isFrame/isCache/visible/status、瞬态 Children/ParentName），并补齐 page.system.menu 全量中/英 i18n（53 key） | business/menu-management.md | Model+i18n 已落地 |
| 2026-07-13 | 部门与岗位管理：后端 SysDept/SysPost 模型 + request DTO + AutoMigrate 注册（对齐前端 Api.System.Dept/Post：树形 dept parentId+ancestors、分页 post、雪花主键字符串、省略 tenantId），补齐 page.system.dept 全量 i18n、新增 page.system.post 全量中/英文案并将 post 页面硬编码改 i18n；重生成 Elegant 路由（system_dept/post/menu/dict/notice）并补路由文案 | business/dept-post-management.md | Model+i18n+路由 已落地 |
| 2026-07-13 | 字典管理：前端补齐 i18n（新增 dictData key、补 common.selected/anyRecords/clear/noSelectRecord）、修正表头权限码 system:user:*→system:dict:*、修字典类型 placeholder key、导出文件名硬编码改 i18n；后端整体缺失（无 DictType/DictData 模型与接口、无菜单权限种子），用户决定延后单独规划 | business/dict-management.md | 前端 i18n 已落地，后端延后 |
| 2026-07-13 | 初始化向导加 Redis 步：须知→数据库→Redis→管理员密码 三步向导，DB/Redis 各带「测试连接」按钮（ephemeral ping 不建库不落盘），Redis 必填单实例三字段，末步一次性提交（落 config.Redis + use-redis:true + 即时连 OPS_REDIS）；新增 /init/db/ping、/init/redis/ping 端点 | business/init-wizard-redis.md | 已实现 |
| 2026-07-14 | 认证链路改 access+refresh 双 httpOnly cookie（对标目标主机 172.21.10.40）：SameSite=Strict + 动态 Secure + CORS(credentials) + refresh 轮换黑名单；鉴权失败改 HTTP200+业务 code（9999 刷新/8888 登出）；登录响应只回 {expiresAt}；删除 x-token/Authorization/localStorage token 等残留。仅基础项，不做防爆破/多点/ClientIP绑定/验证码/int64迁移 | business/httponly-cookie-auth.md | 设计已批准，待实现 |
| 2026-07-14 | 登录集成 go-captcha 行为验证码（click/slide/rotate 三类型 config 可切换，阈值触发，替换原 GIF）：config.captcha.go-captcha 段、sys_captcha service（Provider+单例+Redis/local_cache+触发计数）、GET /auth/captcha、Login 先于密码校验+一次性消费；前端 go-captcha-vue + NModal 按 type 渲染、store 透传 captchaId/captcha、i18n | business/go-captcha-login.md | 已实现 |
| 2026-07-15 | 字典管理后端 model：新增 SysDictType/SysDictData（sys_dict_type/sys_dict_data），对齐前端 Api.System.DictType/DictData（两表均无 status、dict_type 唯一键/索引、isDefault=Y/N、isI18n 预留、createDept 不建），基座 OPS_AUDIT_MODEL + 雪花主键，已注册 RegisterTables | business/dict-management.md | Model 已落地 |
