package system

import (
	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/response"
	system "github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/utils"
)

// RouteApi 动态路由下发(/route,挂 PrivateGroup)。前端 Soybean 静态/动态模式均由此取路由。
type RouteApi struct{}

// GetConstantRoutes 常量路由
// @Tags     Route
// @Summary  常量路由
// @Produce  application/json
// @Success  200  {object}  response.Response{data=[]system.MenuRoute,msg=string}
// @Router   /route/getConstantRoutes [get]
func (a *RouteApi) GetConstantRoutes(c *gin.Context) {
	// dynamic 模式下 constant(login/404/init 等 _builtin)由前端静态生成(fallback),后端返回空数组。
	response.OkWithDetailed([]system.MenuRoute{}, "获取成功", c)
}

// GetUserRoutes 用户动态路由
// @Tags     Route
// @Summary  用户动态路由(按角色过滤后转换的 MenuRoute 树 + home)
// @Produce  application/json
// @Success  200  {object}  response.Response{data=system.UserRoute,msg=string}
// @Router   /route/getUserRoutes [get]
func (a *RouteApi) GetUserRoutes(c *gin.Context) {
	userId := utils.GetUserID(c)
	data, err := routeService.GetUserRoutes(c.Request.Context(), userId)
	if err != nil {
		response.FailWithMessage("获取路由失败", c)
		return
	}
	response.OkWithDetailed(data, "获取成功", c)
}

// IsRouteExist 路由是否存在
// @Tags     Route
// @Summary  路由是否存在(前端路由守卫用)
// @Produce  application/json
// @Param    routeName  query     string  true  "路由名(RouteKey)"
// @Success  200        {object}  response.Response{data=bool,msg=string}
// @Router   /route/isRouteExist [get]
func (a *RouteApi) IsRouteExist(c *gin.Context) {
	routeName := c.Query("routeName")
	userId := utils.GetUserID(c)
	exist, err := routeService.IsRouteExist(c.Request.Context(), userId, routeName)
	if err != nil {
		response.FailWithMessage("查询失败", c)
		return
	}
	response.OkWithDetailed(exist, "获取成功", c)
}
