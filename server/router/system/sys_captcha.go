package system

import "github.com/gin-gonic/gin"

type CaptchaRouter struct{}

// InitCaptchaRouter 注册公开验证码路由(无需鉴权)：GET /auth/captcha。
// public 为 nil 时跳过(ModuleRouter 分组调用场景)。
func (r *CaptchaRouter) InitCaptchaRouter(public *gin.RouterGroup) {
	if public == nil {
		return
	}
	public.Group("auth").GET("captcha", captchaApi.Captcha)
}
