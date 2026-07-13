# 权限基座闭环设计 (Permission Base Closed-Loop)

- **状态**：已批准（2026-07-13）
- **范围**：系统初始化自动写入数据 + casbin/RBAC 双轨权限的最小可用闭环
- **关联**：本 spec 是「权限体系」系列的第一个子项目；业务模块 CRUD、动态菜单、casbin 管理 UI 等为后续独立 spec

---

## 1. 背景与现状

### 1.1 项目架构
- 后端：Go + Gin + GORM，基于 gin-vue-admin (GVA) 范式魔改，module 名 `github.com/hllkk/devops-admin/server`，配置/全局前缀改为 `OPS_`
- 前端：SoybeanAdmin 2.x（Vue3 + Vite + TS + NaiveUI + Pinia + Elegant Router）

### 1.2 权限相关现状（探索结论）

| 组件 | 现状 |
|---|---|
| RBAC model（sys_user/role/menu/dept/post/user_role/role_menu） | ✅ 完整，字段对齐前端，雪花 ID，审计基座 `OPS_AUDIT_MODEL` |
| init 框架（`service/system/sys_init.go`） | ✅ 完整：`SubInitializer` 接口 + `RegisterInit(order)` 排序 + context 传依赖 + 幂等检查；但 `source/` 下 seed 全空（`api.go` 整文件注释），首初始化会报「无可用初始化过程」 |
| casbin enforcer（`utils/casbin.go`） | ⚠️ 单例 `SyncedCachedEnforcer`，gorm-adapter 落 `casbin_rule` 表，缓存 1h；ACL model（`keyMatch2` 路径通配，allow-list）；**matcher 未用 `g()`，实为 ACL 非 RBAC** |
| casbin 中间件（`middleware/casbin_rbac.go`） | ⚠️ sub=RoleId, obj=URL(去 `RouterPrefix`), act=Method；**无超管豁免**，超管会被空策略拦死 |
| casbin 策略写入 | ❌ 全仓零 `AddPolicy/RemovePolicy`，策略表恒空 |
| `sys_menu.perms` 字段 | ⚠️ 已存在（F 按钮权限标识），但后端无任何读取/校验逻辑 |
| `/auth/login`、`/auth/getUserData` 接口 | ❌ 后端完全不存在（前端已在调用） |
| `server/api/system/`、`server/router/system/`（业务部分） | ❌ 空 / 仅 init；无 user/role/menu/auth/casbin 的 service+api+router |
| 前端权限判断 | ✅ `useAuth()` → `hasAuth(code)` / `hasRole(code)`，数据来自后端 `getUserData` 下发的 `permissions[]` / `roles[]`，超管通配 `*:*:*`；路由当前为 static 模式 |

### 1.3 核心矛盾
casbin（API 级，RoleId×URL×Method）与 RBAC 按钮权限（UI 级，perm code）是两个维度，**不是二选一**：前者管后端 API 兜底安全，后者管前端按钮显隐。当前两者均未真正生效。

---

## 2. 目标与非目标

### 2.1 目标
完成「首次初始化 → 超管登录 → 前端拿到 permissions → casbin 兜底非超管」的完整闭环，作为权限体系的可运行基座。

### 2.2 非目标（后续子项目）
- ❌ user/role/menu/dept/post 的业务管理 CRUD 接口
- ❌ 前端动态菜单（当前 static 模式不变，路由仍由 Elegant Router 按 `src/views` 生成）
- ❌ casbin 策略的可视化管理 UI（策略由 init + 联动钩子自动维护）
- ❌ 网盘等业务模块
- ❌ `sys_api` 独立 API 资源表（本期用方式 B：菜单内嵌 apis，不引入该表）

---

## 3. 已确认的核心决策

