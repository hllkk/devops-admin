# 后端 User / Role 模型 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 定义后端 `SysUser`/`SysRole` 持久化模型、`sys_user_role`/`sys_role_menu` 关联表与 request DTO，对齐前端 RuoYi 契约，并注册进 AutoMigrate。

**Architecture:** 方案 A——User/Role 不内嵌 `global.OPS_MODEL`，自定义 int64 业务主键（`UserId`/`RoleId`，`json:",string"`），内嵌 system 包自己的 `AuditModel` 对齐前端 `CommonRecord`。雪花回调 `ops:snowflake_id` 看的是 `PrioritizedPrimaryField`（任意 int/uint 主键），不依赖基座，照常填主键。request DTO 内 ID 全用 `string`/`[]string`（线上 ID 字符串传输），service 层转 int64。

**Tech Stack:** Go 1.x + Gin + GORM v2 + glebarez/sqlite（纯 Go，测试用内存库）+ 项目自实现 `utils/snowflake`。

**Spec:** `docs/specs/2026-07-12-system-user-role-models-design.md`

## Global Constraints

- Go module 路径：`github.com/hllkk/devops-admin/server`；所有 `go` 命令在 `server/` 目录下执行。
- 基座约定：**仅** SysUser/SysRole 用 `model/system/AuditModel`；其它模型继续用 `global.OPS_MODEL`，不要改。
- 雪花主键：int64，`json:"<field>,string"`；回调仅在主键为 0 时填充，**显式 ID 不被覆盖**。
- 线上 ID 全字符串：DTO 的 ID 字段一律 `string`/`[]string`，不要写 `int`/`[]int`。
- 前端契约：json key 全 camelCase，与 `Api.System.User`/`Role` 字段名 1:1。
- 状态码：`"0"` 正常 / `"1"` 停用（char，不是 bool）。
- 严禁动 service/api/router（不在本计划范围）；严禁改 `Login` 接口与 JWT claims（已知技术债，登录时再处理）。
- 测试用纯 `testing` + `t.Errorf`（对齐 `utils/snowflake/snowflake_test.go` 既有风格），不引入 testify。
- 每个任务结束提交一次；commit message 用 conventional commits。

## File Structure

| 文件 | 责任 | 动作 |
|---|---|---|
| `server/model/system/base.go` | `AuditModel` 审计 mixin + 状态常量 | 新建 |
| `server/model/system/sys_user.go` | `SysUser` 实体（替换注释桩，保留 `Login` 接口） | 改 |
| `server/model/system/sys_role.go` | `SysRole` 实体 | 新建 |
| `server/model/system/sys_user_role.go` | 用户-角色关联表（复合主键） | 新建 |
| `server/model/system/sys_role_menu.go` | 角色-菜单关联表（复合主键） | 新建 |
| `server/model/system/request/sys_user.go` | 用户 search/operate/改密/改状态 DTO | 新建 |
| `server/model/system/request/sys_role.go` | 角色 search/operate/授权用户 DTO | 新建 |
| `server/model/system/sys_user_test.go` | SysUser JSON 契约单测 | 新建 |
| `server/model/system/sys_role_test.go` | SysRole JSON 契约单测 | 新建 |
| `server/model/system/request/sys_user_req_test.go` | SysUserReq 绑定单测 | 新建 |
| `server/initialize/system_models_test.go` | 雪花+迁移+复合主键集成测试 | 新建 |
| `server/initialize/gorm.go` | `RegisterTables()` 注册 4 张表 | 改 |

---

## Task 1: AuditModel + SysUser 模型

**Files:**
- Create: `server/model/system/base.go`
- Create: `server/model/system/sys_user_test.go`
- Modify: `server/model/system/sys_user.go`（替换 16-20 行注释桩，保留 `Login` 接口）

**Interfaces:**
- Produces: `system.AuditModel`（嵌入结构）、`system.SysUser`（实体）、`system.SysUser.TableName()`、`system.StatusEnable`/`system.StatusDisable` 常量。Task 2/3/5 依赖。

- [ ] **Step 1: 写失败测试** — 创建 `server/model/system/sys_user_test.go`：

