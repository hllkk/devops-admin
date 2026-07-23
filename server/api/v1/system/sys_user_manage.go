package system

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/response"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/excel"
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

// ExportUser
// @Tags      SysUser
// @Summary   导出用户(按列表查询条件,Excel)
// @Produce   application/octet-stream
// @Param     userName     query  string  false  "用户名(模糊)"
// @Param     nickName     query  string  false  "昵称(模糊)"
// @Param     phonenumber  query  string  false  "手机号(模糊)"
// @Param     status       query  string  false  "状态(0正常 1停用)"
// @Param     deptId       query  int     false  "主部门ID"
// @Param     roleId       query  int     false  "角色ID"
// @Router    /system/user/export [post]
func (a *UserApi) ExportUser(c *gin.Context) {
	// 前端 useDownload 以 application/x-www-form-urlencoded 提交,用 ShouldBind 绑定表单体
	var q systemReq.UserSearch
	if err := c.ShouldBind(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, err := userService.ExportList(c.Request.Context(), q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("导出用户失败")
		response.FailWithMessage("导出失败", c)
		return
	}
	buf, err := excel.Export(list, userHeaders, "用户")
	if err != nil {
		response.FailWithMessage("导出失败", c)
		return
	}
	writeXlsx(c, "用户列表", buf)
}

// ImportTemplate
// @Tags      SysUser
// @Summary   下载用户导入模板(仅表头的 xlsx)
// @Router    /system/user/importTemplate [post]
func (a *UserApi) ImportTemplate(c *gin.Context) {
	buf, err := excel.ExportTemplate(userImportHeaders, "用户")
	if err != nil {
		response.FailWithMessage("生成模板失败", c)
		return
	}
	writeXlsx(c, "用户导入模板", buf)
}

// ImportUser
// @Tags      SysUser
// @Summary   导入用户(Excel)
// @Accept    multipart/form-data
// @Param     file            formData  file  true   "xlsx 文件"
// @Param     updateSupport   formData  bool  false  "是否更新已存在用户"
// @Success   200  {object}  response.Response{msg=string}
// @Router    /system/user/importData [post]
func (a *UserApi) ImportUser(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.FailWithMessage("文件获取失败", c)
		return
	}
	rows, err := excel.Parse(file, userImportHeaders)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	updateSupport := c.PostForm("updateSupport") == "true"
	insert, updateCnt, skip, failCnt, failMsgs := userService.ImportUsers(c.Request.Context(), rows, updateSupport, utils.GetUserID(c))
	// 汇总信息回写 msg,前端 UserImportModal 以 v-html 渲染(失败明细逐行 <br> 拼接)
	msg := fmt.Sprintf("导入完成:新增 %d,更新 %d,跳过 %d,失败 %d", insert, updateCnt, skip, failCnt)
	if len(failMsgs) > 0 {
		msg += "<br>" + strings.Join(failMsgs, "<br>")
	}
	response.OkWithMessage(msg, c)
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

// ChangeMyPassword
// @Tags      SysUser
// @Summary   当前用户修改密码(密码过期强制改密入口)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  systemReq.ChangeMyPasswordParams  true  "旧密码 + 新密码"
// @Success   200   {object}  response.Response{data=bool,msg=string}
// @Router    /system/user/profile/updatePwd [put]
func (a *UserApi) ChangeMyPassword(c *gin.Context) {
	var req systemReq.ChangeMyPasswordParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	user, err := userService.ChangeMyPassword(c.Request.Context(), utils.GetUserID(c), req.OldPassword, req.NewPassword)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	// 改密成功后重发 MustChangePwd=false 的 token,否则用户仍带过期标记,继续被 guard 拦截。
	token, claims, err := utils.LoginTokenWithExpire(&user, false)
	if err != nil {
		response.FailWithMessage("密码已修改,但下发新 token 失败,请重新登录", c)
		return
	}
	expire := int(claims.RegisteredClaims.ExpiresAt.Unix() - time.Now().Unix())
	utils.SetToken(c, token, expire)
	response.OkWithDetailed(true, "密码修改成功", c)
}

// UpdateMyProfile
// @Tags      SysUser
// @Summary   当前用户修改基本资料(昵称/邮箱/手机号/性别)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  systemReq.UpdateMyProfileParams  true  "基本资料"
// @Success   200   {object}  response.Response{data=bool,msg=string}
// @Router    /system/user/profile [put]
func (a *UserApi) UpdateMyProfile(c *gin.Context) {
	var req systemReq.UpdateMyProfileParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := userService.UpdateMyProfile(c.Request.Context(), utils.GetUserID(c), req); err != nil {
		response.FailWithMessage("修改失败", c)
		return
	}
	response.OkWithDetailed(true, "修改成功", c)
}

// UpdateMyAvatar
// @Tags      SysUser
// @Summary   当前用户上传头像
// @Accept    multipart/form-data
// @Produce   application/json
// @Param     avatarfile  formData  file  true  "头像图片(jpg/jpeg/png/gif/webp)"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /system/user/profile/avatar [post]
func (a *UserApi) UpdateMyAvatar(c *gin.Context) {
	file, err := c.FormFile("avatarfile")
	if err != nil {
		response.FailWithMessage("文件获取失败", c)
		return
	}
	if _, err := userService.UpdateMyAvatar(c.Request.Context(), utils.GetUserID(c), file); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "头像更新成功", c)
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