| 决策 | 选择 | 说明 |
|---|---|---|
| 权限体系形态 | **两套并存联动（GVA 原版）** | casbin 兜底 API + RBAC 按钮权限驱动前端 UI，菜单/角色管理为单一真源双向联动 |
| API 资源载体 | **方式 B：菜单即 API 资源** | C 型菜单承载 API 资源生成 casbin 策略；F 型按钮挂 perms 服务前端 |
| C 菜单 API 存储形式 | **JSON 数组内嵌 `apis`** | `sys_menu.apis: [{path,method}]`，一个菜单挂多 API，不引入新表 |
| 本期范围 | **权限基座闭环** | 见 §2 |
| 超管豁免数据源 | **JWT claims 加 `SuperAdmin`** | 登录时由 `sys_role.super_admin` 写入，中间件零查库 |
| seed 菜单粒度 | **完整种 5 个系统管理菜单 + 各自按钮 perms** | 与前端 `_admin/system/{user,role,menu,dept,post}` 五页对齐，perms 齐全才能验 `hasAuth` |
| seed 角色 | **种 3 个：superadmin / admin / user** | roleKey 对齐前端 `hasRole` 硬编码（super/admin 放行）；对应用户 super/admin/test1 |
| 角色菜单范围 | **super 全 / admin 全 / user 无** | super 超管豁免挂全部；admin 管理员挂全部系统管理菜单；user 不挂（test1 验证「被限制」，等业务模块上线再授权） |
| seed 部门 | **种 3 级**：根部门 + 北京总部 + 天津工厂 | super/admin → 北京总部；test1 → 天津工厂 |
| seed 岗位 | **留空** | 不种岗位，用户也不关联岗位 |
| C 菜单 `apis` 路径 | **按 RESTful 约定预填** | 本期不被命中（超管豁免），为后续 CRUD 接入时 casbin 自动生效铺路 |

---

## 4. 数据模型变更

### 4.1 `sys_menu` 新增 `Apis` 字段

```go
// model/system/sys_menu.go
import "gorm.io/datatypes"

type SysMenu struct {
    // ...现有字段保持不变...
    Apis datatypes.JSON `gorm:"column:apis;type:json;comment:'C菜单挂载的API资源[{path,method}]'" json:"apis"`
    // F 按钮、M 目录不使用该字段（留空）
    // ...OPS_AUDIT_MODEL...
}
```

- **依赖**：`go.mod` 新增 `gorm.io/datatypes`（如尚未引入）
- **迁移**：字段加入后由现有 `RegisterTables()` 的 `AutoMigrate` 自动加列；无需手写迁移脚本
- **格式约定**：`[{"path":"/system/user/list","method":"GET"},{"path":"/system/user","method":"POST"}, ...]`
- **path 约定**：存**去掉 `RouterPrefix` 后**的路径，与 casbin 中间件计算的 `obj`（`strings.TrimPrefix(path, RouterPrefix)`）保持一致

### 4.2 `jwt.BaseClaims` 新增 `SuperAdmin`

```go
// model/system/request/jwt.go
type BaseClaims struct {
    UUID       uuid.UUID
    ID         uint
    Username   string
    NickName   string
    RoleId     uint
    SuperAdmin bool   // 新增
}
```

- **兼容性**：旧 token 该字段为零值 `false` → 视为非超管，需重新登录。init 后首登即获取含新字段 token，无运行期影响
- **写入点**：`/auth/login` 签发 token 时，按用户所属角色的 `sys_role.super_admin` 写入

> 备注：`BaseClaims.ID` / `RoleId` 当前仍为 GVA 基座的 `uint` 空壳（见 `aiDoc/frontend-backend/boundary.md` 主键契约的例外说明）。本期**不动这两者的类型**，仅新增 `SuperAdmin` 字段；uint↔int64 的统一留待用户/登录链路全面实现时处理。

---

## 5. init seed 设计（`source/` 新建）

### 5.1 initializer 清单与排序

每个 initializer 实现 `SubInitializer` 接口（`InitializerName / MigrateTable / InitializeData / TableCreated / DataInserted`），通过 `RegisterInit(order, i)` 注册，框架按 order 升序执行。依赖靠 order 大小保证（B=A+1 表示 B 在 A 后）。

| 文件 | initializer 名 | order | 依赖 | 说明 |
|---|---|---|---|---|
| `source/system/sys_dept.go` | `sys_dept` | `InitOrderSystem+1` (11) | 无 | 种 3 级部门：根部门 + 北京总部 + 天津工厂 |
| `source/system/sys_role.go` | `sys_role` | `+2` (12) | 无 | 种 3 角色：superadmin(超管) / admin / user |
| `source/system/sys_menu.go` | `sys_menu` | `+3` (13) | 无 | 种完整菜单树（含 F 按钮 perms + C 菜单 apis） |
| `source/system/sys_user.go` | `sys_user` | `+4` (14) | dept | 种 3 用户：super/admin/test1，各自 dept_id |
| `source/system/sys_user_role.go` | `sys_user_role` | `+5` (15) | role+user | super→superadmin, admin→admin, test1→user |
| `source/system/sys_role_menu.go` | `sys_role_menu` | `+6` (16) | role+menu | superadmin/admin ↔ 全部菜单；user 不挂 |
| `source/system/sys_casbin.go` | `sys_casbin` | `+7` (17) | role_menu+menu.apis | 遍历 3 角色调 `UpdateCasbin`（user 无菜单则跳过） |