```go
package system

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSysUserJSONContract(t *testing.T) {
	now := time.Date(2026, 7, 12, 10, 0, 0, 0, time.UTC)
	u := SysUser{
		UserId:   1234567890123456,
		UserName: "alice",
		Password: "secret",
		Status:   StatusEnable,
		AuditModel: AuditModel{
			CreateBy:   "admin",
			CreateTime: now,
			UpdateBy:   "admin",
			UpdateTime: now,
		},
	}
	out, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// userId 必须是字符串（雪花防精度丢失）
	if v, ok := m["userId"].(string); !ok || v != "1234567890123456" {
		t.Errorf("userId 应为字符串 \"1234567890123456\"，实际 %v", m["userId"])
	}
	// password 永不外泄
	if _, has := m["password"]; has {
		t.Error("password 不应被序列化")
	}
	// deletedAt 不外泄
	if _, ok := m["deletedAt"]; ok {
		t.Error("deletedAt 不应被序列化")
	}
	// 审计与业务字段 camelCase 齐全
	for _, k := range []string{"userName", "status", "createBy", "createTime", "updateBy", "updateTime"} {
		if _, ok := m[k]; !ok {
			t.Errorf("缺少字段 %s", k)
		}
	}
}
```

- [ ] **Step 2: 跑测试确认失败** — Run: `cd server && go test ./model/system/ -run TestSysUserJSONContract -v`
  Expected: FAIL，编译错误 `undefined: SysUser` / `undefined: AuditModel` / `undefined: StatusEnable`。

- [ ] **Step 3: 创建 base.go** — 创建 `server/model/system/base.go`：

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

// 状态码（char），与前端 EnableStatus 对齐：0 正常 / 1 停用
const (
	StatusEnable  = "0"
	StatusDisable = "1"
)
```

- [ ] **Step 4: 实现 SysUser** — 用下面内容**整体替换** `server/model/system/sys_user.go`：

```go
package system

import (
	"time"

	"github.com/google/uuid"
)

// Login 认证链路抽象（GVA 基座遗留）。ID 仍是 uint，待登录实现时统一改 int64/string
// （见 aiDoc/frontend-backend/boundary.md 主键 ID 契约的例外说明）。
type Login interface {
	GetUsername() string
	GetNickname() string
	GetUUID() uuid.UUID
	GetUserId() uint
	GetAuthorityId() uint
	GetUserInfo() any
}

// SysUser 系统用户，对齐前端 Api.System.User。
// 主键 UserId 由雪花回调 ops:snowflake_id 自动填充（不内嵌 global.OPS_MODEL）。
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

// TableName 自定义表名 sys_user
func (SysUser) TableName() string { return "sys_user" }
```

- [ ] **Step 5: 跑测试确认通过** — Run: `cd server && go test ./model/system/ -run TestSysUserJSONContract -v`
  Expected: PASS `--- PASS: TestSysUserJSONContract`.

- [ ] **Step 6: vet** — Run: `cd server && go vet ./model/system/`
  Expected: 无输出（通过）。

- [ ] **Step 7: 提交**
```bash
git add server/model/system/base.go server/model/system/sys_user.go server/model/system/sys_user_test.go
git commit -m "feat(system): 新增 SysUser 模型与 AuditModel 审计基座"
```

---

## Task 2: SysRole 模型

**Files:**
- Create: `server/model/system/sys_role.go`
- Create: `server/model/system/sys_role_test.go`

**Interfaces:**
- Consumes: `system.AuditModel`、`system.StatusEnable`（来自 Task 1）。
- Produces: `system.SysRole`、`system.SysRole.TableName()`。Task 3/5 依赖。

- [ ] **Step 1: 写失败测试** — 创建 `server/model/system/sys_role_test.go`：

```go
package system

import (
	"encoding/json"
	"testing"
)

func TestSysRoleJSONContract(t *testing.T) {
	r := SysRole{
		RoleId:   1,
		RoleName: "超管",
		RoleKey:  "SUPER",
		RoleSort: 1,
		Status:   StatusEnable,
	}
	out, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// roleId 必须是字符串
	if v, ok := m["roleId"].(string); !ok || v != "1" {
		t.Errorf("roleId 应为字符串 \"1\"，实际 %v", m["roleId"])
	}
	// 业务字段齐全
	for _, k := range []string{"roleName", "roleKey", "roleSort", "menuCheckStrictly", "status", "superAdmin", "flag"} {
		if _, ok := m[k]; !ok {
			t.Errorf("缺少字段 %s", k)
		}
	}
	// deletedAt 不外泄
	if _, ok := m["deletedAt"]; ok {
		t.Error("deletedAt 不应被序列化")
	}
}
```

- [ ] **Step 2: 跑测试确认失败** — Run: `cd server && go test ./model/system/ -run TestSysRoleJSONContract -v`
  Expected: FAIL，`undefined: SysRole`。

- [ ] **Step 3: 实现 SysRole** — 创建 `server/model/system/sys_role.go`：

```go
package system

