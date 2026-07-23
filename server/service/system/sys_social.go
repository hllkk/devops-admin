package system

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"golang.org/x/oauth2"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
	"github.com/hllkk/devops-admin/server/utils/crypto"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// SocialService 第三方账号绑定/社交登录业务(对齐 docs/superpowers/specs/2026-07-23-social-binding-design.md)
type SocialService struct{}

// CallbackResult 回调处理结果:绑定成功(无 user) 或 登录成功(带 user,供 api 调 TokenNext 签 JWT)
type CallbackResult struct {
	IsLogin bool
	User    *system.SysUser
}

// 支持的 source(对齐前端 SocialSource)
var socialValidSources = map[string]bool{
	"wechat_open": true,
	"gitee":       true,
	"github":      true,
}

func isValidSocialSource(s string) bool { return socialValidSources[s] }

// providerCfg 从 sys_auth_config 提取某 provider 的配置
type providerCfg struct {
	Enabled      bool
	ClientId     string
	ClientSecret string
	CallbackUrl  string
}

func providerConfig(source string, cfg system.SysAuthConfig) (providerCfg, bool) {
	switch source {
	case "github":
		return providerCfg{cfg.GithubEnabled, cfg.GithubClientId, cfg.GithubClientSecret, cfg.GithubCallbackUrl}, true
	case "gitee":
		return providerCfg{cfg.GiteeEnabled, cfg.GiteeClientId, cfg.GiteeClientSecret, cfg.GiteeCallbackUrl}, true
	case "wechat_open":
		return providerCfg{cfg.WechatEnabled, cfg.WechatClientId, cfg.WechatClientSecret, cfg.WechatCallbackUrl}, true
	}
	return providerCfg{}, false
}

// --- state 编解码 ---
// 用 URLEncoding(非 StdEncoding):state 经 OAuth provider 走 URL query 回传,
// StdEncoding 产生的 + 会被浏览器按 query 规则解码成空格,破坏 state;URLEncoding 用 -_ 规避。
// 前端 social-callback 的 atob 需兼容 -_(见 social-callback/index.vue)。
func encodeState(st systemReq.SocialState) (string, error) {
	b, err := json.Marshal(st)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func decodeState(s string) (systemReq.SocialState, error) {
	var st systemReq.SocialState
	if s == "" {
		return st, errors.New("state 为空")
	}
	b, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return st, fmt.Errorf("state 解码失败: %w", err)
	}
	if err := json.Unmarshal(b, &st); err != nil {
		return st, fmt.Errorf("state 解析失败: %w", err)
	}
	return st, nil
}

// --- 通用 HTTP GET JSON ---

