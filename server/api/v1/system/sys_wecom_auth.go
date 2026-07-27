package system

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/common/response"
	"github.com/hllkk/devops-admin/server/model/system"
	systemSvc "github.com/hllkk/devops-admin/server/service/system"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// WecomAuthApi 企业微信扫码登录接口。
//
// 与社交登录(SocialApi)的差异:企微走"自渲染二维码 + 后端回调 + 前端轮询"模式,
// 回调由企业微信服务器(PC 扫码)或企微客户端 WebView 发起,非用户 PC 浏览器直连,
// 故 token 不在回调响应里直接下发,而是暂存 Redis 由前端轮询命中 confirmed 时下发 cookie。
// 交互形态对齐 /home/remote/devops-admin,但身份存储复用 sys_social(source=wecom)、
// token 下发复用单 x-token cookie(utils.SetToken),不另起 sys_wecom_users 表与双 cookie。
type WecomAuthApi struct{}

const (
	wecomQrcodeKeyPrefix  = "wecom:qrcode:"  // + sceneId:PC 扫码状态机
	wecomWebviewKeyPrefix = "wecom:webview:" // + state:企微客户端免登状态
	wecomLoginTTL         = 2 * time.Minute
	wecomQrcodeCountdown  = 120 // 前端倒计时(秒)
)

// 扫码状态机
const (
	wecomStatusWaiting   = "waiting"
	wecomStatusConfirmed = "confirmed"
	wecomStatusFail      = "fail"
	wecomStatusExpired   = "expired"
)

// wecomLoginPayload PC 扫码登录态(回调写入,轮询读取下发 cookie)。
type wecomLoginPayload struct {
	Status      string `json:"status"`
	Token       string `json:"token,omitempty"`       // JWT,仅 confirmed 时有
	ExpiresAtMs int64  `json:"expiresAtMs,omitempty"` // access token 过期毫秒
	Username    string `json:"username,omitempty"`    // recordSession 用
	DeptId      int64  `json:"deptId,omitempty"`      // recordSession 用
	ErrMsg      string `json:"errMsg,omitempty"`      // fail 时的错误信息
}

// wecomWebviewPayload 企微客户端 WebView 免登暂存(携带登录后回跳路径)。
type wecomWebviewPayload struct {
	Status   string `json:"status"`
	Redirect string `json:"redirect"`
}

// wecomClientFromConfig 从 sys_auth_config 构造企微客户端。
// 应用 Secret 对齐 WecomClientSecret(字段注释为 ClientSecret/CorpSecret 双语义)。
func wecomClientFromConfig(cfg system.SysAuthConfig) *utils.WecomClient {
	return &utils.WecomClient{
		CorpID:      cfg.WecomCorpId,
		AgentID:     cfg.WecomAgentId,
		CorpSecret:  cfg.WecomClientSecret,
		RedirectURI: cfg.WecomCallbackUrl,
	}
}

// QrCodeView 生成 PC 扫码登录二维码。
// @Tags      Wecom
// @Summary   企业微信扫码登录-获取二维码
// @Produce   application/json
// @Success   200  {object}  response.Response{data=object,msg=string}  "{sceneId,oauthUrl,countdown}"
// @Router    /auth/wecomLogin [get]
func (a *WecomAuthApi) QrCodeView(c *gin.Context) {
	ctx := c.Request.Context()
	cfg := (&systemSvc.AuthConfigService{}).Current(ctx)
	if !cfg.WecomEnabled {
		response.FailWithMessage("企业微信登录未启用", c)
		return
	}
	client := wecomClientFromConfig(cfg)
	if !client.Configured() {
		response.FailWithMessage("企业微信配置不完整", c)
		return
	}
	sceneId := uuid.New().String()
	payload, _ := json.Marshal(wecomLoginPayload{Status: wecomStatusWaiting})
	if err := global.OPS_REDIS.Set(ctx, wecomQrcodeKeyPrefix+sceneId, payload, wecomLoginTTL).Err(); err != nil {
		logger.WithCtx(ctx).Mod("wecom").Err(err).Error("写入扫码状态失败")
		response.FailWithMessage("获取二维码失败", c)
		return
	}
	response.OkWithData(gin.H{
		"sceneId":   sceneId,
		"oauthUrl":  client.AuthorizeURL(sceneId),
		"countdown": wecomQrcodeCountdown,
	}, c)
}