// SysRole 系统角色，对齐前端 Api.System.Role。
// 超管角色由初始化流程以显式 RoleId=1 种子（雪花回调只在主键为 0 时填，不覆盖）。
type SysRole struct {
	RoleId            int64  `gorm:"column:role_id;primaryKey;autoIncrement:false" json:"roleId,string"`
	RoleName          string `gorm:"column:role_name;size:64" json:"roleName"`
	RoleKey           string `gorm:"column:role_key;size:100" json:"roleKey"`
	RoleSort          int    `gorm:"column:role_sort;default:0" json:"roleSort"`
	MenuCheckStrictly bool   `gorm:"column:menu_check_strictly;default:false" json:"menuCheckStrictly"`
	Status            string `gorm:"column:status;size:1;default:'0'" json:"status"`
	SuperAdmin        bool   `gorm:"column:super_admin;default:false" json:"superAdmin"`
	Remark            string `gorm:"column:remark;size:500;default:''" json:"remark"`
	Flag              bool   `gorm:"-" json:"flag"` // 瞬态：当前用户是否拥有此角色，不入库
	AuditModel
}

// TableName 自定义表名 sys_role
func (SysRole) TableName() string { return "sys_role" }
```

- [ ] **Step 4: 跑测试确认通过** — Run: `cd server && go test ./model/system/ -run TestSysRoleJSONContract -v`
  Expected: PASS。

- [ ] **Step 5: vet** — Run: `cd server && go vet ./model/system/`
  Expected: 无输出。

- [ ] **Step 6: 提交**
```bash
git add server/model/system/sys_role.go server/model/system/sys_role_test.go
git commit -m "feat(system): 新增 SysRole 模型"
```

---

## Task 3: 关联表 + 雪花/迁移集成测试

**Files:**
- Create: `server/model/system/sys_user_role.go`
- Create: `server/model/system/sys_role_menu.go`
- Create: `server/initialize/system_models_test.go`

**Interfaces:**
- Consumes: `system.SysUser`、`system.SysRole`（Task 1/2）；`initialize.RegisterCallbacks(db)`、`snowflake.MustInit`（既有）。
- Produces: `system.SysUserRole`、`system.SysRoleMenu`。Task 5（迁移注册）依赖。
- 依赖：`github.com/glebarez/sqlite`（已在 go.mod，纯 Go 无 CGO）。

- [ ] **Step 1: 写失败测试** — 创建 `server/initialize/system_models_test.go`：

```go
package initialize

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/utils/snowflake"
)

