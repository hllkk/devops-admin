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

// MenuApi 菜单管理(对齐前端 /system/menu/* 资源)
type MenuApi struct{}

// GetMenuList
// @Tags      SysMenu
// @Summary   获取菜单列表(树形平表,前端组装树)
// @Produce   application/json
// @Param     menuName  query  string  false  "菜单名称(模糊匹配)"
// @Param     status    query  string  false  "菜单状态(精确 0正常1停用)"
// @Param     menuType  query  string  false  "菜单类型(精确 M目录C菜单F按钮)"
// @Param     parentId  query  int     false  "父菜单ID(精确,查按钮列表用)"
// @Success   200  {object}  response.Response{data=[]system.SysMenu,msg=string}
// @Router    /system/menu/list [get]
func (a *MenuApi) GetMenuList(c *gin.Context) {
	var q systemReq.MenuSearch
	_ = c.ShouldBindQuery(&q)
	list, err := menuService.GetMenuList(c.Request.Context(), q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("获取菜单列表失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(list, "获取成功", c)
}

// GetMenuTreeSelect
// @Tags      SysMenu
// @Summary   获取菜单树(已组装,树选择用)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=[]system.MenuTreeSelectNode,msg=string}
// @Router    /system/menu/treeselect [get]
func (a *MenuApi) GetMenuTreeSelect(c *gin.Context) {
	list, err := menuService.GetMenuTreeSelect(c.Request.Context())
	if err != nil {
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(list, "获取成功", c)
}

// GetRoleMenuTreeSelect
// @Tags      SysMenu
// @Summary   获取角色菜单权限树(角色分配菜单回显用)
// @Produce   application/json
// @Param     roleId  path  int  true  "角色ID"
// @Success   200  {object}  response.Response{data=system.RoleMenuTreeSelect,msg=string}
// @Router    /system/menu/roleMenuTreeselect/{roleId} [get]
func (a *MenuApi) GetRoleMenuTreeSelect(c *gin.Context) {
	roleId, err := strconv.ParseInt(c.Param("roleId"), 10, 64)
	if err != nil {
		response.FailWithMessage("无效的角色ID", c)
		return
	}
	result, err := menuService.GetRoleMenuTreeSelect(c.Request.Context(), roleId)
	if err != nil {
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(result, "获取成功", c)
}

// CreateMenu
// @Tags      SysMenu
// @Summary   新增菜单
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  systemReq.MenuOperateParams  true  "菜单信息"
// @Success   200   {object}  response.Response{data=bool,msg=string}
// @Router    /system/menu [post]
func (a *MenuApi) CreateMenu(c *gin.Context) {
	var req systemReq.MenuOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := menuService.CreateMenu(c.Request.Context(), req, utils.GetUserID(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "新增成功", c)
}

// UpdateMenu
// @Tags      SysMenu
// @Summary   修改菜单
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  systemReq.MenuOperateParams  true  "菜单信息(含 menuId)"
// @Success   200   {object}  response.Response{data=bool,msg=string}
// @Router    /system/menu [put]
func (a *MenuApi) UpdateMenu(c *gin.Context) {
	var req systemReq.MenuOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := menuService.UpdateMenu(c.Request.Context(), req, utils.GetUserID(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "修改成功", c)
}

// DeleteMenu
// @Tags      SysMenu
// @Summary   删除菜单(存在子菜单或已分配角色时禁删)
// @Produce   application/json
// @Param     menuId  path  int  true  "菜单ID"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /system/menu/{menuId} [delete]
func (a *MenuApi) DeleteMenu(c *gin.Context) {
	menuId, err := strconv.ParseInt(c.Param("menuId"), 10, 64)
	if err != nil {
		response.FailWithMessage("无效的菜单ID", c)
		return
	}
	if err := menuService.DeleteMenu(c.Request.Context(), menuId); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "删除成功", c)
}

// CascadeDeleteMenu
// @Tags      SysMenu
// @Summary   级联删除菜单(含子孙,清理角色关联)
// @Produce   application/json
// @Param     menuIds  path  string  true  "菜单ID列表(逗号分隔)"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /system/menu/cascade/{menuIds} [delete]
func (a *MenuApi) CascadeDeleteMenu(c *gin.Context) {
	ids := make([]int64, 0, 4)
	for s := range strings.SplitSeq(c.Param("menuIds"), ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			response.FailWithMessage("无效的菜单ID: "+s, c)
			return
		}
		ids = append(ids, n)
	}
	if err := menuService.CascadeDeleteMenu(c.Request.Context(), ids); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "删除成功", c)
}