> 岗位（`sys_post`）**不种** seed（表仍由 `RegisterTables` 自动建，留空）。用户-岗位关联本期不做。

### 5.2 依赖值传递

initializer 间通过 `context.WithValue` 传递已创建实体的 ID 映射，避免重复查库或硬编码 ID 误用：

- `sys_dept` 把 `{deptName: deptId}` 写入 ctx（如 `北京总部 → deptId`）
- `sys_role` 把 `{roleKey: roleId}` 写入 ctx（如 `superadmin → 1`）
- `sys_menu` 把 `{menuName/perms: menuId}` 写入 ctx（供 role_menu / casbin 取用）
- `sys_user` 把 `{username: userId}` 写入 ctx
- 下游 initializer 从 ctx 取依赖 ID 构建关联

### 5.3 seed 内容详述

**(a) 角色**（`sys_role`，3 个）

| RoleId | RoleName | RoleKey | SuperAdmin | RoleSort | 说明 |
|---|---|---|---|---|---|
| 1 | 超级管理员 | `superadmin` | true | 0 | 超管，casbin 豁免，前端 `hasRole` 放行 |
| 2 | 管理员 | `admin` | false | 1 | 系统管理员，挂全部系统管理菜单；前端 `hasRole` 放行 |
| 3 | 普通用户 | `user` | false | 2 | 不挂系统管理菜单，`permissions` 为空 |

（雪花回调不覆盖显式主键，RoleId 固定为 1/2/3；`Status='0'`）

**(b) 部门**（`sys_dept`，3 级）

| DeptId | DeptName | ParentId | Ancestors | 说明 |
|---|---|---|---|---|
| 1 | XXX科技 | 0 | 0 | 根部门（⚠ 名称为占位，部署前按实际公司名替换） |
| 2 | 北京总部 | 1 | 0,1 | super / admin 挂此部门 |
| 3 | 天津工厂 | 1 | 0,1 | test1 挂此部门 |

（`Ancestors` 为 RuoYi 惯例的祖先链逗号分隔；`Status='0'`）

**(c) 菜单树**（`sys_menu`，5 个系统管理菜单 + 按钮；C 菜单 apis 按约定预填）

| 菜单 | type | perms | apis（去 RouterPrefix） |
|---|---|---|---|
| 系统管理 | M | - | - |
| ├ 用户管理 | C | system:user:list | `[{GET /system/user/list},{POST /system/user},{PUT /system/user},{DELETE /system/user/:id}]` |
│ ├ 新增 | F | `system:user:add` | - |
│ ├ 修改 | F | `system:user:edit` | - |
│ ├ 删除 | F | `system:user:remove` | - |
│ ├ 导出 | F | `system:user:export` | - |
│ ├ 导入 | F | `system:user:import` | - |
│ └ 重置密码 | F | `system:user:resetPwd` | - |
| ├ 角色管理 | C | system:role:list | `[{GET /system/role/list},{POST /system/role},{PUT /system/role},{DELETE /system/role/:id}]` |
│ ├ 新增/修改/删除/导出 | F | `system:role:{add,edit,remove,export}` | - |
| ├ 菜单管理 | C | system:menu:list | `[{GET /system/menu/list},{POST /system/menu},{PUT /system/menu},{DELETE /system/menu/:id}]` |
│ ├ 新增/修改/删除 | F | `system:menu:{add,edit,remove}` | - |
| ├ 部门管理 | C | system:dept:list | `[{GET /system/dept/list},{POST /system/dept},{PUT /system/dept},{DELETE /system/dept/:id}]` |
│ ├ 新增/修改/删除 | F | `system:dept:{add,edit,remove}` | - |
| └ 岗位管理 | C | system:post:list | `[{GET /system/post/list},{POST /system/post},{PUT /system/post},{DELETE /system/post/:id}]` |
  ├ 新增/修改/删除/导出 | F | `system:post:{add,edit,remove,export}` | - |

> perms 清单源自前端 `_admin/system/*` 各页实际使用的 `hasAuth(...)` 字面量，确保下发的 `permissions[]` 覆盖前端全部按钮判断。

**(d) 用户**（`sys_user`，3 个）

| UserName | NickName | DeptId | RoleKey | Password |
|---|---|---|---|---|
| super | 超管 | 北京总部(2) | superadmin | bcrypt(初始密码) |
| admin | 管理员 | 北京总部(2) | admin | bcrypt(初始密码) |
| test1 | 测试用户 | 天津工厂(3) | user | bcrypt(初始密码) |

初始密码统一取自 `InitDB` 入口写入 ctx 的 `AdminPassword`（`sys_init.go:101`）；未提供则用安全默认值并强制日志告警。`Status='0'`。DeptId / RoleKey 经 ctx 映射取实际 ID。