func httpGetJSON(ctx context.Context, rawURL string, headers map[string]string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("请求 %s 失败: status=%d body=%s", rawURL, resp.StatusCode, string(body))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// socialProfile 三平台统一的用户资料 + token 快照
type socialProfile struct {
	OpenId       string
	UnionId      string
	NickName     string
	Avatar       string
	Email        string
	AccessToken  string
	RefreshToken string
	ExpireIn     int64
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

// --- GetAuthURL 拼授权 URL ---
// jwtUserId>0 表示当前已登录(JWT 校验通过) → 绑定意图;否则登录意图。
func (s *SocialService) GetAuthURL(ctx context.Context, source, domain string, jwtUserId int64) (string, error) {
	if !isValidSocialSource(source) {
		return "", errors.New("不支持的登录方式")
	}
	cfg := (&AuthConfigService{}).Current(ctx)
	pc, ok := providerConfig(source, cfg)
	if !ok {
		return "", errors.New("不支持的登录方式")
	}
	if !pc.Enabled {
		return "", errors.New("该第三方登录未启用")
	}

	st := systemReq.SocialState{Domain: domain, Source: source}
	if jwtUserId > 0 {
		st.UserId = jwtUserId
		st.Intent = "bind"
	}
	stateStr, err := encodeState(st)
	if err != nil {
		return "", err
	}

	switch source {
	case "github":
		conf := &oauth2.Config{
			ClientID: pc.ClientId, ClientSecret: pc.ClientSecret, RedirectURL: pc.CallbackUrl,
			Scopes:   []string{"read:user", "user:email"},
			Endpoint: oauth2.Endpoint{AuthURL: "https://github.com/login/oauth/authorize", TokenURL: "https://github.com/login/oauth/access_token", AuthStyle: oauth2.AuthStyleInParams},
		}
		return conf.AuthCodeURL(stateStr, oauth2.SetAuthURLParam("redirect_uri", pc.CallbackUrl)), nil
	case "gitee":
		conf := &oauth2.Config{
			ClientID: pc.ClientId, ClientSecret: pc.ClientSecret, RedirectURL: pc.CallbackUrl,
			Scopes:   []string{"user_info", "emails"},
			Endpoint: oauth2.Endpoint{AuthURL: "https://gitee.com/oauth/authorize", TokenURL: "https://gitee.com/oauth/token", AuthStyle: oauth2.AuthStyleInParams},
		}
		return conf.AuthCodeURL(stateStr, oauth2.SetAuthURLParam("redirect_uri", pc.CallbackUrl)), nil
	case "wechat_open":
		return fmt.Sprintf(
			"https://open.weixin.qq.com/connect/qrconnect?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_login&state=%s",
			pc.ClientId, url.QueryEscape(pc.CallbackUrl), stateStr,
		), nil
	}
	return "", errors.New("不支持的登录方式")
}

// --- Callback 统一回调入口 ---
// jwtUserId 为 api 层从 JWT cookie 解析并校验过的 userId;绑定流程要求其与 state.userId 一致(防伪造 userId 绑定)。
func (s *SocialService) Callback(ctx context.Context, req systemReq.SocialLoginForm, jwtUserId int64) (*CallbackResult, error) {
	if !isValidSocialSource(req.Source) {
		return nil, errors.New("不支持的登录方式")
	}
	st, err := decodeState(req.SocialState)
	if err != nil {
		return nil, errors.New("授权状态校验失败，请重试")
	}
	// 防 state 篡改:state.source 必须与请求 source 一致
	if st.Source != req.Source {
		return nil, errors.New("授权状态校验失败，请重试")
	}
	cfg := (&AuthConfigService{}).Current(ctx)
	pc, ok := providerConfig(req.Source, cfg)
	if !ok {
		return nil, errors.New("不支持的登录方式")
	}
	if !pc.Enabled {
		return nil, errors.New("该第三方登录未启用")
	}

	prof, err := s.fetchProfile(ctx, req.Source, pc, req.SocialCode)
	if err != nil {
		logger.WithCtx(ctx).Mod("social").Err(err).Error("第三方授权失败")
		return nil, errors.New("第三方授权失败，请重试")
	}

	// 绑定意图:state 带 bind+userId,且与 JWT 验证过的 userId 一致
	if st.Intent == "bind" && st.UserId > 0 && st.UserId == jwtUserId {
		if err := s.bind(ctx, req.Source, st.UserId, prof); err != nil {
			return nil, err
		}
		return &CallbackResult{IsLogin: false}, nil
	}
	// 否则走登录流程
	user, err := s.login(ctx, req.Source, prof)
	if err != nil {
		return nil, err
	}
	return &CallbackResult{IsLogin: true, User: user}, nil
}

func (s *SocialService) fetchProfile(ctx context.Context, source string, pc providerCfg, code string) (*socialProfile, error) {
	switch source {
	case "github":
		return s.fetchGithubProfile(ctx, pc, code)
	case "gitee":
		return s.fetchGiteeProfile(ctx, pc, code)
	case "wechat_open":
		return s.fetchWechatProfile(ctx, pc, code)
	}
	return nil, errors.New("不支持的登录方式")
}

// --- GitHub ---

type githubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Email     string `json:"email"`
}

type githubEmail struct {
	Email   string `json:"email"`
	Primary bool   `json:"primary"`
}

func (s *SocialService) fetchGithubProfile(ctx context.Context, pc providerCfg, code string) (*socialProfile, error) {
	conf := &oauth2.Config{
		ClientID: pc.ClientId, ClientSecret: pc.ClientSecret, RedirectURL: pc.CallbackUrl,
		Scopes:   []string{"read:user", "user:email"},
		Endpoint: oauth2.Endpoint{AuthURL: "https://github.com/login/oauth/authorize", TokenURL: "https://github.com/login/oauth/access_token", AuthStyle: oauth2.AuthStyleInParams},
	}
	tok, err := conf.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("github 授权失败: %w", err)
	}
	headers := map[string]string{"Authorization": "Bearer " + tok.AccessToken, "Accept": "application/vnd.github+json"}
	var u githubUser
	if err := httpGetJSON(ctx, "https://api.github.com/user", headers, &u); err != nil {
		return nil, err
	}
	email := u.Email
	if email == "" {
		var emails []githubEmail
		if err := httpGetJSON(ctx, "https://api.github.com/user/emails", headers, &emails); err == nil {
			for _, e := range emails {
				if e.Primary {
					email = e.Email
					break
				}
			}
		}
	}
	return &socialProfile{
		OpenId:       strconv.FormatInt(u.ID, 10),
		NickName:     firstNonEmpty(u.Name, u.Login),
		Avatar:       u.AvatarURL,
		Email:        email,
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		ExpireIn:     tokenExpireIn(tok.Expiry),
	}, nil
}

