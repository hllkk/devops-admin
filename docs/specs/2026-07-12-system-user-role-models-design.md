# 后端 User / Role 模型设计

- 日期：2026-07-12
- 状态：设计中（待评审 → 实施）
- 范围：`server/` Model 层（持久化模型 + 关联表 + request DTO + AutoMigrate 注册）
- 关联前端：`web/src/typings/api/system.api.d.ts`（`Api.System.User` / `Api.System.Role`，commit 5199562，仅前端）

## 1. 目标

前端已落地 RuoYi 风格的用户/角色页面与类型（`/system/user`、`/system/role` 一整套接口调用），但后端**没有任何 user/role 模型**（`SysUser` 是注释桩，`SysRole` 不存在）。本设计定义后端 Model 层，使后端数据模型与前端契约 1:1 对齐，并把表注册进 AutoMigrate，为后续 service/api/router 三层与鉴权链路打基础。

## 2. 范围

**本次包含：**
- 持久化模型：`SysUser`、`SysRole`
- 关联表：`SysUserRole`、`SysRoleMenu`
- request DTO：用户/角色的 search + operate + 改密/改状态/角色授权
- AutoMigrate 注册（`initialize/gorm.go` `RegisterTables()`）
- 文档与业务记忆同步

**本次不含（下一阶段）：**
- service / api / router 三层实现
- 部门（Dept，树形）、岗位（Post）建模 —— `deptId` 暂存普通字段，`postIds` 留空数组
- 超管角色 `RoleId=1` 种子（初始化流程的活）
- 鉴权链路 int64/string 改造（`Login` 接口、JWT claims）

## 3. 关键背景与勘误

1. **基座是 `global.OPS_MODEL`，不是 `GVA_MODEL`。** `aiDoc/modules/backend-layer-rules.md` 里"优先继承 `global.GVA_MODEL`"是 stale 文档，代码中 `GVA_MODEL` 不存在。`OPS_MODEL` 定义于 `server/global/common.go`：
   ```go
   type OPS_MODEL struct {
       ID        int64          `gorm:"primaryKey;autoIncrement:false" json:"ID,string"`
       CreatedAt time.Time
       UpdatedAt time.Time
       DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
   }
   ```
2. **雪花回调不依赖基座。** `initialize/callbacks.go` 的 `ops:snowflake_id` 回调看的是 `Schema.PrioritizedPrimaryField`——任意 `int`/`uint` 主键在为 0 时都会被填雪花 ID。所以模型**不必内嵌 `OPS_MODEL`**，只要主键是 int64 即可被自动填充。
3. **前端契约是 RuoYi 风格且已固化。** `Api.System.User`/`Role` 包在 `CommonRecord<T>` 里，实际产出/消费 camelCase 的 `userId`/`roleId`/`createTime`/`createBy`/`updateTime`，`EnableStatus='0'|'1'`，分页 `{pageNum,pageSize,total,rows}`，响应 `{code:string,msg,data}`。前端代码已直接消费这些 key（如 `role/index.vue` 的 `row.roleId === 1`）。

## 4. 核心决策：契约 reconcile

前端 RuoYi key（`userId`/`createTime`）与 `OPS_MODEL` 的 GVA key（`ID`/`CreatedAt`）完全对不上。三个方案：

| 方案 | 做法 | 结论 |
|---|---|---|
| **A（采纳）** | User/Role 不内嵌 `OPS_MODEL`；各自定义 int64 业务主键（`UserId`/`RoleId`，`json:"userId,string"`），内嵌 system 模块自己的审计 mixin 对齐 `CommonRecord` | JSON 与前端 1:1，雪花回调照常生效 |
| B | 内嵌 `OPS_MODEL` + 叠加前端字段 / 自定义 `MarshalJSON` | 两个主键概念或每张表写 marshal，脏 |
| C | 改前端贴 `OPS_MODEL` key | 前端刚落地，违背"后端跟前端走" |

**用户拍板：方案 A；审计 mixin 放 `server/model/system/`（命名 `AuditModel`）；其它模型继续用 `global.OPS_MODEL`。** 边界清晰：`OPS_MODEL` = GVA 内部表基座；`system/AuditModel` = system 对外 RuoYi 表基座。

## 5. 设计明细

### 5.1 审计 mixin — `server/model/system/base.go`（新建）

```go
package system

import (
	"time"
	"gorm.io/gorm"
)

// AuditModel 对齐前端 Api.Common.CommonRecord 的审计字段，仅用于 system 模块
// 需贴合 RuoYi/前端契约的对外模型（SysUser / SysRole）。
// 其它模型继续使用 global.OPS_MODEL。
type AuditModel struct {
	CreateBy   string         `gorm:"column:create_by;size:64;default:''" json:"createBy"`
	CreateTime time.Time      `gorm:"column:create_time;autoCreateTime" json:"createTime"`
	UpdateBy   string         `gorm:"column:update_by;size:64;default:''" json:"updateBy"`
	UpdateTime time.Time      `gorm:"column:update_time;autoUpdateTime" json:"updateTime"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

