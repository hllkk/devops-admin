package gateway

import "github.com/gin-gonic/gin"

// ProviderRouter AI 供应商路由(对齐前端 /gateway/provider/* 资源)
type ProviderRouter struct{}

// InitProviderRouter 挂在 PrivateGroup，鉴权/操作日志/数据权限由该组全局中间件统一处理。
func (p *ProviderRouter) InitProviderRouter(Router *gin.RouterGroup) {
	r := Router.Group("gateway/provider")
	{
		r.GET("list", providerApi.GetProviderList)     // 分页获取供应商列表
		r.GET(":id", providerApi.GetProvider)         // 供应商详情
		r.POST("", providerApi.CreateProvider)        // 新增供应商
		r.PUT("", providerApi.UpdateProvider)         // 修改供应商
		r.DELETE(":ids", providerApi.BatchDeleteProvider) // 批量删除供应商
		// 套餐余量旁路(厂商侧只读快照,api_prefix /gateway/provider/* 已覆盖)
		r.GET(":id/balance", providerBalanceApi.GetProviderBalances)         // 余量明细(坐席/共享包)
		r.GET(":id/balance-config", providerBalanceApi.GetBalanceConfig)     // 采集配置(掩码)
		r.PUT(":id/balance-config", providerBalanceApi.SaveBalanceConfig)    // 保存采集配置(AK/SK)
		r.POST(":id/balance-sync", providerBalanceApi.SyncProviderBalance)   // 手动同步
	}
}