**(e) 关联**
- `sys_user_role`：(super, 1) / (admin, 2) / (test1, 3)
- `sys_role_menu`：(1, 全部 menuId) / (2, 全部 menuId)；角色 3(user) 不挂

**(f) casbin 策略**（`sys_casbin` initializer）
- 遍历全部角色调 `UpdateCasbin(roleId)`：清旧策略 → 遍历其 C 菜单 apis → `AddPolicy(roleId, path, method)`
- 结果：角色 1(superadmin)、2(admin) 各生成一套系统管理 API 策略；角色 3(user) 无菜单 → 无策略
- superadmin 实际走 claims 豁免不依赖策略；生成策略是为保持联动机制统一，并为「将来取消豁免 / 细粒度收权」留一致性基础

---

## 6. auth 链路（新建 service / api / router）

新增文件：
- `service/system/sys_auth.go`
- `api/system/sys_auth.go`
- `router/system/sys_auth.go`

### 6.1 `POST /auth/login`（挂 PublicGroup，免鉴权）
1. 入参：`username` / `password`
2. 按 username 查 `sys_user`，校验 bcrypt 密码
3. 查 `sys_user_role` 拿该用户全部 RoleId，查 `sys_role`：
   - `SuperAdmin` = 任一角色 `super_admin=true` 即为 true
   - `BaseClaims.RoleId` 取主角色（本期每用户单角色即唯一角色；多角色场景的 casbin 适配见 §10）
4. 签发 JWT（`BaseClaims` 写入 `Username / ID / RoleId / SuperAdmin`），处理 token 黑名单/刷新沿用现有 `jwt_black_list` 与 JWT 工具
5. 返回 `{ token }`，响应结构遵循 `{ code:"0000", data, msg }`

### 6.2 `GET /auth/getUserData`（挂 PrivateGroup）
聚合返回，对齐前端 `Api.Auth.UserInfo`：
```json
{
  "code": "0000",
  "data": {
    "user": { /* SysUser + roles: Role[] */ },
    "roles": ["superadmin"],
    "permissions": ["*:*:*"]   // 超管；非超管为 F 按钮 perms 聚合
  }
}
```

聚合算法（见 §7）。

---

## 7. permissions / roles 聚合逻辑（getUserData 核心）

```
输入：当前登录用户 userId（来自 JWT claims）

roles[]:
  userId → sys_user_role → roleIds
        → sys_role → 收集 role_key（status='0'）

permissions[]:
  userId → sys_user_role → roleIds
        → 若任一角色 super_admin=true → 直接返回 ["*:*:*"]
        → 否则：
            roleIds → sys_role_menu → menuIds
                   → sys_menu where menuType in ('C','F') and status='0'
                   → 收集 perms 去重 → 返回
```

> 本期不校验菜单/角色的 `visible` 字段对 perms 的影响（visible 仅影响前端菜单显隐，不影响按钮权限聚合）。
>
> 说明：C 菜单（页面）的 perms（如 `system:user:list`）同样聚合进 `permissions[]`，供前端列表/菜单可见性判断；F 按钮的 perms 供按钮显隐。目录（M）通常无 perms，不参与聚合。

---

## 8. casbin 联动与超管豁免

### 8.1 联动钩子（新增 `service/system/sys_casbin.go`）

```go
// UpdateCasbin 重算指定角色的 casbin 策略
func UpdateCasbin(roleId int64) error {
    e := utils.GetCasbin()
    // 1. 清该角色旧策略（第 0 个字段=sub 的过滤删除）
    e.RemoveFilteredPolicy(0, strconv.FormatInt(roleId, 10))
    // 2. 查该角色关联的 C 型菜单
    menuIds := /* sys_role_menu -> menuIds */
    menus  := /* sys_menu where menuId in menuIds and menuType='C' */
    // 3. 遍历 apis 写新策略
    for _, m := range menus {
        for _, api := range parseApis(m.Apis) {
            e.AddPolicy(strconv.FormatInt(roleId,10), api.Path, api.Method)
        }
    }
    // 4. 失效缓存
    e.InvalidateCache()
    return nil
}
```

**调用点**：
- init 的 `sys_casbin` initializer（本期唯一调用点）
- 未来角色管理 CRUD 的「设置角色菜单」方法（后续子项目接入）

### 8.2 中间件超管豁免（改 `middleware/casbin_rbac.go`）

```go
func CasbinHandler() gin.HandlerFunc {
    return func(c *gin.Context) {
        waitUse, _ := utils.GetClaims(c)
        if waitUse.SuperAdmin {     // 新增：超管直接放行
            c.Next()
            return
        }
        // ...原 Enforce 逻辑保持不变...
    }
}
```