// --- Gitee ---

type giteeUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Email     string `json:"email"`
}

func (s *SocialService) fetchGiteeProfile(ctx context.Context, pc providerCfg, code string) (*socialProfile, error) {
	conf := &oauth2.Config{
		ClientID: pc.ClientId, ClientSecret: pc.ClientSecret, RedirectURL: pc.CallbackUrl,
		Scopes:   []string{"user_info", "emails"},
		Endpoint: oauth2.Endpoint{AuthURL: "https://gitee.com/oauth/authorize", TokenURL: "https://gitee.com/oauth/token", AuthStyle: oauth2.AuthStyleInParams},
	}
	tok, err := conf.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("gitee 授权失败: %w", err)
	}
	var u giteeUser
	if err := httpGetJSON(ctx, "https://gitee.com/api/v5/user?access_token="+tok.AccessToken, nil, &u); err != nil {
		return nil, err
	}
	return &socialProfile{
		OpenId:       strconv.FormatInt(u.ID, 10),
		NickName:     firstNonEmpty(u.Name, u.Login),
		Avatar:       u.AvatarURL,
		Email:        u.Email,
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		ExpireIn:     tokenExpireIn(tok.Expiry),
	}, nil
}

// --- 微信开放平台(token 交换参数不标准,手动 net/http) ---

type wechatTokenResp struct {
	AccessToken string `json:"access_token"`
	OpenID      string `json:"openid"`
	UnionID     string `json:"unionid"`
	ExpiresIn   int64  `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

type wechatUserResp struct {
	OpenID     string `json:"openid"`
	UnionID    string `json:"unionid"`
	Nickname   string `json:"nickname"`
	HeadImgURL string `json:"headimgurl"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

func (s *SocialService) fetchWechatProfile(ctx context.Context, pc providerCfg, code string) (*socialProfile, error) {
	tokenURL := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		pc.ClientId, pc.ClientSecret, code,
	)
	var tr wechatTokenResp
	if err := httpGetJSON(ctx, tokenURL, nil, &tr); err != nil {
		return nil, fmt.Errorf("微信授权失败: %w", err)
	}
	if tr.ErrCode != 0 || tr.AccessToken == "" {
		return nil, fmt.Errorf("微信授权失败: %s", tr.ErrMsg)
	}
	userURL := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s",
		tr.AccessToken, tr.OpenID,
	)
	var ur wechatUserResp
	if err := httpGetJSON(ctx, userURL, nil, &ur); err != nil {
		return nil, err
	}
	if ur.ErrCode != 0 {
		return nil, fmt.Errorf("微信用户信息获取失败: %s", ur.ErrMsg)
	}
	return &socialProfile{
		OpenId:      tr.OpenID,
		UnionId:     firstNonEmpty(tr.UnionID, ur.UnionID),
		NickName:    ur.Nickname,
		Avatar:      ur.HeadImgURL,
		AccessToken: tr.AccessToken,
		ExpireIn:    tr.ExpiresIn,
	}, nil
}

// tokenExpireIn 把 oauth2.Token.Expiry 换算成剩余秒数,零值表示无有效期信息
func tokenExpireIn(expiry time.Time) int64 {
	if expiry.IsZero() {
		return 0
	}
	if d := int64(time.Until(expiry).Seconds()); d > 0 {
		return d
	}
	return 0
}

