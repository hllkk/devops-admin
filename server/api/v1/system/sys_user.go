package system

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	jwt "github.com/golang-jwt/jwt/v5"
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

	// 解析客户端 UA(浏览器/操作系统/设备类型),供登录日志记录
	uaBrowser, uaOS, uaDevice := utils.ParseUserAgent(c.Request.UserAgent())

	// 1. 账号锁定检查
	if cfg.LoginFailLockCount > 0 && systemSvc.IsAccountLocked(c.Request.Context(), l.Username) {
		response.FailWithMessage("账号已锁定，请 "+strconv.Itoa(cfg.LoginFailLockTime)+" 分钟后再试", c)
		_ = loginLogService.CreateLoginLog(c.Request.Context(), system.SysLoginLog{
			UserName: l.Username, Ipaddr: c.ClientIP(), Browser: uaBrowser, Os: uaOS, DeviceType: uaDevice,
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
				UserName: l.Username, Ipaddr: ip, Browser: uaBrowser, Os: uaOS, DeviceType: uaDevice,
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
			UserName: l.Username, Ipaddr: ip, Browser: uaBrowser, Os: uaOS, DeviceType: uaDevice,
			Status: "1", Msg: "用户名或密码错误",
		})
		return
	}
	if user.Status != "0" {
		response.FailWithMessage("用户已被停用", c)
		_ = loginLogService.CreateLoginLog(c.Request.Context(), system.SysLoginLog{
			UserName: l.Username, Ipaddr: ip, Browser: uaBrowser, Os: uaOS, DeviceType: uaDevice,
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
	uaBrowser, uaOS, uaDevice := utils.ParseUserAgent(c.Request.UserAgent())
	token, claims, err := utils.LoginTokenWithExpire(&user, mustChangePwd)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("获取token失败")
		response.FailWithMessage("获取token失败", c)
		return
	}
	_ = loginLogService.CreateLoginLog(c.Request.Context(), system.SysLoginLog{
		UserName: user.UserName, Ipaddr: c.ClientIP(), Browser: uaBrowser, Os: uaOS, DeviceType: uaDevice,
		Status: "0", Msg: "登录成功",
	})
	expire := int(claims.RegisteredClaims.ExpiresAt.Unix() - time.Now().Unix())
	resp := systemRes.LoginResponse{
		ExpiresAt:          claims.RegisteredClaims.ExpiresAt.Unix() * 1000,
		NeedChangePassword: mustChangePwd,
	}
	if !global.OPS_CONFIG.System.UseMultipoint {
		utils.SetToken(c, token, expire)
		b.recordSession(c, user, token, claims)
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
		b.recordSession(c, user, token, claims)
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
		b.recordSession(c, user, token, claims)
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

// GetUserInfo 获取当前登录用户信息（/auth/getUserInfo）返回 user + roles(roleKey) + permissions(perms)
func (b *BaseApi) GetUserInfo(c *gin.Context) {
	userId := utils.GetUserID(c)
	user, roles, perms, apps, defaultRouter, err := userService.GetUserDetail(c.Request.Context(), userId)
	if err != nil {
		response.FailWithMessage("获取用户信息失败", c)
		return
	}
	response.OkWithDetailed(systemRes.UserInfoResponse{
		User: user, Roles: roles, Permissions: perms, Apps: apps, DefaultRouter: defaultRouter,
	}, "获取成功", c)
}

// Logout 退出登录(清 cookie + 当前 token 入黑名单 + 删除在线设备会话记录)
func (b *BaseApi) Logout(c *gin.Context) {
	token := utils.GetToken(c)
	if token != "" {
		ctx := c.Request.Context()
		_ = systemSvc.JwtServiceApp.JsonInBlacklist(ctx, system.JwtBlacklist{Jwt: token})
		// 同步清理当前会话的在线设备记录
		if claims, err := utils.NewJWT().ParseToken(token); err == nil {
			_ = onlineService.RemoveSession(ctx, claims.BaseClaims.ID, claims.RegisteredClaims.ID)
		}
	}
	utils.ClearToken(c)
	response.OkWithMessage("退出成功", c)
}

// recordSession 登录成功后把当前会话写入在线设备列表(Redis)。
// 记录失败仅记日志,不阻断登录流程。
func (b *BaseApi) recordSession(c *gin.Context, user system.SysUser, token string, claims systemReq.CustomClaims) {
	ip := c.ClientIP()
	browser, osStr, device := utils.ParseUserAgent(c.Request.UserAgent())
	if err := onlineService.RecordSession(c.Request.Context(), claims.BaseClaims.ID, user.DeptId, system.OnlineSession{
		Token:         token,
		TokenId:       claims.RegisteredClaims.ID,
		UserName:      user.UserName,
		Ipaddr:        ip,
		LoginLocation: utils.ParseIPLocation(ip),
		Browser:       browser,
		Os:            osStr,
		DeviceType:    device,
		LoginTime:     time.Now(),
	}); err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("记录在线设备会话失败")
	}
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

// GetConstantRoutes 已迁移至 RouteApi(见 sys_route.go),与 getUserRoutes/isRouteExist 统一到 /route 资源。
