package system

import (
	"github.com/gin-gonic/gin"

	"github.com/hllkk/devops-admin/server/model/common/response"
)

type CaptchaApi struct{}

// Captcha
// @Tags        Auth
// @Summary     生成验证码
// @Description 按触发策略决定是否返回验证码：captchaEnabled=false 表示当前无需验证码；为 true 时返回 type/captchaId 与对应图片。type=image 为传统图形验证码，click/slide/rotate 为 go-captcha 行为验证码。
// @Param       username  query     string  false  "用户名(阈值模式下用于判断账号是否触发验证码)"
// @Produce     application/json
// @Success     200  {object}  response.Response{data=system.CaptchaResult,msg=string}  "captchaEnabled/type/captchaId/masterImage/tileImage/thumbImage"
// @Router      /auth/captcha [get]
func (a *CaptchaApi) Captcha(c *gin.Context) {
	username := c.Query("username")
	res, err := captchaService.Get(c.Request.Context(), username, c.ClientIP())
	if err != nil {
		response.FailWithMessage("验证码生成失败", c)
		return
	}
	response.OkWithDetailed(res, "获取成功", c)
}