// QrCodeStatusView 轮询扫码状态;命中 confirmed 时下发 x-token cookie 并返回 expiresAt。
// @Tags      Wecom
// @Summary   企业微信扫码登录-轮询状态
// @Produce   application/json
// @Param     sceneId  query  string  true  "QrCodeView 返回的 sceneId"
// @Success   200  {object}  response.Response{data=object,msg=string}  "{status,expiresAt?,errMsg?}"
// @Router    /auth/qrCodeStatus [get]
func (a *WecomAuthApi) QrCodeStatusView(c *gin.Context) {
	ctx := c.Request.Context()
	sceneId := c.Query("sceneId")
	if sceneId == "" {
		response.FailWithMessage("参数错误", c)
		return
	}
	raw, err := global.OPS_REDIS.Get(ctx, wecomQrcodeKeyPrefix+sceneId).Result()
	if err == redis.Nil {
		response.OkWithData(gin.H{"status": wecomStatusExpired}, c)
		return
	}
	if err != nil {
		logger.WithCtx(ctx).Mod("wecom").Err(err).Error("读取扫码状态失败")
		response.FailWithMessage("登录状态查询失败", c)
		return
	}
	var payload wecomLoginPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		response.FailWithMessage("登录状态查询失败", c)
		return
	}
	if payload.Status == wecomStatusConfirmed && payload.Token != "" {
		// 用户 PC 浏览器请求:下发 cookie + 记录会话(IP/UA 准确),一次性 Del 防重放
		a.deliverWecomLogin(c, payload.DeptId, payload.Username, payload.Token)
		global.OPS_REDIS.Del(ctx, wecomQrcodeKeyPrefix+sceneId)
		response.OkWithData(gin.H{
			"status":    wecomStatusConfirmed,
			"expiresAt": payload.ExpiresAtMs,
		}, c)
		return
	}
	response.OkWithData(gin.H{
		"status": payload.Status,
		"errMsg": payload.ErrMsg,
	}, c)
}

// WecomCallbackView 企业微信 OAuth2 回调(PC 扫码 + WebView 免登共用,按 state 落点区分模式)。
// @Tags      Wecom
// @Summary   企业微信扫码登录-回调
// @Produce   text/html
// @Param     code   query  string  true  "授权 code"
// @Param     state  query  string  true  "sceneId(PC)/一次性令牌(WebView)"
// @Router    /wecomCallback [get]
func (a *WecomAuthApi) WecomCallbackView(c *gin.Context) {
	ctx := c.Request.Context()
	code, state := c.Query("code"), c.Query("state")
	if code == "" || state == "" {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(wecomFailHTML("登录参数缺失,请重新登录")))
		return
	}

	// 按 state 落点判定模式:PC 扫码(qrcode key) vs WebView 免登(webview key)
	mode := "qrcode"
	raw, err := global.OPS_REDIS.Get(ctx, wecomQrcodeKeyPrefix+state).Result()
	if err == redis.Nil {
		mode = "webview"
		raw, err = global.OPS_REDIS.Get(ctx, wecomWebviewKeyPrefix+state).Result()
	}
	if err == redis.Nil {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(wecomFailHTML("二维码或链接已过期,请重新登录")))
		return
	}
	if err != nil {
		logger.WithCtx(ctx).Mod("wecom").Err(err).Error("读取登录态失败")
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(wecomFailHTML("登录服务异常,请重试")))
		return
	}

	// 拉资料 → 建号/登录 → 签发
	token, expiresAtMs, username, deptId, failMsg := a.handleWecomLogin(c, code)
	if failMsg != "" {
		if mode == "qrcode" {
			// PC:写入 fail 让轮询看到错误
			payload, _ := json.Marshal(wecomLoginPayload{Status: wecomStatusFail, ErrMsg: failMsg})
			global.OPS_REDIS.Set(ctx, wecomQrcodeKeyPrefix+state, payload, wecomLoginTTL)
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(wecomFailHTML(failMsg)))
		return
	}

	if mode == "webview" {
		// WebView:回调即用户浏览器(企微客户端),直接下发 cookie + 设 localStorage + 跳回 SPA
		redirect := "/"
		var wp wecomWebviewPayload
		if json.Unmarshal([]byte(raw), &wp) == nil && wp.Redirect != "" {
			redirect = wp.Redirect
		}
		a.deliverWecomLogin(c, deptId, username, token)
		global.OPS_REDIS.Del(ctx, wecomWebviewKeyPrefix+state)
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(utils.BuildWebviewLoginHTML(redirect, expiresAtMs)))
		return
	}

	// PC 扫码:回调来自企微服务器,无法直接给用户浏览器下发 cookie → 暂存 confirmed,等轮询下发
	payload, _ := json.Marshal(wecomLoginPayload{
		Status:      wecomStatusConfirmed,
		Token:       token,
		ExpiresAtMs: expiresAtMs,
		Username:    username,
		DeptId:      deptId,
	})
	if err := global.OPS_REDIS.Set(ctx, wecomQrcodeKeyPrefix+state, payload, wecomLoginTTL).Err(); err != nil {
		logger.WithCtx(ctx).Mod("wecom").Err(err).Error("暂存登录态失败")
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(wecomFailHTML("登录服务异常,请重试")))
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(wecomPCWaitHTML))
}

