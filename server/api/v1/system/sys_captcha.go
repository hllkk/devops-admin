package system

import (
	"github.com/gin-gonic/gin"

	"github.com/hllkk/devops-admin/server/model/common/response"
)

type BaseApi struct{}

// Captcha
// @Tags      Base
// @Summary  生成验证码(go-captcha 行为验证)
// @Produce  application/json
// @Param    username  query  string  false  "用户名(触发阈值判断)"
// @Success  200  {object}  response.Response{data=system.CaptchaResult,msg=string}
// @Router   /base/captcha [get]
func (b *BaseApi) Captcha(c *gin.Context) {
	username := c.Query("username")
	result, err := captchaService.Get(c.Request.Context(), username, c.ClientIP())
	if err != nil {
		response.FailWithMessage("验证码获取失败", c)
		return
	}
	response.OkWithDetailed(result, "验证码获取成功", c)
}
