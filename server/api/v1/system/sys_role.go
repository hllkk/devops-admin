package system

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/response"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/excel"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// RoleApi 角色管理(对齐前端 /system/role/* 资源)
type RoleApi struct{}

// GetRoleList
// @Tags      SysRole
// @Summary   分页获取角色列表
// @Produce   application/json
// @Param     roleName  query  string  false  "角色名称(模糊匹配)"
// @Param     roleKey   query  string  false  "角色权限字符(模糊匹配)"
// @Param     status    query  string  false  "角色状态(精确 0正常1停用)"
// @Param     pageNum   query  int     true   "页码"
// @Param     pageSize  query  int     true   "每页大小"
// @Success   200  {object}  response.Response{data=response.PageResult{rows=[]system.SysRole},msg=string}
// @Router    /system/role/list [get]
func (a *RoleApi) GetRoleList(c *gin.Context) {
	var q systemReq.RoleSearch
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := roleService.GetRoleList(c.Request.Context(), q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("获取角色列表失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		Rows:     list,
		Total:    total,
		PageNum:  q.PageNum,
		PageSize: q.PageSize,
	}, "获取成功", c)
}

// CreateRole
// @Tags      SysRole
// @Summary   新增角色(含分配菜单)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  systemReq.RoleOperateParams  true  "角色信息(含 menuIds)"
// @Success   200   {object}  response.Response{data=bool,msg=string}
// @Router    /system/role [post]
func (a *RoleApi) CreateRole(c *gin.Context) {
	var req systemReq.RoleOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := roleService.CreateRole(c.Request.Context(), req, utils.GetUserID(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "新增成功", c)
}

// UpdateRole
// @Tags      SysRole
// @Summary   修改角色(全量替换分配菜单)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  systemReq.RoleOperateParams  true  "角色信息(含 roleId 与 menuIds)"
// @Success   200   {object}  response.Response{data=bool,msg=string}
// @Router    /system/role [put]
func (a *RoleApi) UpdateRole(c *gin.Context) {
	var req systemReq.RoleOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := roleService.UpdateRole(c.Request.Context(), req, utils.GetUserID(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "修改成功", c)
}

// UpdateRoleStatus
// @Tags      SysRole
// @Summary   修改角色状态
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  systemReq.RoleOperateParams  true  "角色信息(含 roleId 与 status)"
// @Success   200   {object}  response.Response{data=bool,msg=string}
// @Router    /system/role/changeStatus [put]
func (a *RoleApi) UpdateRoleStatus(c *gin.Context) {
	var req systemReq.RoleOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := roleService.UpdateRoleStatus(c.Request.Context(), req, utils.GetUserID(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "修改成功", c)
}

// BatchDeleteRole
// @Tags      SysRole
// @Summary   批量删除角色(已分配用户时禁删)
// @Produce   application/json
// @Param     ids  path  string  true  "角色ID列表(逗号分隔)"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /system/role/{ids} [delete]
func (a *RoleApi) BatchDeleteRole(c *gin.Context) {
	ids := make([]int64, 0, 4)
	for s := range strings.SplitSeq(c.Param("ids"), ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			response.FailWithMessage("无效的角色ID: "+s, c)
			return
		}
		ids = append(ids, n)
	}
	if err := roleService.DeleteRole(c.Request.Context(), ids); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "删除成功", c)
}

// GetAllocatedUserList
// @Tags      SysRole
// @Summary   获取角色已分配用户列表
// @Produce   application/json
// @Param     roleId       query  int     true   "角色ID"
// @Param     userName     query  string  false  "用户名(模糊匹配)"
// @Param     phonenumber  query  string  false  "手机号(模糊匹配)"
// @Param     pageNum      query  int     true   "页码"
// @Param     pageSize     query  int     true   "每页大小"
// @Success   200  {object}  response.Response{data=response.PageResult{rows=[]system.SysUser},msg=string}
// @Router    /system/role/authUser/allocatedList [get]
func (a *RoleApi) GetAllocatedUserList(c *gin.Context) {
	var q systemReq.RoleUserSearch
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := roleService.GetAllocatedUserList(c.Request.Context(), q)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		Rows:     list,
		Total:    total,
		PageNum:  q.PageNum,
		PageSize: q.PageSize,
	}, "获取成功", c)
}