// WecomWebviewLoginView 企微客户端 WebView 免登入口:生成一次性 state 后 302 跳转授权。
// @Tags      Wecom
// @Summary   企业微信客户端免登-跳转授权
// @Produce   text/html
// @Param     redirect  query  string  false  "登录后回跳路径(同源相对)"
// @Router    /auth/wecomWebviewLogin [get]
func (a *WecomAuthApi) WecomWebviewLoginView(c *gin.Context) {
	ctx := c.Request.Context()
	cfg := (&systemSvc.AuthConfigService{}).Current(ctx)
	if !cfg.WecomEnabled {
		response.FailWithMessage("企业微信登录未启用", c)
		return
	}
	client := wecomClientFromConfig(cfg)
	if !client.Configured() {
		response.FailWithMessage("企业微信配置不完整", c)
		return
	}
	state, err := utils.RandomWecomStateToken()
	if err != nil {
		response.FailWithMessage("登录服务异常", c)
		return
	}
	redirect := utils.SafeRedirectPath(c.Query("redirect"))
	payload, _ := json.Marshal(wecomWebviewPayload{Status: wecomStatusWaiting, Redirect: redirect})
	if err := global.OPS_REDIS.Set(ctx, wecomWebviewKeyPrefix+state, payload, wecomLoginTTL).Err(); err != nil {
		response.FailWithMessage("登录服务异常", c)
		return
	}
	// 返回 oauthUrl 由前端 location.replace 发起(而非服务端 302):
	// fetch 跟随跨域 302 到企微域名会触发 CORS 且拿不到可控结果,前端跳转可控错误处理。
	response.OkWithData(gin.H{"oauthUrl": client.AuthorizeURL(state)}, c)
}

// handleWecomLogin 用 code 走完 拉资料 → 建号/登录 → 签发。出错时 failMsg 非空。
func (a *WecomAuthApi) handleWecomLogin(c *gin.Context, code string) (token string, expiresAtMs int64, username string, deptId int64, failMsg string) {
	ctx := c.Request.Context()
	cfg := (&systemSvc.AuthConfigService{}).Current(ctx)
	client := wecomClientFromConfig(cfg)
	prof, err := client.ProfileByCode(ctx, code)
	if err != nil {
		logger.WithCtx(ctx).Mod("wecom").Err(err).Error("企微授权失败")
		return "", 0, "", 0, "企业微信授权失败,请重试"
	}
	user, err := a.loginOrRegister(ctx, prof, cfg)
	if err != nil {
		logger.WithCtx(ctx).Mod("wecom").Err(err).Error("企微登录建号失败")
		return "", 0, "", 0, err.Error()
	}
	token, expiresAtMs, err = a.issueWecomLogin(c, *user)
	if err != nil {
		return "", 0, "", 0, "登录失败,请重试"
	}
	return token, expiresAtMs, user.UserName, user.DeptId, ""
}

