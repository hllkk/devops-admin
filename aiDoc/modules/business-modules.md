# 业务模块 (business-modules)

> devops-admin 的业务按模块组织，每个模块一节，记录 model/接口/边界。随业务开发补充。
>
> **实现状态总览（2026-07-15 校准）**
> - **当前已落地主线：system 权限管理基座**——含用户/角色/菜单/部门/岗位/字典模型、初始化向导、httpOnly cookie 认证、go-captcha 验证码（详见下方「系统设置（已实现）」节与 `aiDoc/memory/business/`）。
> - **网盘模块：规划中、尚未启动**（见下方「网盘模块（规划中，未启动）」节）。前端 i18n 已预留 `disk` 等模块文案（`src/locales` 的 `module` 命名空间），后端无对应代码、`demand-index.md` 无相关记录。原需求骨架 `SoyDisk-Product-Spec.md` **当前不在仓库内**，启动前需重新确认去向。

## 网盘模块（规划中，未启动）

> 下方为企业内部网盘的**规划蓝图**，仅作设计参考，**尚未实现**。后端无 `server/` 对应代码，`demand-index.md` 无相关业务记录；启动前需先找回或重建 `SoyDisk-Product-Spec.md`。

企业内部网盘：管理员上传大型安装包/交付物，对外用「链接 + 提取码」匿名分享给客户下载，对内按账号/部门共享并支持 OnlyOffice 多人协同编辑；含版本历史、回收站、配额管控、企业微信扫码登录。存储用 RustFS，协同用 OnlyOffice。

### 文件管理
- 上传：分片 + 断点续传 + hash 秒传，单文件上限 5GB，落 RustFS
- 文件/文件夹 CRUD：新建/重命名/移动/删除（进回收站）
- 预览：txt/pdf/Office（OnlyOffice 渲染），统一新页面打开

### 对外分享
- 链接 + 提取码 + 有效期（永久/7天/1天/自定义）+ 可选下载次数限制
- 外部客户匿名下载，不算系统用户，权限档为"可下载"

### 内部共享
- 按账号/部门授权，无需提取码
- 权限档：只读 / 编辑（编辑 = 文件操作层 rename/move/delete/overwrite + OnlyOffice 内容层协同）

### 配额
- User 默认 100M（上传门槛）；管理员可调；公司总量上限 2T（兜底）
- 按文件实际大小计，**不含版本历史占用**；超额上传拦截并提示

### 版本历史
- 覆盖上传 → 旧版进历史；默认留 10 份；可一键回滚

### 回收站
- 删除进回收站；可恢复；过期清理（`server/task/`）

### OnlyOffice 协同
- 有编辑权限者多人实时编辑（光标/改动同步）；自动保存 + 版本记录
- 权限与文件操作层共用一个"编辑权限"开关

### 企业微信登录
- 扫码登录，无需独立账号密码；部门/员工组织架构自动同步
- 离职账号随企微停用，其共享权限同步失效

### 角色与配额管理
- 三级角色：超管 / 管理员 / User
- **角色·用户·部门管理复用 SoybeanAdmin 基座，不重复实现**
- 配额在基座用户管理中调整；总量上限在系统设置-网盘配置

### 公共资料库
- 一键覆盖全员可见可下载，管理员不必逐人共享

## 系统设置（已实现）

全局配置中心：键值对表 `sys_setting`（`name`=分类 + `value`=JSON 文本），按分类聚合读写，不随业务表扩张。登录页用公开接口读展示配置，管理员用 admin 接口读写完整配置。源实现搬运自 `main` 子模块（backend/frontend），已按项目规则重构。

### 存储与接口
- 表 `sys_setting`：雪花主键 + `OPS_MODEL` + `name`(唯一索引) + `value`(text) + `desc`，一个分类一行。
- 聚合 DTO `SystemSettings`：`general` / `security` / `authentication` / `ldap` / `notify` / `disk` 六类指针，未配置返回 nil。
- 接口（单数路径 `/system/setting/*`）：`GET /system/setting/public`（公开，脱敏子集）、`GET /system/setting`（admin 完整）、`PUT /system/setting`（admin 整体保存，事务内按分类 upsert）。统一 `{code,data,msg}`，code 字符串 `0000`/`0001`。
- 鉴权分层：public 组（无需登录，登录页用）/ admin 组（JWT + RequireAdmin）。
- 菜单与权限 seed：`source/system/sys_menu.go`（菜单 `system:setting:list` + 按钮 `system:setting:save` + 关联 Apis）；默认配置 seed：`source/system/sys_setting.go`（general + security），初始化向导写入。

### 配置分类
- **general** 通用：站点名称/描述/Logo/Favicon、默认用户密码/角色、验证码（类型/长度/过期/误差）、登录与操作日志保留天数。
- **security** 安全：密码策略（最小长度/大小写/数字/特殊字符）、登录失败锁定（次数/时长）、IP 黑白名单。
- **authentication** 认证（阶段二）：企业微信/微信/Gitee/GitHub 第三方登录；密钥类字段返回时脱敏。
- **ldap**（阶段二）：扁平结构——服务器/绑定账号/基础 OU/同步策略；master 暂无 LDAP 后端。
- **notify** 通知（阶段二）：邮件（SMTP），预留短信/飞书/Webhook。
- **disk** 网盘（阶段二）：上传限制/配额/分享密码/OnlyOffice/转码/解压缩——**网盘存储总量上限配置在此**（见网盘模块·配额），网盘模块落地后消费。

### 前端
- 页面 `_admin/system/setting/index.vue`：左侧分类菜单 + 右侧表单，整体加载、整体保存（用 `??` 兜底，保留合法的 0/false）。
- 当前进度：general + security 两个 Tab；auth/disk/ldap/notify 随对应模块落地再补（后端存储已就绪，只需加 module）。
- logo/favicon 当前为 URL 输入，上传组件待阶段二。