// --- 绑定 ---
// 每次查询独立 tx(global.OPS_DB.WithContext(ctx) 起新会话),不复用被 finisher 污染的 db 变量,
// 避免 WHERE 累积 / ErrRecordNotFound 残留短路(见 backend-layer-rules GORM 链式查询规则)。
func (s *SocialService) bind(ctx context.Context, source string, userId int64, prof *socialProfile) error {
	// (source, openId) 查重:一个第三方账号只能绑一个本地用户
	var occ system.SysSocial
	global.OPS_DB.WithContext(ctx).Where("source = ? AND open_id = ?", source, prof.OpenId).Limit(1).Find(&occ)
	if occ.ID > 0 {
		if occ.UserId != userId {
			return errors.New("该第三方账号已被其他用户绑定")
		}
		return errors.New("您已绑定该平台的账号")
	}
	// 微信 unionId 查重(同一微信用户跨应用去重)
	if prof.UnionId != "" {
		var u2 system.SysSocial
		global.OPS_DB.WithContext(ctx).Where("source = ? AND union_id = ?", source, prof.UnionId).Limit(1).Find(&u2)
		if u2.ID > 0 && u2.UserId != userId {
			return errors.New("该第三方账号已被其他用户绑定")
		}
	}
	// (userId, source) 查重:一个本地用户同一 provider 只能绑一个
	var mine system.SysSocial
	global.OPS_DB.WithContext(ctx).Where("user_id = ? AND source = ?", userId, source).Limit(1).Find(&mine)
	if mine.ID > 0 {
		return errors.New("您已绑定该平台的账号")
	}

	// token AES-256-GCM 加密落库
	tokenKey := global.OPS_CONFIG.Social.TokenKey
	if tokenKey == "" {
		logger.Bg().Mod("social").Error("social.token-key 未配置,无法加密 token")
		return errors.New("绑定失败")
	}
	encAccess, err := crypto.AESGCMEncrypt(prof.AccessToken, tokenKey)
	if err != nil {
		logger.WithCtx(ctx).Mod("social").Err(err).Error("access token 加密失败")
		return errors.New("绑定失败")
	}
	encRefresh, _ := crypto.AESGCMEncrypt(prof.RefreshToken, tokenKey) // 无 refresh 时加密空串,容错

	rec := system.SysSocial{
		UserId:       userId,
		Source:       source,
		OpenId:       prof.OpenId,
		UnionId:      prof.UnionId,
		AuthId:       source + "_" + prof.OpenId,
		NickName:     prof.NickName,
		Avatar:       prof.Avatar,
		Email:        prof.Email,
		AccessToken:  encAccess,
		RefreshToken: encRefresh,
		ExpireIn:     prof.ExpireIn,
	}
	if err := global.OPS_DB.WithContext(ctx).Create(&rec).Error; err != nil {
		logger.WithCtx(ctx).Mod("social").Err(err).Error("写入 sys_social 失败")
		return errors.New("绑定失败")
	}
	return nil
}

// --- 登录 ---

func (s *SocialService) login(ctx context.Context, source string, prof *socialProfile) (*system.SysUser, error) {
	var rec system.SysSocial
	global.OPS_DB.WithContext(ctx).Where("source = ? AND open_id = ?", source, prof.OpenId).Limit(1).Find(&rec)
	if rec.ID == 0 && prof.UnionId != "" {
		// 微信 unionId 兜底查询(同一微信用户不同应用 openId 不同)
		var rec2 system.SysSocial
		global.OPS_DB.WithContext(ctx).Where("source = ? AND union_id = ?", source, prof.UnionId).Limit(1).Find(&rec2)
		if rec2.ID > 0 {
			rec = rec2
		}
	}
	if rec.ID == 0 {
		return nil, errors.New("该账号未绑定本地用户，请先注册")
	}

	// 查 SysUser 全量(Preload Roles 供 GetSuperAdmin/claims 使用)
	var user system.SysUser
	if err := global.OPS_DB.WithContext(ctx).Preload("Roles").Where("id = ?", rec.UserId).First(&user).Error; err != nil {
		return nil, errors.New("用户不存在")
	}
	if user.Status != "0" {
		return nil, errors.New("用户已被停用")
	}
	return &user, nil
}

// --- List ---

func (s *SocialService) List(ctx context.Context, userId int64) ([]system.SysSocial, error) {
	var list []system.SysSocial
	if err := global.OPS_DB.WithContext(ctx).Where("user_id = ?", userId).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// --- Unbind ---
// 解绑前校验:至少保留一种登录方式(本地密码可登录 或 其它社交绑定)
func (s *SocialService) Unbind(ctx context.Context, userId, socialId int64) error {
	var rec system.SysSocial
	global.OPS_DB.WithContext(ctx).Where("id = ?", socialId).Limit(1).Find(&rec)
	if rec.ID == 0 {
		return errors.New("关联记录不存在")
	}
	if rec.UserId != userId {
		return errors.New("无权解绑该账号")
	}

	total := 0
	// 本地密码可登录?
	var u system.SysUser
	global.OPS_DB.WithContext(ctx).Where("id = ?", userId).Limit(1).Find(&u)
	if u.UserId != 0 && u.Password != "" {
		secCfg := (&SecurityConfigService{}).Current(ctx)
		if !IsPasswordExpired(ctx, u.PasswordUpdatedAt, secCfg, time.Now()) {
			total++
		}
	}
	// 其它社交绑定
	var count int64
	global.OPS_DB.WithContext(ctx).Model(&system.SysSocial{}).Where("user_id = ? AND id <> ?", userId, socialId).Count(&count)
	total += int(count)
	if total == 0 {
		return errors.New("请至少保留一种登录方式")
	}

	if err := global.OPS_DB.WithContext(ctx).Delete(&rec).Error; err != nil {
		logger.WithCtx(ctx).Mod("social").Err(err).Error("解绑失败")
		return errors.New("解绑失败")
	}
	return nil
}