// loginOrRegister 企微 userid → 本地用户:已绑定则校验状态并刷新资料,未绑定则自动建号。
// 自动建号是企微场景与 github/gitee(拒绝自动建号)的本质差异——企业员工扫码即应可用。
func (a *WecomAuthApi) loginOrRegister(ctx context.Context, prof *utils.WecomProfile, cfg system.SysAuthConfig) (*system.SysUser, error) {
	// 已绑定:查 sys_social(wecom, userid)
	var rec system.SysSocial
	global.OPS_DB.WithContext(ctx).Where("source = ? AND open_id = ?", "wecom", prof.UserID).Limit(1).Find(&rec)
	if rec.ID > 0 {
		var user system.SysUser
		if err := global.OPS_DB.WithContext(ctx).Preload("Roles").Where("id = ?", rec.UserId).First(&user).Error; err != nil {
			return nil, errors.New("关联用户不存在,请联系管理员")
		}
		if user.Status != "0" {
			return nil, errors.New("用户已被停用")
		}
		a.refreshWecomSocial(ctx, &rec, prof)
		return &user, nil
	}

	// 未绑定:自动建号
	if cfg.WecomDefaultRoleId == 0 {
		return nil, errors.New("企业微信默认角色未配置,请联系管理员")
	}
	var role system.SysRole
	global.OPS_DB.WithContext(ctx).Where("role_id = ?", cfg.WecomDefaultRoleId).Limit(1).Find(&role)
	if role.RoleId == 0 {
		return nil, errors.New("企业微信默认角色不存在,请联系管理员")
	}
	if role.Status != "0" {
		return nil, errors.New("企业微信默认角色已停用,请联系管理员")
	}

	var newUser system.SysUser
	err := global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		randomPwd, e := randomWecomPassword()
		if e != nil {
			return e
		}
		now := time.Now()
		user := system.SysUser{
			UserName:          "wecom_" + prof.UserID, // 前缀避免与手动建号登录名冲突
			NickName:          prof.Name,
			Password:          utils.BcryptHash(randomPwd),
			Email:             prof.Email,
			Phonenumber:       prof.Mobile,
			Avatar:            prof.Avatar,
			RoleId:            cfg.WecomDefaultRoleId,
			Status:            "0",
			PasswordUpdatedAt: &now, // 标记刚设置,避免密码过期判定
		}
		if e := tx.Create(&user).Error; e != nil {
			return fmt.Errorf("创建用户失败: %w", e)
		}
		// 角色关联(sys_user_role),Preload Roles 依赖此连接记录
		if e := tx.Create(&system.SysUserRole{SysUserId: user.UserId, SysRoleId: cfg.WecomDefaultRoleId}).Error; e != nil {
			return fmt.Errorf("分配角色失败: %w", e)
		}
		if e := tx.Create(&system.SysSocial{
			UserId:   user.UserId,
			Source:   "wecom",
			OpenId:   prof.UserID,
			AuthId:   "wecom_" + prof.UserID,
			NickName: prof.Name,
			Avatar:   prof.Avatar,
			Email:    prof.Email,
			Mobile:   prof.Mobile,
		}).Error; e != nil {
			return fmt.Errorf("建立社交绑定失败: %w", e)
		}
		newUser = user
		return nil
	})
	if err != nil {
		return nil, err
	}

	// 回填 Roles 供 JWT claims(GetSuperAdmin 依赖)
	if e := global.OPS_DB.WithContext(ctx).Preload("Roles").Where("id = ?", newUser.UserId).First(&newUser).Error; e != nil {
		return nil, fmt.Errorf("加载用户角色失败: %w", e)
	}
	return &newUser, nil
}

// refreshWecomSocial 增量刷新企微资料快照(仅覆盖变化的非空字段)。
func (a *WecomAuthApi) refreshWecomSocial(ctx context.Context, rec *system.SysSocial, prof *utils.WecomProfile) {
	updates := map[string]interface{}{}
	if prof.Name != "" && prof.Name != rec.NickName {
		updates["nick_name"] = prof.Name
	}
	if prof.Avatar != "" && prof.Avatar != rec.Avatar {
		updates["avatar"] = prof.Avatar
	}
	if prof.Mobile != "" && prof.Mobile != rec.Mobile {
		updates["mobile"] = prof.Mobile
	}
	if prof.Email != "" && prof.Email != rec.Email {
		updates["email"] = prof.Email
	}
	if len(updates) == 0 {
		return
	}
	if err := global.OPS_DB.WithContext(ctx).Model(&system.SysSocial{}).Where("id = ?", rec.ID).Updates(updates).Error; err != nil {
		logger.WithCtx(ctx).Mod("wecom").Err(err).Warn("刷新企微资料快照失败(非阻断)")
	}
}

