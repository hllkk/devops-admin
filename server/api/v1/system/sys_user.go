package system

import (
	"strconv"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/common/response"
	"github.com/hllkk/devops-admin/server/model/system"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
	systemRes "github.com/hllkk/devops-admin/server/model/system/response"
	systemSvc "github.com/hllkk/devops-admin/server/service/system"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/logger"
	"github.com/redis/go-redis/v9"
)

// Login
// @Tags     Base
// @Summary  用户登录
// @Produce  application/json
// @Param    data  body      systemReq.Login                                        true  "用户名, 密码, 验证码"
// @Success  200   {object}  response.Response{data=systemRes.LoginResponse,msg=string}
// @Router   /base/login [post]
func (b *BaseApi) Login(c *gin.Context) {
	var l systemReq.Login
	if err := c.ShouldBindJSON(&l); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	cfg := securityConfigService.Current(c.Request.Context())

	// 1. 账号锁定检查
	if cfg.LoginFailLockCount > 0 && systemSvc.IsAccountLocked(c.Request.Context(), l.Username) {
		response.FailWithMessage("账号已锁定，请 "+strconv.Itoa(cfg.LoginFailLockTime)+" 分钟后再试", c)
		_ = loginLogService.CreateLoginLog(c.Request.Context(), system.SysLoginLog{
			UserName: l.Username, Ipaddr: c.ClientIP(), Browser: c.Request.UserAgent(),
			Status: "1", Msg: "账号已锁定",
		})
		return
	}

	// 2. 验证码检查（go-captcha：CaptchaService 按触发策略判断 + 校验一次性消费）
	ip := c.ClientIP()
	if captchaService.NeedCaptcha(c.Request.Context(), l.Username, ip) {
		if err := captchaService.Verify(c.Request.Context(), l.CaptchaId, l.Captcha); err != nil {
			captchaService.RecordLoginFail(c.Request.Context(), l.Username, ip)
			response.FailWithMessage("验证码错误", c)
			_ = loginLogService.CreateLoginLog(c.Request.Context(), system.SysLoginLog{
				UserName: l.Username, Ipaddr: ip, Browser: c.Request.UserAgent(),
				Status: "1", Msg: "验证码错误",
			})
			return
		}
	}

	// 3. 凭证校验
	user, err := userService.Login(c.Request.Context(), &system.SysUser{UserName: l.Username, Password: l.Password})
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("登录失败")
		captchaService.RecordLoginFail(c.Request.Context(), l.Username, ip)
		systemSvc.RecordLoginFail(c.Request.Context(), l.Username, cfg)
		response.FailWithMessage(err.Error(), c)
		_ = loginLogService.CreateLoginLog(c.Request.Context(), system.SysLoginLog{
			UserName: l.Username, Ipaddr: ip, Browser: c.Request.UserAgent(),
			Status: "1", Msg: "用户名或密码错误",
		})
		return
	}
	if user.Status != "0" {
		response.FailWithMessage("用户已被停用", c)
		_ = loginLogService.CreateLoginLog(c.Request.Context(), system.SysLoginLog{
			UserName: l.Username, Ipaddr: ip, Browser: c.Request.UserAgent(),
			Status: "1", Msg: "用户被停用",
		})
		return
	}

	// 4. 登录成功 清除失败计数与锁
	systemSvc.ClearLoginFail(c.Request.Context(), l.Username)
	captchaService.ResetLoginFail(c.Request.Context(), l.Username, ip)

	// 5. 密码过期检查（SysUser.PasswordUpdatedAt 字段补齐前传 nil）
	needChange := systemSvc.IsPasswordExpired(c.Request.Context(), user.PasswordUpdatedAt, cfg, time.Now())
	b.TokenNext(c, user, needChange)
}

// TokenNext 登录成功后签发 jwt，token 经 httpOnly Cookie 下发，响应体只回 expiresAt。
// 开启 System.UseMultipoint 时实现多端互踢(对齐 GVA):同一用户新登录将旧 token 入黑名单。
func (b *BaseApi) TokenNext(c *gin.Context, user system.SysUser, mustChangePwd bool) {
	token, claims, err := utils.LoginTokenWithExpire(&user, mustChangePwd)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("获取token失败")
		response.FailWithMessage("获取token失败", c)
		return
	}
	_ = loginLogService.CreateLoginLog(c.Request.Context(), system.SysLoginLog{
		UserName: user.UserName, Ipaddr: c.ClientIP(), Browser: c.Request.UserAgent(),
		Status: "0", Msg: "登录成功",
	})
	expire := int(claims.RegisteredClaims.ExpiresAt.Unix() - time.Now().Unix())
	resp := systemRes.LoginResponse{
		ExpiresAt:          claims.RegisteredClaims.ExpiresAt.Unix() * 1000,
		NeedChangePassword: mustChangePwd,
	}
	if !global.OPS_CONFIG.System.UseMultipoint {
		utils.SetToken(c, token, expire)
		response.OkWithDetailed(resp, "登录成功", c)
		return
	}
	// 多端登录:旧活跃 token 入黑名单,新 token 写入 Redis 活跃态
	if jwtStr, err := systemSvc.JwtServiceApp.GetRedisJWT(c.Request.Context(), user.GetUsername()); err == redis.Nil {
		if err := utils.SetRedisJWT(token, user.GetUsername()); err != nil {
			logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("设置登录状态失败")
			response.FailWithMessage("设置登录状态失败", c)
			return
		}
		utils.SetToken(c, token, expire)
		response.OkWithDetailed(resp, "登录成功", c)
	} else if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("设置登录状态失败")
		response.FailWithMessage("设置登录状态失败", c)
	} else {
		if err := systemSvc.JwtServiceApp.JsonInBlacklist(c.Request.Context(), system.JwtBlacklist{Jwt: jwtStr}); err != nil {
			response.FailWithMessage("jwt作废失败", c)
			return
		}
		if err := utils.SetRedisJWT(token, user.GetUsername()); err != nil {
			response.FailWithMessage("设置登录状态失败", c)
			return
		}
		utils.SetToken(c, token, expire)
		response.OkWithDetailed(resp, "登录成功", c)
	}
}

