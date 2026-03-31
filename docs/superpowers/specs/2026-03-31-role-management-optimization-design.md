---
name: 角色管理功能优化
description: 完整实现后端角色管理功能，修复前后端对接问题，补充模型缺失字段
type: project
created: 2026-03-31
---

# 角色管理功能优化设计文档

## 问题概述

当前角色管理功能存在严重的后端实现缺失问题：

1. **路由缺失**：后端只实现了 `select` 和 `list` 两个接口，前端调用的 10+ 个接口全部缺失
2. **Service层缺失**：只有查询方法，缺少创建、更新、删除、状态修改、用户授权等
3. **API层缺失**：对应Service层的所有增删改方法都未实现
4. **模型字段缺失**：缺少 `dataScope` 字段
5. **前后端字段映射不一致**：参数名称不匹配导致查询失败

## 问题根因分析

### 前后端字段映射问题

| 功能 | 前端参数名 | 后端当前参数名 | 应统一为 |
|------|-----------|---------------|---------|
| 角色名称 | `roleName` | `name` | `roleName` |
| 角色编码 | `roleKey` | `code` | `roleKey` |
| 分页页码 | `pageNum` | `page` | `pageNum` |

### 模型字段缺失

`SysRole` 模型缺少 `DataScope` 字段，但前端已使用该字段显示数据范围权限。

### 后端接口缺失清单

| 接口 | 前端调用 | 后端实现 | 状态 |
|------|---------|---------|------|
| GET `/system/role/list` | ✅ | ✅ | 已实现 |
| GET `/system/role/select` | ✅ | ✅ | 已实现 |
| GET `/system/role/:id` | ✅ | ❌ | 缺失 |
| POST `/system/role` | ✅ | ❌ | 缺失 |
| PUT `/system/role` | ✅ | ❌ | 缺失 |
| DELETE `/system/role/:ids` | ✅ | ❌ | 缺失 |
| PUT `/system/role/changeStatus` | ✅ | ❌ | 缺失 |
| PUT `/system/role/dataScope` | ✅ | ❌ | 缺失 |
| GET `/system/role/authUser/allocatedList` | ✅ | ❌ | 缺失 |
| PUT `/system/role/authUser/selectAll` | ✅ | ❌ | 缺失 |
| PUT `/system/role/authUser/cancelAll` | ✅ | ❌ | 缺失 |
| GET `/system/role/deptTree/:id` | ✅ | ❌ | 缺失 |
| POST `/system/role/export` | ✅ | ❌ | 缺失 |

## 解决方案

采用方案A：完整实现后端角色管理功能。

## 详细设计

### 一、模型层修复

**文件：** `backend/model/system/sys_role.go`

添加 `DataScope` 字段：

```go
type SysRole struct {
    global.OPS_MODEL
    RbacOperatorModel
    RoleName    string       `gorm:"column:role_name;size:32;unique;index;not null;comment:角色名称" json:"roleName"`
    RoleCode    string       `gorm:"column:role_code;size:20;unique;index;not null;comment:角色编码" json:"roleKey"`
    Description string       `gorm:"size:255;comment:角色描述" json:"description"`
    SortNum     int          `gorm:"column:sort_num;default:0;comment:排序" json:"roleSort"`
    Status      string       `gorm:"not null;type:char(1);default:1;comment:状态（'1'正常 '0'停用）" json:"status"`
    DataScope   string       `gorm:"type:char(1);default:1;comment:数据范围" json:"dataScope"`
    // ... 关联字段保持不变
}
```

**注意：** 同时添加 `column` 标签确保数据库字段映射正确。

### 二、Request 层完善

**文件：** `backend/model/system/request/sys_role.go`

