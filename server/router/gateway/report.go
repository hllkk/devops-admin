package gateway

import "github.com/gin-gonic/gin"

// ReportRouter 效能报告路由(对齐前端 /gateway/report/* 资源)
type ReportRouter struct{}

// InitReportRouter 挂在 PrivateGroup，鉴权/操作日志由该组全局中间件统一处理。
// 接口权限走菜单 api_prefix(route.ai-audit_report)：user 角色不授，管理员/决策层视角。
// export 用 POST 对齐前端 useDownload 的表单下载模式。
func (r *ReportRouter) InitReportRouter(Router *gin.RouterGroup) {
	g := Router.Group("gateway/report")
	{
		g.GET("list", reportApi.GetReportList)             // 分页列表
		g.GET(":id", reportApi.GetReport)                  // 详情(结构化内容)
		g.POST("generate", reportApi.GenerateReport)       // 手动生成
		g.POST("export/:id", reportApi.ExportReport)       // Excel 导出(流下载)
	}
}
