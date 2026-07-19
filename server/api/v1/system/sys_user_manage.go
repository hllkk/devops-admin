package system

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/response"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// UserApi 用户管理(对齐前端 /system/user/* 管理员侧资源;auth 链路的 Register/GetUserInfo 仍在 BaseApi)
type UserApi struct{}

// GetUserList
// @Tags      SysUser
// @Summary   分页获取用户列表
// @Produce   application/json
// @Param     deptId       query  int     false  "主部门ID(精确)"
// @Param     userName     query  string  false  "用户名(模糊匹配)"
// @Param     nickName     query  string  false  "昵称(模糊匹配)"
// @Param     phonenumber  query  string  false  "手机号(模糊匹配)"
// @Param     status       query  string  false  "状态(精确 0正常1停用)"
// @Param     roleId       query  int     false  "角色ID(精确)"
// @Param     pageNum      query  int     true   "页码"
// @Param     pageSize     query  int     true   "每页大小"
// @Success   200  {object}  response.Response{data=response.PageResult{rows=[]system.SysUser},msg=string}
// @Router    /system/user/list [get]
func (a *UserApi) GetUserList(c *gin.Context) {
	var q systemReq.UserSearch
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := userService.GetList(c.Request.Context(), q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("获取用户列表失败")
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

// GetDeptUserList
// @Tags      SysUser
// @Summary   获取部门下用户列表(部门负责人选择用)
// @Produce   application/json
// @Param     deptId  path  int  true  "部门ID"
// @Success   200  {object}  response.Response{data=[]system.SysUser,msg=string}
// @Router    /system/user/list/dept/{deptId} [get]
func (a *UserApi) GetDeptUserList(c *gin.Context) {
	deptId, err := strconv.ParseInt(c.Param("deptId"), 10, 64)
	if err != nil {
		response.FailWithMessage("无效的部门ID", c)
		return
	}
	list, err := userService.GetDeptUserList(c.Request.Context(), deptId)
	if err != nil {
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(list, "获取成功", c)
}

// GetUserDetail
// @Tags      SysUser
// @Summary   获取用户详情(postIds/roleIds/roles)
// @Produce   application/json
// @Param     userId  path  int  true  "用户ID"
// @Success   200  {object}  response.Response{data=system.UserInfo,msg=string}
// @Router    /system/user/{userId} [get]
func (a *UserApi) GetUserDetail(c *gin.Context) {
	userId, err := strconv.ParseInt(c.Param("userId"), 10, 64)
	if err != nil {
		response.FailWithMessage("无效的用户ID", c)
		return
	}
	info, err := userService.GetDetail(c.Request.Context(), userId)
	if err != nil {
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(info, "获取成功", c)
}

// GetDeptTree
// @Tags      SysUser
// @Summary   获取用户页部门树(复用部门模块)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=[]system.DeptTreeNode,msg=string}
// @Router    /system/user/deptTree [get]
func (a *UserApi) GetDeptTree(c *gin.Context) {
	tree, err := departmentService.GetDeptTree(c.Request.Context())
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("获取用户部门树失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(tree, "获取成功", c)
}

// CreateUser
// @Tags      SysUser
// @Summary   新增用户(含分配角色/岗位)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  systemReq.UserOperateParams  true  "用户信息(含 roleIds/postIds)"
// @Success   200   {object}  response.Response{data=bool,msg=string}
// @Router    /system/user [post]
func (a *UserApi) CreateUser(c *gin.Context) {
	var req systemReq.UserOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := userService.Create(c.Request.Context(), req, utils.GetUserID(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "新增成功", c)
}

// UpdateUser
// @Tags      SysUser
// @Summary   修改用户(全量替换角色/岗位)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  systemReq.UserOperateParams  true  "用户信息(含 userId、roleIds/postIds)"
// @Success   200   {object}  response.Response{data=bool,msg=string}
// @Router    /system/user [put]
func (a *UserApi) UpdateUser(c *gin.Context) {
	var req systemReq.UserOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := userService.Update(c.Request.Context(), req, utils.GetUserID(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "修改成功", c)
}

// UpdateUserStatus
// @Tags      SysUser
// @Summary   修改用户状态
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  systemReq.UserOperateParams  true  "用户信息(含 userId 与 status)"
// @Success   200   {object}  response.Response{data=bool,msg=string}
// @Router    /system/user/changeStatus [put]
func (a *UserApi) UpdateUserStatus(c *gin.Context) {
	var req systemReq.UserOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := userService.UpdateStatus(c.Request.Context(), req, utils.GetUserID(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "修改成功", c)
}

// ResetUserPwd
// @Tags      SysUser
// @Summary   重置用户密码
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  systemReq.ResetUserPwdParams  true  "用户ID + 新密码"
// @Success   200   {object}  response.Response{data=bool,msg=string}
// @Router    /system/user/resetPwd [put]
func (a *UserApi) ResetUserPwd(c *gin.Context) {
	var req systemReq.ResetUserPwdParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := userService.ResetPwd(c.Request.Context(), req, utils.GetUserID(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "重置成功", c)
}

// BatchDeleteUser
// @Tags      SysUser
// @Summary   批量删除用户
// @Produce   application/json
// @Param     userIds  path  string  true  "用户ID列表(逗号分隔)"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /system/user/{userIds} [delete]
func (a *UserApi) BatchDeleteUser(c *gin.Context) {
	ids := make([]int64, 0, 4)
	for s := range strings.SplitSeq(c.Param("userIds"), ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			response.FailWithMessage("无效的用户ID: "+s, c)
			return
		}
		ids = append(ids, n)
	}
	if err := userService.Delete(c.Request.Context(), ids); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "删除成功", c)
}