```go
package request

import (
    "github.com/hllkk/devopsAdmin/model/common/request"
)

// RoleSearchParams 角色搜索参数
type RoleSearchParams struct {
    RoleName string `json:"roleName" form:"roleName"`
    RoleKey  string `json:"roleKey" form:"roleKey"`
    Status   string `json:"status" form:"status"`
    request.PageInfo
}

// CreateRoleRequest 创建角色参数
type CreateRoleRequest struct {
    RoleName    string  `json:"roleName" binding:"required"`
    RoleKey     string  `json:"roleKey" binding:"required"`
    RoleSort    int     `json:"roleSort"`
    Status      string  `json:"status"`
    DataScope   string  `json:"dataScope"`
    Description string  `json:"description"`
    MenuIds     []int64 `json:"menuIds"`
}

// UpdateRoleRequest 更新角色参数
type UpdateRoleRequest struct {
    ID          int64   `json:"roleId" binding:"required"`
    RoleName    string  `json:"roleName"`
    RoleKey     string  `json:"roleKey"`
    RoleSort    int     `json:"roleSort"`
    Status      string  `json:"status"`
    DataScope   string  `json:"dataScope"`
    Description string  `json:"description"`
    MenuIds     []int64 `json:"menuIds"`
}

// ChangeRoleStatusRequest 修改角色状态参数
type ChangeRoleStatusRequest struct {
    RoleId int64  `json:"roleId" binding:"required"`
    Status string `json:"status" binding:"required"`
}

// UpdateDataScopeRequest 修改数据权限参数
type UpdateDataScopeRequest struct {
    RoleId    int64   `json:"roleId" binding:"required"`
    DataScope string  `json:"dataScope" binding:"required"`
    DeptIds   []int64 `json:"deptIds"`
}

// AuthUserParams 用户授权参数
type AuthUserParams struct {
    RoleId  int64   `form:"roleId" binding:"required"`
    UserIds string  `form:"userIds"` // 逗号分隔的用户ID
}
```

### 三、Response 层完善

**文件：** `backend/model/system/response/sys_role.go`

```go
package response

// RoleResponse 角色列表响应
type RoleResponse struct {
    ID          int64  `json:"roleId"`
    RoleName    string `json:"roleName"`
    RoleKey     string `json:"roleKey"`
    RoleSort    int    `json:"roleSort"`
    DataScope   string `json:"dataScope"`
    Status      string `json:"status"`
    Description string `json:"description"`
    CreateAt    string `json:"createTime"`
}

// RoleDetailResponse 角色详情响应
type RoleDetailResponse struct {
    ID          int64   `json:"roleId"`
    RoleName    string `json:"roleName"`
    RoleKey     string `json:"roleKey"`
    RoleSort    int    `json:"roleSort"`
    DataScope   string `json:"dataScope"`
    Status      string `json:"status"`
    Description string  `json:"description"`
    MenuIds     []int64 `json:"menuIds"`
    DeptIds     []int64 `json:"deptIds"`
}

// RoleDeptTreeResponse 角色部门树响应
type RoleDeptTreeResponse struct {
    Depts        []DeptTreeItem `json:"depts"`
    CheckedKeys  []int64        `json:"checkedKeys"`
}

// DeptTreeItem 部门树节点
type DeptTreeItem struct {
    ID       int64          `json:"id"`
    Label    string         `json:"label"`
    Children []DeptTreeItem `json:"children,omitempty"`
}
```

### 四、Service 层完整实现

**文件：** `backend/service/system/sys_role.go`

需实现的方法：

```go
type RoleService struct{}

// GetRoleSelectList 获取角色选择框列表（已实现，需修复 status 查询）
func (r *RoleService) GetRoleSelectList() ([]SysResponse.RoleSimpleResponse, error)

// GetRoleList 获取角色列表（需修复字段映射）
func (r *RoleService) GetRoleList(params request.RoleSearchParams) ([]SysResponse.RoleResponse, int64, error)

// GetRoleDetail 获取角色详情
func (r *RoleService) GetRoleDetail(id int64) (*SysResponse.RoleDetailResponse, error)

// CreateRole 创建角色
func (r *RoleService) CreateRole(params *request.CreateRoleRequest) error

// UpdateRole 更新角色
func (r *RoleService) UpdateRole(params *request.UpdateRoleRequest) error

// DeleteRoles 批量删除角色
func (r *RoleService) DeleteRoles(ids []int64) error

// ChangeRoleStatus 修改角色状态
func (r *RoleService) ChangeRoleStatus(params *request.ChangeRoleStatusRequest) error

// UpdateDataScope 修改数据权限
func (r *RoleService) UpdateDataScope(params *request.UpdateDataScopeRequest) error

// GetRoleUserList 获取角色已授权用户列表
func (r *RoleService) GetRoleUserList(roleId int64, params request.PageInfo) ([]SysResponse.UserListResponse, int64, error)

// AuthUser 批量授权用户
func (r *RoleService) AuthUser(roleId int64, userIds []int64) error

// CancelAuthUser 批量取消授权用户
func (r *RoleService) CancelAuthUser(roleId int64, userIds []int64) error

// GetRoleDeptTree 获取角色部门树
func (r *RoleService) GetRoleDeptTree(roleId int64) (*SysResponse.RoleDeptTreeResponse, error)

// ExportRoleList 导出角色列表
func (r *RoleService) ExportRoleList(params request.RoleSearchParams) ([]byte, string, error)
```