// issueWecomLogin 签发企微登录 token:复用 LoginTokenWithExpire + 登录日志 + 多端互踢。
// 不在此 SetToken/记录会话——下发时机由调用方经 deliverWecomLogin 决定,
// 以拿到用户真实浏览器 IP/UA(PC 轮询/WebView 回调)。
func (a *WecomAuthApi) issueWecomLogin(c *gin.Context, user system.SysUser) (token string, expiresAtMs int64, err error) {
	ctx := c.Request.Context()
	uaBrowser, uaOS, uaDevice := utils.ParseUserAgent(c.Request.UserAgent())
	token, claims, err := utils.LoginTokenWithExpire(&user, false)
	if err != nil {
		logger.WithCtx(ctx).Mod("wecom").Err(err).Error("签发 token 失败")
		return "", 0, err
	}
	_ = loginLogService.CreateLoginLog(ctx, system.SysLoginLog{
		UserName: user.UserName, Ipaddr: c.ClientIP(), Browser: uaBrowser, Os: uaOS, DeviceType: uaDevice,
		Status: "0", Msg: "企业微信登录成功",
	})
	expiresAtMs = claims.RegisteredClaims.ExpiresAt.Unix() * 1000
	// 多端互踢(对齐 TokenNext):旧活跃 token 入黑名单,新 token 写 Redis 活跃态
	if global.OPS_CONFIG.System.UseMultipoint {
		if jwtStr, e := systemSvc.JwtServiceApp.GetRedisJWT(ctx, user.GetUsername()); e == nil && jwtStr != "" {
			_ = systemSvc.JwtServiceApp.JsonInBlacklist(ctx, system.JwtBlacklist{Jwt: jwtStr})
		}
		_ = utils.SetRedisJWT(token, user.GetUsername())
	}
	return token, expiresAtMs, nil
}

// deliverWecomLogin 向当前浏览器下发登录态:httpOnly cookie + 记录在线会话。
// 必须在用户真实浏览器请求(PC 轮询 QrCodeStatusView / WebView 回调)中调用。
func (a *WecomAuthApi) deliverWecomLogin(c *gin.Context, deptId int64, username, token string) {
	claims, err := utils.NewJWT().ParseToken(token)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("wecom").Err(err).Error("下发登录态时解析 token 失败")
		return
	}
	expire := int(claims.ExpiresAt.Unix() - time.Now().Unix())
	if expire < 0 {
		expire = 0
	}
	utils.SetToken(c, token, expire)
	// 复用 BaseApi.recordSession 记录在线设备(DeptId/UserName 走快照,其余来自 claims)
	(&BaseApi{}).recordSession(c, system.SysUser{DeptId: deptId, UserName: username}, token, *claims)
}

// randomWecomPassword 生成不可用于密码登录的随机密码(自动建号专用,32 位 hex)。
func randomWecomPassword() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// wecomFailHTML 企微回调失败提示页(msg 经 HTML 转义防注入)。
func wecomFailHTML(msg string) string {
	return fmt.Sprintf(`<!DOCTYPE html><html lang="zh-CN"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0"><title>登录失败</title>`+
		`<style>body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;display:flex;align-items:center;justify-content:center;height:100vh;margin:0;background:#f5f5f5;color:#d4380d;font-size:16px;text-align:center;padding:0 20px}</style>`+
		`</head><body><p>%s</p></body></html>`, html.EscapeString(msg))
}

// wecomPCWaitHTML PC 扫码回调成功提示(用户在企微 App 侧看到,PC 端靠轮询完成登录)。
const wecomPCWaitHTML = `<!DOCTYPE html><html lang="zh-CN"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0"><title>登录成功</title>` +
	`<style>body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;display:flex;align-items:center;justify-content:center;height:100vh;margin:0;background:#f5f5f5;color:#52c41a;font-size:16px}</style>` +
	`</head><body><p>登录成功,请在电脑端继续</p></body></html>`