### 8.3 casbin v3 API 适配
- 本项目锁版本：`casbin/v3 v3.10.0` + `gorm-adapter/v3 v3.41.0`
- v3 的 `Enforce(sub,obj,act)` 已在现有中间件验证可用
- `AddPolicy` / `RemoveFilteredPolicy` / `InvalidateCache` 的**确切签名以 v3 实际 API 为准**（v3 ≠ v2，可能返回值/context 不同）；实现时核对 `go.sum` 锁定版本与 casbin v3 文档，不照搬 v2 用法

---

## 9. 验证闭环

```
1. 部署后端 → POST /init/initdb（前端引导填表 / 直连）
   → 跑 7 个 initializer：建部门/角色/菜单/用户/关联/casbin 策略
2. POST /auth/login（分别用 super / admin / test1 + 初始密码）→ 返回 JWT
   - super ：claims SuperAdmin=true,  RoleId=1
   - admin ：claims SuperAdmin=false, RoleId=2
   - test1 ：claims SuperAdmin=false, RoleId=3
3. GET /auth/getUserData：
   - super → roles:["superadmin"], permissions:["*:*:*"]
   - admin → roles:["admin"], permissions:[全部系统管理 perms（C+F 聚合）]
   - test1 → roles:["user"],         permissions:[]（无菜单）
4. 前端：
   - super / admin → hasAuth('system:user:add') 等全部放行
   - test1 → 所有系统管理按钮被 hasAuth 隐藏（permissions 为空）
5. API 兜底（casbin 中间件）：
   - super → SuperAdmin 豁免放行
   - admin → 命中其角色策略（系统管理 API）放行
   - test1 → 无策略 → 系统管理 API 返回「权限不足」
```

### 9.1 验收标准
- [ ] `POST /init/initdb` 成功执行全部 initializer，无「无可用初始化过程」错误；重复执行因幂等检查不报错
- [ ] DB 中存在：3 部门、3 角色(superadmin/admin/user)、3 用户(super/admin/test1)、5 系统管理菜单及按钮、关联表、`casbin_rule` 策略（角色 1/2 各一套）
- [ ] `POST /auth/login` 对 super/admin/test1 均返回有效 JWT
- [ ] `GET /auth/getUserData`：super→`["*:*:*"]`/`["superadmin"]`；admin→全部系统管理 perms/`["admin"]`；test1→`[]`/`["user"]`
- [ ] 前端：super/admin 登录后系统管理按钮全可见；test1 登录后系统管理按钮全隐藏
- [ ] API 兜底：super 豁免放行；admin 命中策略放行；test1 访问系统管理 API 返回「权限不足」
- [ ] 单元/集成测试覆盖：`UpdateCasbin` 策略生成（多角色）、`getUserData` 聚合（超管/管理员/普通用户三分支）

---

## 10. 影响面与风险

| 项 | 说明 / 缓解 |
|---|---|
| casbin v3 API 差异 | 实现时以 v3 文档为准；`UpdateCasbin` 写入后立即 `InvalidateCache`，避免 1h 缓存导致策略滞后 |
| `BaseClaims` 类型债 | 本期仅加 `SuperAdmin`，不动 `ID/RoleId` 的 uint 类型；统一留待后续用户/登录链路全面实现 |
| seed 菜单 apis 预填但 API 未实现 | 本期超管豁免不依赖策略命中；后续 CRUD 接入时 API 路径需与预填一致，否则非超管会被误拦 |
| `getUserData` 的 `user.roles` 嵌套结构 | 对齐前端 `Api.Auth.UserInfo`，含完整 Role 对象数组供展示；顶层 `roles`/`permissions` 为权限判断用 |
| 依赖新增 `gorm.io/datatypes` | 需更新 `go.mod`；该包为 GORM 官方扩展，无额外风险 |

---

## 11. 后续子项目（不在本期，仅备忘）

1. 系统管理 CRUD：user / role / menu / dept / post 的 service+api+router，前端 `_admin/system/*` 五页对接
2. 角色管理 UI 接入 casbin 联动：角色「分配菜单」保存时调 `UpdateCasbin`
3. `BaseClaims` 类型统一为 int64 + JWT token 内 ID 精度（claims 改 string 存储雪花 ID）
4. 前端动态菜单（`VITE_AUTH_ROUTE_MODE=dynamic`），后端 `/route/getUserRoutes` 按 sys_menu 生成
5. casbin 策略可视化管理 UI（可选，多数场景靠联动钩子维护即可）