const (
	StatusEnable  = "0" // 正常
	StatusDisable = "1" // 停用
)
```

`autoCreateTime`/`autoUpdateTime` 由 GORM 自动维护；`CreateBy`/`UpdateBy` 需当前登录用户，属 service 层职责（无鉴权前留空）。前端 `createDept?` 可选 `any`，不落库。

### 5.2 SysUser — `server/model/system/sys_user.go`（替换注释桩，保留 `Login` 接口）

```go
type SysUser struct {
	UserId      int64      `gorm:"column:user_id;primaryKey;autoIncrement:false" json:"userId,string"`
	DeptId      int64      `gorm:"column:dept_id;default:0" json:"deptId,string"`
	DeptName    string     `gorm:"column:dept_name;size:64;default:''" json:"deptName"`
	UserName    string     `gorm:"column:user_name;size:64;uniqueIndex" json:"userName"`
	NickName    string     `gorm:"column:nick_name;size:64" json:"nickName"`
	UserType    string     `gorm:"column:user_type;size:16;default:'00'" json:"userType"`
	Email       string     `gorm:"column:email;size:128;default:''" json:"email"`
	Phonenumber string     `gorm:"column:phonenumber;size:20;default:''" json:"phonenumber"`
	Sex         string     `gorm:"column:sex;size:1;default:'2'" json:"sex"`
	Avatar      string     `gorm:"column:avatar;size:512;default:''" json:"avatar"`
	Password    string     `gorm:"column:password;size:128" json:"-"`
	Status      string     `gorm:"column:status;size:1;default:'0'" json:"status"`
	LoginIp     string     `gorm:"column:login_ip;size:128;default:''" json:"loginIp"`
	LoginDate   *time.Time `gorm:"column:login_date" json:"loginDate"`
	Remark      string     `gorm:"column:remark;size:500;default:''" json:"remark"`
	AuditModel
}

func (SysUser) TableName() string { return "sys_user" }
```

- `Password` `json:"-"`：永不外泄；新增/改密走 request DTO 绑定，不直接 marshal 模型。
- `LoginDate *time.Time`：指针可空（首次登录前为 null）。
- `UserId` 为唯一主键，雪花回调自动填。
- `uniqueIndex` + 软删除潜在冲突见第 6 节坑位。

### 5.3 SysRole — `server/model/system/sys_role.go`（新建）

```go
type SysRole struct {
	RoleId            int64  `gorm:"column:role_id;primaryKey;autoIncrement:false" json:"roleId,string"`
	RoleName          string `gorm:"column:role_name;size:64" json:"roleName"`
	RoleKey           string `gorm:"column:role_key;size:100" json:"roleKey"`
	RoleSort          int    `gorm:"column:role_sort;default:0" json:"roleSort"`
	MenuCheckStrictly bool   `gorm:"column:menu_check_strictly;default:false" json:"menuCheckStrictly"`
	Status            string `gorm:"column:status;size:1;default:'0'" json:"status"`
	SuperAdmin        bool   `gorm:"column:super_admin;default:false" json:"superAdmin"`
	Remark            string `gorm:"column:remark;size:500;default:''" json:"remark"`
	Flag              bool   `gorm:"-" json:"flag"`
	AuditModel
}

func (SysRole) TableName() string { return "sys_role" }
```

- `Flag`：前端角色分配视图的瞬态字段（当前用户是否拥有此角色），`gorm:"-"` 不入库。
- 超管角色由初始化流程以显式 `RoleId=1` 种子（雪花回调只在主键为 0 时填，不覆盖显式 ID）。

### 5.4 关联表（复合主键）

```go
// server/model/system/sys_user_role.go
type SysUserRole struct {
	UserId int64 `gorm:"column:user_id;primaryKey;autoIncrement:false" json:"userId,string"`
	RoleId int64 `gorm:"column:role_id;primaryKey;autoIncrement:false" json:"roleId,string"`
}
func (SysUserRole) TableName() string { return "sys_user_role" }

// server/model/system/sys_role_menu.go
type SysRoleMenu struct {
	RoleId int64 `gorm:"column:role_id;primaryKey;autoIncrement:false" json:"roleId,string"`
	MenuId int64 `gorm:"column:menu_id;primaryKey;autoIncrement:false" json:"menuId,string"`
}
func (SysRoleMenu) TableName() string { return "sys_role_menu" }
```

**service 层强制约定：** 关联表插入必须显式传两个 ID。雪花回调会把复合主键首字段当 `PrioritizedPrimaryField`，仅在为 0 时填——只要总显式传值，就不会被错误覆盖。

### 5.5 request DTO — `server/model/system/request/`

**契约原则：** 线上 ID 一律字符串（雪花防精度丢失），DTO 内 ID 字段全部 `string`/`[]string`，service 层转 int64 落库。与现有 `sys_error` 按 `ID string` 删除的惯例一致。

```go
// sys_user.go
type SysUserSearch struct {
	DeptId        *string `json:"deptId" form:"deptId"`
	UserName      string  `json:"userName" form:"userName"`
	NickName      string  `json:"nickName" form:"nickName"`
	Phonenumber   string  `json:"phonenumber" form:"phonenumber"`
	Status        string  `json:"status" form:"status"`
	RoleId        *string `json:"roleId" form:"roleId"`
	OrderByColumn string  `json:"orderByColumn" form:"orderByColumn"`
	IsAsc         string  `json:"isAsc" form:"isAsc"`
	request.PageInfo
}

