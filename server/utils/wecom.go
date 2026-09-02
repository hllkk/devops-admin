package utils

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// 企业微信自建应用 网页授权/扫码登录 封装。
//
// 设计取舍(对照 /home/remote/devops-admin 的 backend/utils/wecom.go):
//   - 配置不在此读取:WecomClient 由 api 层从 sys_auth_config(AuthConfigService.Current) 注入,
//     避免 utils 反向依赖 service 造成循环依赖。当前项目企微 Secret 明文存 sys_auth_config,
//     故无需远程的 AgentSecret 解密链路。
//   - access_token 缓存统一走 global.OPS_CACHE(Redis 优先、内存降级),并用
//     global.OPS_Concurrency_Control(singleflight) 防并发击穿,替代远程的手写 sync.Mutex+内存缓存。
//   - 授权链路与远程一致:oauth2/authorize + scope=snsapi_privateinfo,URL 既用于 PC 扫码
//     (前端 qrcode 库渲染成二维码)又用于企微客户端 WebView 免登;该 scope 可拿到 user_ticket
//     进而换手机/邮箱(qrConnect iframe 方式拿不到,已弃用)。

const (
	wecomQyAPIBase              = "https://qyapi.weixin.qq.com/cgi-bin"
	wecomAuthorizeURL           = "https://open.weixin.qq.com/connect/oauth2/authorize"
	wecomAccessTokenCachePrefix = "wecom:access_token:" // + CorpID
	wecomAccessTokenRefreshLead = 120 * time.Second     // 提前 120s 过期,规避边界失效
)

// wecomHTTPClient 复用连接池的共享 HTTP 客户端
var wecomHTTPClient = &http.Client{
	Timeout: 15 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	},
}

// WecomClient 企业微信自建应用客户端。零值不可用,需由 api 层注入配置后使用。
type WecomClient struct {
	CorpID      string // 企业 CorpId
	AgentID     int    // 应用 AgentId
	CorpSecret  string // 应用 Secret(corpsecret,换 access_token 用)
	RedirectURI string // OAuth2 回调地址(企微管理端配置的可信域名下)
}

// Configured 判断企微配置是否齐全(可生成授权 URL / 换 token)。
func (c *WecomClient) Configured() bool {
	return c.CorpID != "" && c.CorpSecret != "" && c.AgentID != 0 && c.RedirectURI != ""
}

// AuthorizeURL 构造网页授权 URL(scope=snsapi_privateinfo)。
// state 由调用方传入并保证为企微允许的字符集([a-zA-Z0-9]):PC 扫码传 sceneId(UUID),
// WebView 免登传一次性令牌(RandomWecomStateToken)。redirect_uri 经 QueryEscape 转义。
func (c *WecomClient) AuthorizeURL(state string) string {
	return fmt.Sprintf("%s?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_privateinfo&state=%s&agentid=%d#wechat_redirect",
		wecomAuthorizeURL,
		c.CorpID,
		url.QueryEscape(c.RedirectURI),
		state,
		c.AgentID,
	)
}

// AccessToken 取企业微信 access_token(全局唯一,默认 2h 有效期)。
// 缓存策略:OPS_CACHE 提前 120s 过期;并发回源走 singleflight 防击穿。
func (c *WecomClient) AccessToken(ctx context.Context) (string, error) {
	key := wecomAccessTokenCachePrefix + c.CorpID
	if v, ok := global.OPS_CACHE.Get(key); ok {
		if s, ok := v.(string); ok && s != "" {
			return s, nil
		}
	}
	v, err, _ := global.OPS_Concurrency_Control.Do(key, func() (interface{}, error) {
		// singleflight 命中合并请求后,double-check 缓存(前一个并发可能刚写回)
		if v, ok := global.OPS_CACHE.Get(key); ok {
			if s, ok := v.(string); ok && s != "" {
				return s, nil
			}
		}
		return c.fetchAccessToken(ctx)
	})
	if err != nil {
		return "", err
	}
	return v.(string), nil
}