// AuthUserSelectAll
// @Tags      SysRole
// @Summary   批量给角色授权用户
// @Produce   application/json
// @Param     roleId   query  int     true  "角色ID"
// @Param     userIds  query  string  true  "用户ID列表(逗号分隔)"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /system/role/authUser/selectAll [put]
func (a *RoleApi) AuthUserSelectAll(c *gin.Context) {
	var p systemReq.RoleAuthUserParams
	if err := c.ShouldBindQuery(&p); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	userIds, ok := parseUserIds(p.UserIds, c)
	if !ok {
		return
	}
	if err := roleService.AuthUserSelectAll(c.Request.Context(), p.RoleId, userIds); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "授权成功", c)
}

// AuthUserCancelAll
// @Tags      SysRole
// @Summary   批量取消角色用户授权
// @Produce   application/json
// @Param     roleId   query  int     true  "角色ID"
// @Param     userIds  query  string  true  "用户ID列表(逗号分隔)"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /system/role/authUser/cancelAll [put]
func (a *RoleApi) AuthUserCancelAll(c *gin.Context) {
	var p systemReq.RoleAuthUserParams
	if err := c.ShouldBindQuery(&p); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	userIds, ok := parseUserIds(p.UserIds, c)
	if !ok {
		return
	}
	if err := roleService.AuthUserCancelAll(c.Request.Context(), p.RoleId, userIds); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "取消授权成功", c)
}

// parseUserIds 解析逗号分隔的用户ID;失败时已写错误响应并返回 (nil,false)。
func parseUserIds(raw string, c *gin.Context) ([]int64, bool) {
	ids := make([]int64, 0, 4)
	for s := range strings.SplitSeq(raw, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			response.FailWithMessage("无效的用户ID: "+s, c)
			return nil, false
		}
		ids = append(ids, n)
	}
	return ids, true
}

// ExportRole
// @Tags      SysRole
// @Summary   导出角色(Excel)
// @Router    /system/role/export [post]
func (a *RoleApi) ExportRole(c *gin.Context) {
	var q systemReq.RoleSearch
	if err := c.ShouldBind(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, err := roleService.ExportRoleList(c.Request.Context(), q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("导出角色失败")
		response.FailWithMessage("导出失败", c)
		return
	}
	buf, err := excel.Export(list, roleHeaders, "角色")
	if err != nil {
		response.FailWithMessage("导出失败", c)
		return
	}
	writeXlsx(c, "角色列表", buf)
}

// UpdateRoleDataScope
// @Tags      SysRole
// @Summary   分配角色数据权限
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  systemReq.RoleOperateParams  true  "角色信息(含 roleId/dataScope/deptIds)"
// @Success   200   {object}  response.Response{data=bool,msg=string}
// @Router    /system/role/dataScope [put]
func (a *RoleApi) UpdateRoleDataScope(c *gin.Context) {
	var req systemReq.RoleOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := roleService.UpdateRoleDataScope(c.Request.Context(), req, utils.GetUserID(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "修改成功", c)
}

// GetRoleDeptTreeSelect
// @Tags      SysRole
// @Summary   获取角色数据权限部门树(含已选部门)
// @Produce   application/json
// @Param     roleId  path  int  true  "角色ID"
// @Success   200  {object}  response.Response{data=system.RoleDeptTreeSelect,msg=string}
// @Router    /system/role/deptTree/{roleId} [get]
func (a *RoleApi) GetRoleDeptTreeSelect(c *gin.Context) {
	roleId, err := strconv.ParseInt(c.Param("roleId"), 10, 64)
	if err != nil {
		response.FailWithMessage("无效的角色ID", c)
		return
	}
	result, err := roleService.GetRoleDeptTreeSelect(c.Request.Context(), roleId)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "获取成功", c)
}
