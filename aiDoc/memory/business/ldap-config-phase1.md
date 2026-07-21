# LDAP 配置存取落地（阶段一完成，登录链路待实现）

> 2026-07-21 · 前后端 · 新功能 · 9 文件 +124/-8

## 阶段一（已完成）：配置存取闭环

### 改动文件

**前端（4 文件）**：

- **新建** `web/src/views/_admin/system/setting/modules/ldap-setting.vue`：12 字段 3 tab（连接配置/属性映射/用户策略），风格对齐 security-setting.vue，走 i18n
- **新建类型** `web/src/typings/api/system.api.d.ts`：`Api.System.LdapSettingConfig`（12 字段），`Setting` 聚合类型新增 `ldap`
- **修改** `web/src/views/_admin/system/setting/index.vue`：import LdapSetting + LDAP_DEFAULTS + config.ldap 读写 + 模板双布局接入
- **修改** `web/src/locales/langs/zh-cn.ts` + `en-us.ts`：新增 25 条 LDAP 表单 i18n key

**后端（5 文件）**：

- **新建** `server/model/system/sys_ldap_config.go`：`SysLdapConfig` 单行表（id=1）+ `DefaultLdapConfig()` 工厂
- **新建** `server/service/system/sys_ldap_config.go`：`LdapConfigService` 含 `Get/Set/Current/LoadAll` + `atomic.Value` 内存缓存，复制 SecurityConfigService 模式
- **修改** `server/model/system/request/sys_setting.go`：`SettingConfig` 新增 `Ldap *SysLdapConfig`
- **修改** `server/service/system/sys_setting.go`：`Get`/`Set` 聚合读写补充 ldap 段落
- **修改** `server/service/system/enter.go` + `api/v1/system/enter.go` + `core/server.go`：注册 + 启动加载

### 字段清单

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| enabled | bool | false | LDAP 总开关 |
| host | string | localhost | 服务器地址 |
| port | int | 389 | 端口 |
| useSSL | bool | false | 是否 LDAPS |
| bindDN | string | "" | 管理员绑定 DN |
| bindPass | string | "" | 管理员密码 |
| baseDN | string | "" | 搜索 Base DN |
| filter | string | (uid=%s) | 用户过滤器，%s 替换登录名 |
| attrUsername | string | uid | 用户名属性 |
| attrNickname | string | cn | 昵称属性 |
| attrEmail | string | mail | 邮箱属性 |
| autoCreate | bool | false | LDAP 通过后自动创建本地用户 |

### 相比参考项目（/home/remote/devops-admin/frontend）的精简

| 删除的字段 | 原因 |
|-----------|------|
| syncEnabled / syncStrategy / conflictStrategy / syncDefaultEnabled | 参考项目纯占位桩，后端从未实现 |
| searchPageSize | 非核心认证字段 |
| server（自由文本 URL） | 拆为结构化 host + port |
| fieldMapping（JSON 文本域） | 拆为独立 attr* 字段 |
| baseOU | 改用标准 LDAP 术语 baseDN |

---

## 阶段二（待实现）：登录链路 LDAP 认证

### 需要做的事

1. **引入依赖**：`go get github.com/go-ldap/ldap/v3`
2. **新建** `server/utils/ldap.go`：LDAP 搜索+绑定认证工具函数
   - 连接 LDAP 服务器（支持 LDAP/LDAPS，超时 3-5s）
   - 用 bindDN/bindPass 管理员绑定
   - 用 filter 搜索用户 DN（`%s` 替换为登录用户名）
   - 用搜到的 DN + 用户密码二次绑定验证
   - 返回用户属性（uid/cn/mail）供自动创建
3. **改造** `server/api/v1/system/sys_user.go` 的 `Login` 方法：
   - 读 `ldapConfigService.Current(ctx)` 判断 enabled
   - 本地用户不存在且 LDAP 开启 → 走 LDAP 认证
   - 本地密码不匹配且 LDAP 开启 → 尝试 LDAP（混合模式：本地优先，LDAP fallback）
   - LDAP 通过 + autoCreate=true + 本地无用户 → 自动创建 `SysUser`（需分配默认角色）
4. **前端测试连接按钮** `handleTestConnection` → 调后端 `POST /system/setting/ldap/test` 接口

### 关键注意事项

- `bindPass` 存储时应加密（考虑用 `utils.AesEncrypt`），不能明文落库
- LDAP 连接超时 3-5 秒，防止 LDAP 不可用时卡死登录
- 自动创建用户时需分配默认角色