// fetchAccessToken 向企微换取 access_token 并写缓存。仅在缓存未命中时调用。
func (c *WecomClient) fetchAccessToken(ctx context.Context) (string, error) {
	if c.CorpID == "" || c.CorpSecret == "" {
		return "", errors.New("企业微信 CorpId/Secret 未配置")
	}
	reqURL := fmt.Sprintf("%s/gettoken?corpid=%s&corpsecret=%s", wecomQyAPIBase, c.CorpID, c.CorpSecret)
	var resp struct {
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := wecomGet(ctx, reqURL, &resp); err != nil {
		return "", fmt.Errorf("获取企微 access_token 失败: %w", err)
	}
	if resp.ErrCode != 0 {
		return "", fmt.Errorf("获取企微 access_token 失败: errcode=%d errmsg=%s", resp.ErrCode, resp.ErrMsg)
	}
	ttl := time.Duration(resp.ExpiresIn)*time.Second - wecomAccessTokenRefreshLead
	if ttl <= 0 {
		// expires_in 异常小时兜底,避免负 TTL
		ttl = time.Duration(resp.ExpiresIn) * time.Second / 2
	}
	global.OPS_CACHE.Set(wecomAccessTokenCachePrefix+c.CorpID, resp.AccessToken, ttl)
	return resp.AccessToken, nil
}

// UserIDByCode 用网页授权 code 换 userid 与 user_ticket。
// snsapi_privateinfo 时返回 user_ticket(后续换敏感信息);snsapi_base 时 user_ticket 为空。
func (c *WecomClient) UserIDByCode(ctx context.Context, code string) (userID, userTicket string, err error) {
	accessToken, err := c.AccessToken(ctx)
	if err != nil {
		return "", "", err
	}
	reqURL := fmt.Sprintf("%s/auth/getuserinfo?access_token=%s&code=%s", wecomQyAPIBase, accessToken, code)
	var resp struct {
		ErrCode    int    `json:"errcode"`
		ErrMsg     string `json:"errmsg"`
		UserID     string `json:"userid"`
		UserTicket string `json:"user_ticket"`
	}
	if err := wecomGet(ctx, reqURL, &resp); err != nil {
		return "", "", fmt.Errorf("企微 getuserinfo 失败: %w", err)
	}
	if resp.ErrCode != 0 {
		return "", "", fmt.Errorf("企微 getuserinfo 失败: errcode=%d errmsg=%s", resp.ErrCode, resp.ErrMsg)
	}
	return resp.UserID, resp.UserTicket, nil
}

// WecomSensitive 用 user_ticket 换得的敏感信息(不需应用通讯录权限)。
type WecomSensitive struct {
	Mobile string
	Email  string
	Avatar string
	Gender string
}

// UserSensitive 用 user_ticket 换手机/邮箱/头像等敏感信息(走 /auth/getuserdetail,靠 user_ticket 鉴权)。
func (c *WecomClient) UserSensitive(ctx context.Context, userTicket string) (*WecomSensitive, error) {
	if userTicket == "" {
		return &WecomSensitive{}, nil
	}
	accessToken, err := c.AccessToken(ctx)
	if err != nil {
		return nil, err
	}
	reqURL := fmt.Sprintf("%s/auth/getuserdetail?access_token=%s", wecomQyAPIBase, accessToken)
	var resp struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
		Mobile  string `json:"mobile"`
		Email   string `json:"email"`
		Avatar  string `json:"avatar"`
		Gender  string `json:"gender"`
	}
	if err := wecomPost(ctx, reqURL, map[string]string{"user_ticket": userTicket}, &resp); err != nil {
		return nil, fmt.Errorf("企微 getuserdetail 失败: %w", err)
	}
	if resp.ErrCode != 0 {
		return nil, fmt.Errorf("企微 getuserdetail 失败: errcode=%d errmsg=%s", resp.ErrCode, resp.ErrMsg)
	}
	return &WecomSensitive{Mobile: resp.Mobile, Email: resp.Email, Avatar: resp.Avatar, Gender: resp.Gender}, nil
}

// UserName 通过通讯录接口 /user/get 取用户姓名(需应用对该成员有通讯录读取权限)。
// 登录不依赖姓名,失败由调用方做非阻断降级(用 userid/mobile 兜底昵称)。
func (c *WecomClient) UserName(ctx context.Context, userID string) (string, error) {
	accessToken, err := c.AccessToken(ctx)
	if err != nil {
		return "", err
	}
	reqURL := fmt.Sprintf("%s/user/get?access_token=%s&userid=%s", wecomQyAPIBase, accessToken, userID)
	var resp struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
		Name    string `json:"name"`
	}
	if err := wecomGet(ctx, reqURL, &resp); err != nil {
		return "", err
	}
	if resp.ErrCode != 0 {
		return "", fmt.Errorf("errcode=%d errmsg=%s", resp.ErrCode, resp.ErrMsg)
	}
	return resp.Name, nil
}

