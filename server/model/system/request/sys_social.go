package request

// SocialLoginForm 社交登录/绑定回调请求体(对齐前端 Api.Auth.SocialLoginForm)。
//
// 前端 social-callback/index.vue 在回调页组装该结构 POST 到 /auth/social/callback:
//   - 未登录走 authStore.login(登录流程);已登录走 fetchSocialLoginCallback(绑定流程)
//   - 后端不依赖前端 isLogin,统一用 state 里的 userId/intent 权威判断登录还是绑定
type SocialLoginForm struct {
	SocialCode  string `json:"socialCode"`  // 三方授权码 authorization_code
	SocialState string `json:"socialState"` // GetAuthURL 返回的 base64 state
	Source      string `json:"source"`      // wechat_open/gitee/github
	GrantType   string `json:"grantType"`   // 固定 "social"
	ClientId    string `json:"clientId"`    // 前端环境值(可选,当前不消费)
}

// SocialState state 载荷,base64 JSON 编码后随授权 URL 回传三方,回调时原样回传。
//   - 登录场景:仅 Domain + Source
//   - 绑定场景:Domain + Source + UserId + Intent="bind"
type SocialState struct {
	Domain string `json:"domain,omitempty"`
	Source string `json:"source,omitempty"`
	UserId int64  `json:"userId,omitempty"`
	Intent string `json:"intent,omitempty"` // "bind" 表示绑定
}
