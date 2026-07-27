package system

import (
	"github.com/gin-gonic/gin"

	"github.com/hllkk/devops-admin/server/middleware"
)

// WecomRouter 企业微信扫码登录路由(全部公开:登录链路前置,免鉴权)。
//
// 路由清单(前缀 RouterPrefix,通常 /api/v1):
//   - GET /auth/wecomLogin          取二维码(sceneId+oauthUrl)
//   - GET /auth/qrCodeStatus        轮询扫码状态(高频,独立宽松限流)
//   - GET /wecomCallback            企微 OAuth2 回调(PC 扫码 + WebView 免登共用,企微服务器/webview 发起)
//   - GET /auth/wecomWebviewLogin   企微客户端 WebView 免登入口(302 跳转授权)
//
// 注:企业微信可信域名校验 /WW_verify_*.txt 已在 SettingRouter 注册,此处不重复。
type WecomRouter struct{}

func (w *WecomRouter) InitWecomRouter(PublicRouter *gin.RouterGroup) {
	pub := PublicRouter.Group("")
	// 取二维码(登录类敏感接口,走安全配置限流)
	pub.GET("/auth/wecomLogin", middleware.SecurityLimit(), wecomApi.QrCodeView)
	// 轮询状态(高频调用,独立宽松配额 60 次/min/IP,避免与登录失败限流冲突)
	pub.GET("/auth/qrCodeStatus",
		middleware.LimitConfig{
			GenerationKey: func(c *gin.Context) string { return "WecomQrStatus:" + c.ClientIP() },
			CheckOrMark:   middleware.CacheCheckOrMark,
			Expire:        60,
			Limit:         60,
		}.LimitWithTime(),
		wecomApi.QrCodeStatusView,
	)
	// 企微 OAuth2 回调(企微服务器/客户端发起,走安全配置限流)
	pub.GET("/wecomCallback", middleware.SecurityLimit(), wecomApi.WecomCallbackView)
	// 企微客户端 WebView 免登入口
	pub.GET("/auth/wecomWebviewLogin", middleware.SecurityLimit(), wecomApi.WecomWebviewLoginView)
}