// WecomProfile 企业微信用户资料(登录建号用)。
type WecomProfile struct {
	UserID string
	Name   string
	Mobile string
	Email  string
	Avatar string
}

// ProfileByCode 一站式:code → userid + user_ticket → 敏感信息 + 姓名。
// 敏感信息(getuserdetail)与姓名(user/get)均为非阻断:失败用 userid/mobile 兜底,
// 只要拿到 userid 即视为登录身份可用(企微非企业成员或不在应用可见范围时 userid 为空,直接报错)。
func (c *WecomClient) ProfileByCode(ctx context.Context, code string) (*WecomProfile, error) {
	userID, userTicket, err := c.UserIDByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if userID == "" {
		return nil, errors.New("企微未返回 userid(非企业成员或应用可见范围未覆盖)")
	}
	prof := &WecomProfile{UserID: userID}
	if sens, err := c.UserSensitive(ctx, userTicket); err == nil {
		prof.Mobile = sens.Mobile
		prof.Email = sens.Email
		prof.Avatar = sens.Avatar
	} else {
		logger.Bg().Mod("wecom").Err(err).Warn("拉取企微敏感信息失败,降级")
	}
	if name, err := c.UserName(ctx, userID); err == nil && name != "" {
		prof.Name = name
	}
	if prof.Name == "" {
		// 姓名兜底:手机号 → userid
		if prof.Mobile != "" {
			prof.Name = prof.Mobile
		} else {
			prof.Name = prof.UserID
		}
	}
	return prof, nil
}

// --- 通讯录同步(部门/成员拉取) ---

// WecomDepartment 企业微信部门(department/list 返回项)。
type WecomDepartment struct {
	Id       int64  `json:"id"`
	Name     string `json:"name"`
	ParentId int64  `json:"parentid"`
	Order    int64  `json:"order"`
}

// WecomContactUser 企业微信通讯录成员(user/list 返回项)。
// status:1=已激活 2=已禁用 4=未激活 5=退出企业; enable:1=启用 0=禁用;
// department 为该成员所属部门 id 列表(企微一成员可属多部门)。
type WecomContactUser struct {
	UserID     string  `json:"userid"`
	Name       string  `json:"name"`
	Mobile     string  `json:"mobile"`
	Email      string  `json:"email"`
	Avatar     string  `json:"avatar"`
	Department []int64 `json:"department"`
	Position   string  `json:"position"`
	Gender     string  `json:"gender"` // 企微性别(string,"0"未定义/"1"男/"2"女;真实值需 snsapi_privateinfo 授权,通讯录接口可能仅返回"0"或缺失),由 service 层 WecomGenderToSex 转换为项目 Sex 字典值(0男1女2未知)
	Status     int     `json:"status"`
	Enable     int     `json:"enable"`
}

// DepartmentList 拉取全量部门列表(需应用有通讯录读取权限,返回含根部门)。
// errcode 40014/42001(access_token 失效)清缓存重取重试一次(与 SendMessage 同口径)。
func (c *WecomClient) DepartmentList(ctx context.Context) ([]WecomDepartment, error) {
	for attempt := 0; ; attempt++ {
		accessToken, err := c.AccessToken(ctx)
		if err != nil {
			return nil, err
		}
		reqURL := fmt.Sprintf("%s/department/list?access_token=%s", wecomQyAPIBase, accessToken)
		var resp struct {
			ErrCode    int               `json:"errcode"`
			ErrMsg     string            `json:"errmsg"`
			Department []WecomDepartment `json:"department"`
		}
		if err = wecomGet(ctx, reqURL, &resp); err != nil {
			return nil, fmt.Errorf("企微 department/list 失败: %w", err)
		}
		if resp.ErrCode != 0 && attempt == 0 && (resp.ErrCode == 40014 || resp.ErrCode == 42001) {
			// token 失效(缓存提前过期兜底失准):清缓存重取重试一次
			global.OPS_CACHE.Delete(wecomAccessTokenCachePrefix + c.CorpID)
			continue
		}
		if resp.ErrCode != 0 {
			return nil, fmt.Errorf("企微 department/list 失败: errcode=%d errmsg=%s", resp.ErrCode, resp.ErrMsg)
		}
		return resp.Department, nil
	}
}