### 五、API 层完整实现

**文件：** `backend/api/v1/system/sys_role.go`

```go
type RoleApi struct{}

// RoleSelectView 获取角色选择框列表
func (r *RoleApi) RoleSelectView(c *gin.Context)

// RoleListView 获取角色列表
func (r *RoleApi) RoleListView(c *gin.Context)

// RoleDetailView 获取角色详情
func (r *RoleApi) RoleDetailView(c *gin.Context)

// RoleCreateView 创建角色
func (r *RoleApi) RoleCreateView(c *gin.Context)

// RoleUpdateView 更新角色
func (r *RoleApi) RoleUpdateView(c *gin.Context)

// RoleDeleteView 删除角色
func (r *RoleApi) RoleDeleteView(c *gin.Context)

// RoleChangeStatusView 修改角色状态
func (r *RoleApi) RoleChangeStatusView(c *gin.Context)

// RoleUpdateDataScopeView 修改数据权限
func (r *RoleApi) RoleUpdateDataScopeView(c *gin.Context)

// RoleAuthUserListView 获取已授权用户列表
func (r *RoleApi) RoleAuthUserListView(c *gin.Context)

// RoleAuthUserView 批量授权用户
func (r *RoleApi) RoleAuthUserView(c *gin.Context)

// RoleCancelAuthUserView 批量取消授权
func (r *RoleApi) RoleCancelAuthUserView(c *gin.Context)

// RoleDeptTreeView 获取角色部门树
func (r *RoleApi) RoleDeptTreeView(c *gin.Context)

// RoleExportView 导出角色
func (r *RoleApi) RoleExportView(c *gin.Context)
```

### 六、Router 层完善

**文件：** `backend/router/system/sys_role.go`

```go
func (r *RoleRouter) InitRoleRouter(Router *gin.RouterGroup) {
    systemRouter := Router.Group("system")
    {
        roleRouter := systemRouter.Group("role")
        {
            roleRouter.GET("select", RoleApi.RoleSelectView)
            roleRouter.GET("list", RoleApi.RoleListView)
            roleRouter.GET(":id", RoleApi.RoleDetailView)
            roleRouter.POST("", RoleApi.RoleCreateView)
            roleRouter.PUT("", RoleApi.RoleUpdateView)
            roleRouter.DELETE(":ids", RoleApi.RoleDeleteView)
            roleRouter.PUT("changeStatus", RoleApi.RoleChangeStatusView)
            roleRouter.PUT("dataScope", RoleApi.RoleUpdateDataScopeView)
            roleRouter.GET("authUser/allocatedList", RoleApi.RoleAuthUserListView)
            roleRouter.PUT("authUser/selectAll", RoleApi.RoleAuthUserView)
            roleRouter.PUT("authUser/cancelAll", RoleApi.RoleCancelAuthUserView)
            roleRouter.GET("deptTree/:id", RoleApi.RoleDeptTreeView)
            roleRouter.POST("export", RoleApi.RoleExportView)
        }
    }
}
```

## 修改文件清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `backend/model/system/sys_role.go` | 修改 | 添加 DataScope 字段，修复 column 标签 |
| `backend/model/system/request/sys_role.go` | 修改 | 补充请求参数结构体 |
| `backend/model/system/response/sys_role.go` | 修改 | 补充响应结构体 |
| `backend/service/system/sys_role.go` | 修改 | 实现完整业务逻辑 |
| `backend/api/v1/system/sys_role.go` | 修改 | 实现完整 API 接口 |
| `backend/router/system/sys_role.go` | 修改 | 补充路由注册 |

## 测试验证

修复后需验证以下功能：

1. **角色列表**：分页查询、条件筛选
2. **角色详情**：获取单个角色信息
3. **创建角色**：新增角色并关联菜单
4. **编辑角色**：修改角色信息
5. **删除角色**：批量删除角色
6. **状态修改**：启用/停用角色
7. **数据权限**：修改角色数据范围
8. **用户授权**：为角色分配用户
9. **取消授权**：移除角色用户
10. **导出角色**：导出 Excel 文件

## 风险评估

- **中风险**：涉及多个文件的协同修改，需确保分层调用正确
- **数据库兼容**：需确认 `sys_roles` 表是否有 `data_scope` 字段，若无需执行迁移
- **回滚简单**：可通过 Git 回滚代码变更