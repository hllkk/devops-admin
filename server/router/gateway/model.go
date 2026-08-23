package gateway

import "github.com/gin-gonic/gin"

// ModelRouter 模型管理路由(对齐前端 /gateway/model/* 资源)
type ModelRouter struct{}

// InitModelRouter 挂在 PrivateGroup，鉴权/操作日志/数据权限由该组全局中间件统一处理。
// 静态段(active/publish)书写在 :id 之前注册。
func (r *ModelRouter) InitModelRouter(Router *gin.RouterGroup) {
	g := Router.Group("gateway/model")
	{
		g.GET("list", modelApi.GetModelList)          // 分页获取模型列表
		g.GET("active", modelApi.GetActiveModels)     // 对外激活模型列表(home/AiKey 用)
		g.GET("publish/:id", modelApi.GetModelPublish) // 模型发布设置
		g.GET(":id", modelApi.GetModel)               // 模型详情(含部署列表)
		g.POST("", modelApi.CreateModel)              // 新增模型
		g.PUT("", modelApi.UpdateModel)               // 修改模型(改名/改类级联)
		g.PUT("publish", modelApi.PublishModel)       // 更新发布设置
		g.DELETE(":ids", modelApi.BatchDeleteModels)  // 批量删除模型(软删三连)
	}
}