// Register
// @Tags     SysUser
// @Summary  用户注册账号
// @Produce  application/json
// @Param    data  body      systemReq.Register                                      true  "用户名, 昵称, 密码, 角色ID"
// @Success  200   {object}  response.Response{data=systemRes.SysUserResponse,msg=string}
// @Router   /user/admin_register [post]
func (b *BaseApi) Register(c *gin.Context) {
	var r systemReq.Register
	if err := c.ShouldBindJSON(&r); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := utils.ValidatePasswordComplexity(r.Password, securityConfigService.Current(c.Request.Context())); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	user := system.SysUser{
		UserName:    r.Username,
		NickName:    r.NickName,
		Password:    r.Password,
		Email:       r.Email,
		Phonenumber: r.Phonenumber,
		DeptId:      r.DeptId,
		RoleId:      r.RoleId,
		Status:      "0",
	}
	userReturn, err := userService.Register(c.Request.Context(), user)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("注册失败")
		response.FailWithDetailed(systemRes.SysUserResponse{User: userReturn}, "注册失败", c)
		return
	}
	response.OkWithDetailed(systemRes.SysUserResponse{User: userReturn}, "注册成功", c)
}

// GetUserList
// @Tags      SysUser
// @Summary   分页获取用户列表
// @Produce   application/json
// @Param     data  body      systemReq.GetUserList                                  true  "页码, 每页大小"
// @Success   200   {object}  response.Response{data=response.PageResult,msg=string}
// @Router    /user/getUserList [post]
func (b *BaseApi) GetUserList(c *gin.Context) {
	var pageInfo systemReq.GetUserList
	if err := c.ShouldBindJSON(&pageInfo); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := userService.GetUserInfoList(c.Request.Context(), pageInfo)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("获取失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		Rows:     list,
		Total:    total,
		PageNum:  pageInfo.PageNum,
		PageSize: pageInfo.PageSize,
	}, "获取成功", c)
}

// GetUserInfo 获取当前登录用户信息（/auth/getUserInfo）返回 user + roles(roleKey) + permissions(perms)
func (b *BaseApi) GetUserInfo(c *gin.Context) {
	userId := utils.GetUserID(c)
	user, roles, perms, err := userService.GetUserDetail(c.Request.Context(), userId)
	if err != nil {
		response.FailWithMessage("获取用户信息失败", c)
		return
	}
	response.OkWithDetailed(systemRes.UserInfoResponse{
		User: user, Roles: roles, Permissions: perms,
	}, "获取成功", c)
}

// Logout 退出登录(清 cookie + 当前 token 入黑名单)
func (b *BaseApi) Logout(c *gin.Context) {
	token := utils.GetToken(c)
	if token != "" {
		_ = systemSvc.JwtServiceApp.JsonInBlacklist(c.Request.Context(), system.JwtBlacklist{Jwt: token})
	}
	utils.ClearToken(c)
	response.OkWithMessage("退出成功", c)
}

// RefreshToken 刷新 token(读 cookie 续签,新 token 写回 cookie)
func (b *BaseApi) RefreshToken(c *gin.Context) {
	token := utils.GetToken(c)
	if token == "" {
		response.NoAuth("未登录或令牌失效", c)
		return
	}
	j := utils.NewJWT()
	claims, err := j.ParseToken(token)
	if err != nil {
		response.NoAuth("登录已过期，请重新登录", c)
		return
	}
	dr, _ := utils.ParseDuration(global.OPS_CONFIG.JWT.ExpiresTime)
	claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(dr))
	newToken, err := j.CreateToken(*claims)
	if err != nil {
		response.FailWithMessage("刷新token失败", c)
		return
	}
	utils.SetToken(c, newToken, int(dr.Seconds()))
	response.OkWithDetailed(systemRes.LoginResponse{
		ExpiresAt: claims.RegisteredClaims.ExpiresAt.Unix() * 1000,
	}, "刷新成功", c)
}

// GetConstantRoutes 常量路由(static 模式:前端本地管路由,后端返回空数组;
// 后续切换 dynamic 模式时再实现 getUserRoutes 完整下发)
func (b *BaseApi) GetConstantRoutes(c *gin.Context) {
	response.OkWithDetailed([]struct{}{}, "获取成功", c)
}
