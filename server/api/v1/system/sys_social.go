package system

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/response"
	"github.com/hllkk/devops-admin/server/model/system"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
	systemSvc "github.com/hllkk/devops-admin/server/service/system"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// SocialApi 第三方账号绑定/社交登录接口(对齐前端 service/api/system/social.ts + auth.ts)
type SocialApi struct{}

// GetAuthURL
// @Tags      Social
// @Summary   获取第三方授权URL
// @Produce   application/json
// @Param     source  path  string  true  "来源:wechat_open/gitee/github"
// @Param     domain  query string  false "回调域名(默认取 Host)"
// @Success   200  {object}  response.Response{data=string,msg=string}  "授权URL字符串"
// @Router    /auth/binding/{source} [get]
func (a *SocialApi) GetAuthURL(c *gin.Context) {
	source := c.Param("source")
	domain := c.Query("domain")
	if domain == "" {
		domain = c.Request.Host
	}
	// PublicGroup 免鉴权:安静尝试解析 x-token cookie,有效→绑定意图,无效→登录意图
	// (不用 utils.GetClaims,避免未登录用户每次请求刷一条 error 日志)
	var jwtUserId int64
	if token, err := c.Cookie("x-token"); err == nil && token != "" {
		if claims, err := utils.NewJWT().ParseToken(token); err == nil && claims != nil {
			jwtUserId = claims.BaseClaims.ID
		}
	}
	authURL, err := socialService.GetAuthURL(c.Request.Context(), source, domain, jwtUserId)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithData(authURL, c)
}

// Callback
// @Tags      Social
// @Summary   社交登录/绑定回调(交换code+处理绑定或登录)
// @Produce   application/json
// @Param     data  body  systemReq.SocialLoginForm  true  "socialCode/socialState/source/grantType"
// @Success   200  {object}  response.Response  "绑定成功返回提示;登录成功签发JWT(httpOnly cookie)并返回 LoginResponse"
// @Router    /auth/social/callback [post]
func (a *SocialApi) Callback(c *gin.Context) {
	var req systemReq.SocialLoginForm
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	var jwtUserId int64
	if token, err := c.Cookie("x-token"); err == nil && token != "" {
		if claims, err := utils.NewJWT().ParseToken(token); err == nil && claims != nil {
			jwtUserId = claims.BaseClaims.ID
		}
	}
	result, err := socialService.Callback(c.Request.Context(), req, jwtUserId)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if !result.IsLogin {
		response.OkWithMessage("绑定成功", c)
		return
	}
	// 登录成功:按现有 Login 链路算密码过期标记并签发 JWT
	needChange := systemSvc.IsPasswordExpired(
		c.Request.Context(),
		result.User.PasswordUpdatedAt,
		securityConfigService.Current(c.Request.Context()),
		time.Now(),
	)
	(&BaseApi{}).TokenNext(c, *result.User, needChange)
}

// List
// @Tags      Social
// @Summary   当前用户已绑定的第三方账号列表
// @Produce   application/json
// @Success   200  {object}  response.Response{data=[]system.SysSocial,msg=string}  "绑定列表(token字段不返回)"
// @Router    /system/social/list [get]
func (a *SocialApi) List(c *gin.Context) {
	userId := utils.GetUserID(c)
	var list []system.SysSocial
	var err error
	list, err = socialService.List(c.Request.Context(), userId)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("social").Err(err).Error("获取社交绑定列表失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(list, "获取成功", c)
}

// Unbind
// @Tags      Social
// @Summary   解绑指定第三方账号(至少保留一种登录方式)
// @Produce   application/json
// @Param     id  path  string  true  "关联记录ID"
// @Success   200  {object}  response.Response{msg=string}  "解绑成功"
// @Router    /auth/unlock/{id} [delete]
func (a *SocialApi) Unbind(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}
	userId := utils.GetUserID(c)
	if err := socialService.Unbind(c.Request.Context(), userId, id); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("解绑成功", c)
}
