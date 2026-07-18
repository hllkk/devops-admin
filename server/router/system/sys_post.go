package system

import (
	"github.com/gin-gonic/gin"
)

// PostRouter 岗位管理路由(对齐前端 /system/post/* 资源)
type PostRouter struct{}

// InitPostRouter 岗位相关路由挂在 PrivateGroup 下,鉴权与操作日志由该组全局中间件统一处理。
func (p *PostRouter) InitPostRouter(Router *gin.RouterGroup) {
	postRouter := Router.Group("system/post")
	{
		postRouter.GET("list", postApi.GetPostList)           // 分页获取岗位列表
		postRouter.GET("optionselect", postApi.GetPostOption) // 获取岗位选择框列表
		postRouter.GET("deptTree", postApi.GetPostDeptTree)   // 获取岗位页部门树
		postRouter.POST("", postApi.CreatePost)               // 新增岗位
		postRouter.PUT("", postApi.UpdatePost)                // 修改岗位
		postRouter.DELETE(":ids", postApi.BatchDeletePost)    // 批量删除岗位
	}
}