type SysUserReq struct {
	UserId      *string  `json:"userId" form:"userId"`
	DeptId      *string  `json:"deptId" form:"deptId"`
	UserName    string   `json:"userName" form:"userName"`
	NickName    string   `json:"nickName" form:"nickName"`
	Email       string   `json:"email" form:"email"`
	Phonenumber string   `json:"phonenumber" form:"phonenumber"`
	Sex         string   `json:"sex" form:"sex"`
	Password    string   `json:"password" form:"password"`
	Status      string   `json:"status" form:"status"`
	Remark      string   `json:"remark" form:"remark"`
	RoleIds     []string `json:"roleIds" form:"roleIds"`
	PostIds     []string `json:"postIds" form:"postIds"`
}

type SysResetPwdReq struct {
	UserId   string `json:"userId" form:"userId"`
	Password string `json:"password" form:"password"`
}

type SysUserStatusReq struct {
	UserId string `json:"userId" form:"userId"`
	Status string `json:"status" form:"status"`
}
```

```go
// sys_role.go
type SysRoleSearch struct {
	RoleName      string `json:"roleName" form:"roleName"`
	RoleKey       string `json:"roleKey" form:"roleKey"`
	Status        string `json:"status" form:"status"`
	OrderByColumn string `json:"orderByColumn" form:"orderByColumn"`
	IsAsc         string `json:"isAsc" form:"isAsc"`
	request.PageInfo
}

type SysRoleReq struct {
	RoleId            *string  `json:"roleId" form:"roleId"`
	RoleName          string   `json:"roleName" form:"roleName"`
	RoleKey           string   `json:"roleKey" form:"roleKey"`
	RoleSort          int      `json:"roleSort" form:"roleSort"`
	MenuCheckStrictly bool     `json:"menuCheckStrictly" form:"menuCheckStrictly"`
	Status            string   `json:"status" form:"status"`
	Remark            string   `json:"remark" form:"remark"`
	MenuIds           []string `json:"menuIds" form:"menuIds"`
}

type SysRoleAuthUserReq struct {
	RoleId  string `json:"roleId" form:"roleId"`
	UserIds string `json:"userIds" form:"userIds"`
}
```

批量删除走 path（`/system/user/{userIds}`、`/system/role/{roleIds}` 逗号拼接），API 层切分，无需 body DTO。

`request.PageInfo`（已存在）字段：`PageNum json:"page" form:"pageNum"`、`PageSize`、`Keyword`，带 `Paginate()`。前端 `CommonSearchParams` 的 `pageNum`/`pageSize` 走 form 绑定可直接命中。

### 5.6 AutoMigrate 注册 — `server/initialize/gorm.go`

`RegisterTables()` 的 `AutoMigrate(...)` 增加：

```go
err := db.AutoMigrate(
	system.SysError{},
	system.SysUser{},
	system.SysRole{},
	system.SysUserRole{},
	system.SysRoleMenu{},
)
```

system 模块表进 `RegisterTables()`，不进 `bizModel()`。

## 6. 坑位与已知限制

1. **关联表复合主键 + 雪花回调**：插入必须显式传两个 ID（见 5.4）。
2. **超管 `RoleId=1` 种子**：初始化流程的活，本次只保证模型支持。
3. **`Login` 接口与 JWT claims** 仍是 `uint`/老格式；做登录时再统一改 int64/string（boundary.md 已记 TODO），本次不动。
4. **`UserName` `uniqueIndex` + 软删除**：同名重建可能冲突，先简单处理，后续可加含 `deleted_at` 的联合唯一索引。

## 7. 文档与记忆同步（按项目规则）

- 修订 `aiDoc/modules/backend-layer-rules.md`：`GVA_MODEL` → `OPS_MODEL`；补一句 system 对外 RuoYi 模型用 `model/system/AuditModel`。
- 新增业务记忆 `aiDoc/memory/business/system-user-role-models.md` + 更新 `aiDoc/memory/demand-index.md`。

## 8. 验证标准（实施后）

- `go build ./...`、`go vet ./...` 通过。
- 首次启动 AutoMigrate 建出 `sys_user`/`sys_role`/`sys_user_role`/`sys_role_menu` 四张表，列名/类型与 gorm tag 一致。
- 构造一条 `SysUser{}` 落库，`ID`（`UserId`）被雪花回调填为非零 int64，JSON 序列化产出 `userId` 为字符串、`createTime` 为 RFC3339 字符串、不含 `password`。