// DepartmentUsers 拉取指定部门成员详情列表。
// fetchChild=true 时递归子部门;逐部门拉取时建议 false,以便限速与去重可控。
// errcode 40014/42001(access_token 失效)清缓存重取重试一次(与 SendMessage 同口径)。
func (c *WecomClient) DepartmentUsers(ctx context.Context, deptId int64, fetchChild bool) ([]WecomContactUser, error) {
	fetch := 0
	if fetchChild {
		fetch = 1
	}
	for attempt := 0; ; attempt++ {
		accessToken, err := c.AccessToken(ctx)
		if err != nil {
			return nil, err
		}
		reqURL := fmt.Sprintf("%s/user/list?access_token=%s&department_id=%d&fetch_child=%d", wecomQyAPIBase, accessToken, deptId, fetch)
		var resp struct {
			ErrCode  int                `json:"errcode"`
			ErrMsg   string             `json:"errmsg"`
			UserList []WecomContactUser `json:"userlist"`
		}
		if err = wecomGet(ctx, reqURL, &resp); err != nil {
			return nil, fmt.Errorf("企微 user/list 失败: %w", err)
		}
		if resp.ErrCode != 0 && attempt == 0 && (resp.ErrCode == 40014 || resp.ErrCode == 42001) {
			global.OPS_CACHE.Delete(wecomAccessTokenCachePrefix + c.CorpID)
			continue
		}
		if resp.ErrCode != 0 {
			return nil, fmt.Errorf("企微 user/list 失败: errcode=%d errmsg=%s", resp.ErrCode, resp.ErrMsg)
		}
		return resp.UserList, nil
	}
}

// --- 应用消息推送(message/send) ---

// WecomMessageSendBatch 官方 message/send 单次 touser 上限,超出需调用方分批。
const WecomMessageSendBatch = 1000

// WecomTextCard 文本卡片消息(textcard)。URL 为空时 SendMessage 自动降级为纯文本消息。
type WecomTextCard struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Btntxt      string `json:"btntxt"`
}

// WecomMessageResp message/send 响应。errcode=0 即发送成功;invaliduser 非空表示
// 部分 userid 无效被服务端跳过(常见于不在应用可见范围),不视为整体失败。
type WecomMessageResp struct {
	ErrCode      int    `json:"errcode"`
	ErrMsg       string `json:"errmsg"`
	InvalidUser  string `json:"invaliduser"`
	InvalidParty string `json:"invalidparty"`
}

// SendMessage 向一批企业成员发送应用消息(统一 touser 展开,调用方保证单批 ≤1000)。
//   - msg.URL 为空 → 降级纯文本(content=Title+Description,适配未配可信域名的部署)
//   - 开启官方内容去重(enable_duplicate_check,1800s),与业务侧节流双保险防轰炸
//   - errcode 40014/42001(access_token 失效)清缓存重取重试一次;其余 errcode
//     (含 81013 userid 无效、60020 非可信 IP)返回带 errcode 的错误,由上层记日志放弃本批
func (c *WecomClient) SendMessage(ctx context.Context, touser []string, msg WecomTextCard) error {
	if len(touser) == 0 {
		return nil
	}
	if c.CorpID == "" || c.CorpSecret == "" || c.AgentID == 0 {
		return errors.New("企业微信 CorpId/AgentId/Secret 未配置")
	}
	for attempt := 0; ; attempt++ {
		accessToken, err := c.AccessToken(ctx)
		if err != nil {
			return err
		}
		reqURL := fmt.Sprintf("%s/message/send?access_token=%s", wecomQyAPIBase, accessToken)
		var resp WecomMessageResp
		if err = wecomPost(ctx, reqURL, c.buildSendBody(touser, msg), &resp); err != nil {
			return fmt.Errorf("企微 message/send 失败: %w", err)
		}
		if resp.ErrCode != 0 && attempt == 0 && (resp.ErrCode == 40014 || resp.ErrCode == 42001) {
			// token 失效(缓存提前过期兜底失准):清缓存重取重试一次
			global.OPS_CACHE.Delete(wecomAccessTokenCachePrefix + c.CorpID)
			continue
		}
		if resp.ErrCode != 0 {
			return fmt.Errorf("企微 message/send 失败: errcode=%d errmsg=%s invaliduser=%s", resp.ErrCode, resp.ErrMsg, resp.InvalidUser)
		}
		return nil
	}
}

