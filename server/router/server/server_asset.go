package server

import "github.com/gin-gonic/gin"

// AssetRouter 服务器模块统一资产路由(对齐前端 /server/asset/* 资源)
type AssetRouter struct{}

// InitAssetRouter 挂在 PrivateGroup，鉴权/操作日志/数据权限由该组全局中间件统一处理。
func (a *AssetRouter) InitAssetRouter(Router *gin.RouterGroup) {
	r := Router.Group("server/asset")
	{
		// 静态段(list/overview)注册在 :id 前(gin 路由树静态段优先匹配)
		r.GET("list", assetApi.GetAssetList)         // 分页获取资产列表
		r.GET("overview", assetApi.GetAssetOverview) // 服务器概览统计(顶层概览页)
		r.GET(":id", assetApi.GetAsset)              // 资产详情
		r.POST("", assetApi.CreateAsset)             // 新增资产
		r.PUT("", assetApi.UpdateAsset)              // 修改资产
		r.DELETE(":ids", assetApi.BatchDeleteAsset)  // 批量删除资产
	}
}
