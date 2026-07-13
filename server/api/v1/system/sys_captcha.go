package system

import (
	"github.com/gin-gonic/gin"

	"github.com/hllkk/devops-admin/server/model/common/response"
)

type BaseApi struct{}

// Captcha
// @Tags      Base
// @Summary   生成验证码
// @Produce   application/json
// @Success   200  {object}  response.Response{data=object,msg=string}  "captchaEnabled, uuid, img"
// @Router    /auth/code [post]
// Captcha 对齐前端 Api.Auth.CaptchaCode{captchaEnabled, uuid, img}。
// 本期占位：captchaEnabled=false（不启用验证码），真验证码生成（base64 + store.Verify）后续实现。
func (b *BaseApi) Captcha(c *gin.Context) {
	response.OkWithDetailed(gin.H{
		"captchaEnabled": false,
	}, "获取成功", c)
}
