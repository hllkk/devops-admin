package gateway

import "github.com/gin-gonic/gin"

// DeploymentRouter 模型部署路由(对齐前端 /gateway/model/deployment/* 资源)
type DeploymentRouter struct{}

// InitDeploymentRouter 挂在 PrivateGroup，鉴权/操作日志/数据权限由该组全局中间件统一处理。
func (r *DeploymentRouter) InitDeploymentRouter(Router *gin.RouterGroup) {
	g := Router.Group("gateway/model/deployment")
	{
		g.GET("list", deploymentApi.GetDeploymentList)       // 分页获取部署列表
		g.POST("test", deploymentApi.TestDeployment)         // 部署连通性测试
		g.POST("", deploymentApi.CreateDeployment)           // 新增部署
		g.PUT("", deploymentApi.UpdateDeployment)            // 修改部署
		g.DELETE(":ids", deploymentApi.BatchDeleteDeployments) // 批量删除部署
	}
}