// TestSystemModelsMigrateAndSnowflake 验证：
// 1) 四张表能 AutoMigrate；2) SysUser 主键为 0 时雪花回调填非零 ID；
// 3) password 不外泄；4) 关联表复合主键的显式 ID 不被回调覆盖。
func TestSystemModelsMigrateAndSnowflake(t *testing.T) {
	// 雪花初始化（幂等；epoch 同 snowflake_test.go 的 testEpoch）
	snowflake.MustInit(1, time.Unix(1704067200, 0).UTC())

	// sqlite 内存库（cache=shared 让连接池共享同一库）
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	RegisterCallbacks(db)

	if err := db.AutoMigrate(
		&system.SysUser{},
		&system.SysRole{},
		&system.SysUserRole{},
		&system.SysRoleMenu{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	// 主键为 0 → 雪花回调应填充
	u := system.SysUser{UserName: "alice", Password: "should-not-leak", Status: system.StatusEnable}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if u.UserId == 0 {
		t.Fatal("雪花回调应填充 UserId，得到 0")
	}

	// JSON 契约：userId 字符串、password 不外泄
	out, _ := json.Marshal(u)
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := m["userId"].(string); !ok {
		t.Errorf("userId 应为 JSON 字符串，实际 %T", m["userId"])
	}
	if _, has := m["password"]; has {
		t.Error("password 不应被序列化")
	}

	// 关联表复合主键：显式 ID 不被覆盖
	ur := system.SysUserRole{UserId: u.UserId, RoleId: 999}
	if err := db.Create(&ur).Error; err != nil {
		t.Fatalf("create user-role: %v", err)
	}
	if ur.UserId != u.UserId || ur.RoleId != 999 {
		t.Errorf("关联表显式主键被覆盖：got UserId=%d RoleId=%d", ur.UserId, ur.RoleId)
	}

	rm := system.SysRoleMenu{RoleId: 999, MenuId: 7}
	if err := db.Create(&rm).Error; err != nil {
		t.Fatalf("create role-menu: %v", err)
	}
	if rm.RoleId != 999 || rm.MenuId != 7 {
		t.Errorf("role-menu 显式主键被覆盖：got RoleId=%d MenuId=%d", rm.RoleId, rm.MenuId)
	}
}
```

- [ ] **Step 2: 跑测试确认失败** — Run: `cd server && go test ./initialize/ -run TestSystemModelsMigrateAndSnowflake -v`
  Expected: FAIL，`undefined: system.SysUserRole` / `undefined: system.SysRoleMenu`。

- [ ] **Step 3: 实现 SysUserRole** — 创建 `server/model/system/sys_user_role.go`：

```go
package system

// SysUserRole 用户-角色关联表（复合主键）。
// 插入时必须显式指定 UserId/RoleId；雪花回调仅在主键为 0 时填充，不覆盖显式值。
type SysUserRole struct {
	UserId int64 `gorm:"column:user_id;primaryKey;autoIncrement:false" json:"userId,string"`
	RoleId int64 `gorm:"column:role_id;primaryKey;autoIncrement:false" json:"roleId,string"`
}

// TableName 自定义表名 sys_user_role
func (SysUserRole) TableName() string { return "sys_user_role" }
```

- [ ] **Step 4: 实现 SysRoleMenu** — 创建 `server/model/system/sys_role_menu.go`：

```go
package system

// SysRoleMenu 角色-菜单关联表（复合主键）。同 SysUserRole，插入须显式指定两个 ID。
type SysRoleMenu struct {
	RoleId int64 `gorm:"column:role_id;primaryKey;autoIncrement:false" json:"roleId,string"`
	MenuId int64 `gorm:"column:menu_id;primaryKey;autoIncrement:false" json:"menuId,string"`
}

// TableName 自定义表名 sys_role_menu
func (SysRoleMenu) TableName() string { return "sys_role_menu" }
```

- [ ] **Step 5: 跑测试确认通过** — Run: `cd server && go test ./initialize/ -run TestSystemModelsMigrateAndSnowflake -v`
  Expected: PASS。

- [ ] **Step 6: vet** — Run: `cd server && go vet ./initialize/ ./model/system/`
  Expected: 无输出。

- [ ] **Step 7: 提交**
```bash
git add server/model/system/sys_user_role.go server/model/system/sys_role_menu.go server/initialize/system_models_test.go
git commit -m "feat(system): 新增 user-role/role-menu 关联表与雪花迁移集成测试"
```

---

## Task 4: request DTO

**Files:**
- Create: `server/model/system/request/sys_user.go`
- Create: `server/model/system/request/sys_role.go`
- Create: `server/model/system/request/sys_user_req_test.go`

**Interfaces:**
- Consumes: `request.PageInfo`（既有，`github.com/hllkk/devops-admin/server/model/common/request`）。
- Produces: `request.SysUserSearch`/`SysUserReq`/`SysResetPwdReq`/`SysUserStatusReq`/`SysRoleSearch`/`SysRoleReq`/`SysRoleAuthUserReq`。供后续 service/api 层使用。

- [ ] **Step 1: 写失败测试** — 创建 `server/model/system/request/sys_user_req_test.go`：

```go
package request

import (
	"encoding/json"
	"testing"
)

// 验证线上 ID 全字符串：标量 *string、数组 []string 都能从 JSON 正确绑定。
func TestSysUserReqBindsStringIDs(t *testing.T) {
	raw := `{"userId":"123","deptId":"42","userName":"alice","status":"0","roleIds":["1","2"],"postIds":[]}`
	var req SysUserReq
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.UserId == nil || *req.UserId != "123" {
		t.Errorf("userId 绑定错误：%v", req.UserId)
	}
	if req.DeptId == nil || *req.DeptId != "42" {
		t.Errorf("deptId 绑定错误：%v", req.DeptId)
	}
	if len(req.RoleIds) != 2 || req.RoleIds[0] != "1" || req.RoleIds[1] != "2" {
		t.Errorf("roleIds 绑定错误：%v", req.RoleIds)
	}
	if len(req.PostIds) != 0 {
		t.Errorf("postIds 应为空切片，实际 %v", req.PostIds)
	}
}
```

- [ ] **Step 2: 跑测试确认失败** — Run: `cd server && go test ./model/system/request/ -run TestSysUserReqBindsStringIDs -v`
  Expected: FAIL，`undefined: SysUserReq`。

- [ ] **Step 3: 实现用户 DTO** — 创建 `server/model/system/request/sys_user.go`：

```go
package request

import (
	"github.com/hllkk/devops-admin/server/model/common/request"
)

// SysUserSearch 用户列表查询，对齐前端 Api.System.UserSearchParams。
// 注意：嵌入的 request.PageInfo 的 json tag 是 "page"（非 pageNum），
// 但列表接口走 GET query，按 form:"pageNum" 绑定，不受影响。
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

// SysUserReq 用户新增/修改，对齐前端 Api.System.UserOperateParams（create 时 userId 为 nil）。
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

// SysResetPwdReq 重置密码（前端带 isEncrypt 头，加解密在 service/api 层处理）。
type SysResetPwdReq struct {
	UserId   string `json:"userId" form:"userId"`
	Password string `json:"password" form:"password"`
}

// SysUserStatusReq 修改帐号状态。
type SysUserStatusReq struct {
	UserId string `json:"userId" form:"userId"`
	Status string `json:"status" form:"status"`
}
```

- [ ] **Step 4: 实现角色 DTO** — 创建 `server/model/system/request/sys_role.go`：

```go
package request

import (
	"github.com/hllkk/devops-admin/server/model/common/request"
)

// SysRoleSearch 角色列表查询，对齐前端 Api.System.RoleSearchParams。
type SysRoleSearch struct {
	RoleName      string `json:"roleName" form:"roleName"`
	RoleKey       string `json:"roleKey" form:"roleKey"`
	Status        string `json:"status" form:"status"`
	OrderByColumn string `json:"orderByColumn" form:"orderByColumn"`
	IsAsc         string `json:"isAsc" form:"isAsc"`
	request.PageInfo
}

// SysRoleReq 角色新增/修改，对齐前端 Api.System.RoleOperateParams（create 时 roleId 为 nil）。
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

// SysRoleAuthUserReq 角色分配/取消用户（userIds 逗号分隔，由 API 层切分）。
type SysRoleAuthUserReq struct {
	RoleId  string `json:"roleId" form:"roleId"`
	UserIds string `json:"userIds" form:"userIds"`
}
```

- [ ] **Step 5: 跑测试确认通过** — Run: `cd server && go test ./model/system/request/ -run TestSysUserReqBindsStringIDs -v`
  Expected: PASS。

- [ ] **Step 6: 全量 build + vet** — Run: `cd server && go build ./... && go vet ./model/...`
  Expected: 无输出（通过）。

- [ ] **Step 7: 提交**
```bash
git add server/model/system/request/sys_user.go server/model/system/request/sys_role.go server/model/system/request/sys_user_req_test.go
git commit -m "feat(system): 新增用户/角色 request DTO"
```

---

## Task 5: 注册 AutoMigrate + 全量验证

**Files:**
- Modify: `server/initialize/gorm.go:43-45`（`RegisterTables()` 的 `AutoMigrate` 调用）

**Interfaces:**
- Consumes: `system.SysUser`/`SysRole`/`SysUserRole`/`SysRoleMenu`（Task 1/2/3）。
- Produces: 启动时自动建出 4 张系统表。

- [ ] **Step 1: 修改 AutoMigrate** — 在 `server/initialize/gorm.go` 把：

```go
	err := db.AutoMigrate(
		system.SysError{},
	)
```

改为：

```go
	err := db.AutoMigrate(
		system.SysError{},
		system.SysUser{},
		system.SysRole{},
		system.SysUserRole{},
		system.SysRoleMenu{},
	)
```

- [ ] **Step 2: 全量 build + vet** — Run: `cd server && go build ./... && go vet ./...`
  Expected: 无输出（通过）。

- [ ] **Step 3: 全量测试** — Run: `cd server && go test ./model/... ./initialize/ ./utils/snowflake/`
  Expected: 全部 PASS（含本计划新增的 4 个测试 + 既有 snowflake 测试）。

- [ ] **Step 4: 提交**
```bash
git add server/initialize/gorm.go
git commit -m "feat(system): 注册 user/role/user_role/role_menu 表到 AutoMigrate"
```

- [ ] **Step 5: 收尾确认** — 对照 `docs/specs/2026-07-12-system-user-role-models-design.md` 第 8 节验证标准逐条核对：build/vet 通过 ✓、4 张表可 AutoMigrate（Task 3 集成测试已证）✓、SysUser 落库 UserId 被雪花填且 JSON 产出 userId 字符串、无 password（Task 1/3 测试已证）✓。