// buildSendBody 组装 message/send 请求体:textcard(URL 非空)或降级 text(URL 空)。
func (c *WecomClient) buildSendBody(touser []string, msg WecomTextCard) map[string]any {
	body := map[string]any{
		"touser":                   strings.Join(touser, "|"),
		"agentid":                  c.AgentID,
		"enable_duplicate_check":   true,
		"duplicate_check_interval": 1800,
	}
	if msg.URL != "" {
		body["msgtype"] = "textcard"
		body["textcard"] = msg
	} else {
		body["msgtype"] = "text"
		content := msg.Title
		if msg.Description != "" {
			if content != "" {
				content += "\n"
			}
			content += msg.Description
		}
		body["text"] = map[string]string{"content": content}
	}
	return body
}

// SendBotMessage 向企微群机器人 webhook 发送 markdown 消息(每机器人限 20 条/分钟,调用方节流)。
// webhook URL 形如 https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx,由群内添加机器人获得。
func (c *WecomClient) SendBotMessage(ctx context.Context, webhookURL, markdown string) error {
	if webhookURL == "" {
		return errors.New("群机器人 webhook 未配置")
	}
	if markdown == "" {
		return errors.New("消息内容不能为空")
	}
	var resp struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := wecomPost(ctx, webhookURL, map[string]any{
		"msgtype":  "markdown",
		"markdown": map[string]string{"content": markdown},
	}, &resp); err != nil {
		return fmt.Errorf("企微群机器人发送失败: %w", err)
	}
	if resp.ErrCode != 0 {
		return fmt.Errorf("企微群机器人发送失败: errcode=%d errmsg=%s", resp.ErrCode, resp.ErrMsg)
	}
	return nil
}

// --- 通用 HTTP ---

func wecomGet(ctx context.Context, rawURL string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	resp, err := wecomHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status=%d body=%s", resp.StatusCode, wecomTruncate(string(body)))
	}
	return json.Unmarshal(body, out)
}

func wecomPost(ctx context.Context, rawURL string, body interface{}, out interface{}) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := wecomHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status=%d body=%s", resp.StatusCode, wecomTruncate(string(b)))
	}
	return json.Unmarshal(b, out)
}

func wecomTruncate(s string) string {
	const max = 512
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}

// --- WebView 免登辅助(纯函数,供 api 层回调使用) ---

// RandomWecomStateToken 生成符合企微 state 约束([a-zA-Z0-9]、≤128B)的一次性令牌(32 位 hex)。
func RandomWecomStateToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// SafeRedirectPath 校验并归一化登录后回跳路径,仅允许同源相对路径,防开放重定向。
// 规则:必须以 / 开头;禁止 // 与 /\ 防穿越;其余情况归一到 /。
func SafeRedirectPath(raw string) string {
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.HasPrefix(raw, "/\\") {
		return "/"
	}
	return raw
}

// 企业微信 WebView 免登回调设置的前端 localStorage 键。
// 须与前端 VITE_STORAGE_PREFIX(.env: OPS_)一致;前端改前缀需同步此处。
const (
	wecomWebviewAuthStorageKey    = "OPS_isAuthenticated"
	wecomWebviewExpiresStorageKey = "OPS_tokenExpiresAt"
)

// BuildWebviewLoginHTML 构造 WebView 免登回调 HTML:在 SPA 同源下设置 localStorage 登录态标志
// (前端路由守卫 getToken 依赖 OPS_isAuthenticated)与 tokenExpiresAt(scheduleProactiveRefresh 依赖),
// 随后 location.replace 跳回应用。redirect 经 json.Marshal 转义,防 JS/HTML 注入。
// expiresAt 为 access token 过期毫秒时间戳,与 PC 扫码轮询返回的 expiresAt 同源。
func BuildWebviewLoginHTML(redirect string, expiresAtMs int64) string {
	redirectJSON, _ := json.Marshal(redirect)
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>登录中</title>
<style>body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;display:flex;align-items:center;justify-content:center;height:100vh;margin:0;background:#f5f5f5;color:#666;font-size:16px}</style>
</head>
<body>
<p>企业微信登录中…</p>
<script>
try {
  localStorage.setItem('%s', 'true');
  localStorage.setItem('%s', '%d');
} catch (e) {}
window.location.replace(%s);
</script>
</body>
</html>`, wecomWebviewAuthStorageKey, wecomWebviewExpiresStorageKey, expiresAtMs, string(redirectJSON))
}
