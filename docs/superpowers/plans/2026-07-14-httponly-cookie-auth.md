# httpOnly Cookie 认证改造 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把本地 `server/`+`web/` 认证改造为 access+refresh 双 httpOnly cookie，删除 `x-token`/`Authorization`/localStorage token 等旧认证残留。

**Architecture:** 后端签发 access(refresh) 双 JWT（audience 区分），写入 `token`/`refresh-token` 两个 httpOnly cookie（HttpOnly+SameSite=Strict+动态 Secure）；取值优先 `Authorization: Bearer` 头其次 cookie、拒绝 query。鉴权失败统一 HTTP200+业务 code（9999 刷新/8888 登出）。前端 `withCredentials: true`、不带 Authorization 头，登录态靠可读信号 `isAuthenticated`+`tokenExpiresAt`，刷新走 cookie。

**Tech Stack:** Go + Gin + golang-jwt/v5（后端）；Vue3 + TS + axios/@sa/axios（前端，pnpm）。

## Global Constraints

- 语言：代码注释/提交信息一律**中文**；标识符、配置项、库名保持原文。
- 统一响应 `{code,data,msg}`，`code` 为字符串（成功 `"0000"`、失败 `"0001"`、刷新 `"9999"`、登出 `"8888"`）。
- cookie 属性固定：`HttpOnly=true`、`SameSite=Strict`、`Secure=RequestIsSecure(c)`、`Path=/`、`Domain=""`。
- **保留** `config.JWT.BufferTime` 字段与 `config.yaml` 的 `buffer-time: 1d`：`initialize/other.go` 启动时强制解析它，删除会 panic。
- `BaseClaims.ID` 保持 `uint`，不做 int64 迁移。
- 前端请求一律 `withCredentials: true`，`getAuthorization()` 恒返回 `null` 且为 null 时不设 Authorization 头。
- 每个任务结束 `go build ./...`（后端）或 `pnpm -C web typecheck`（前端）必须通过后再提交。
- 提交信息中文，按 task 粒度提交。

---

## 文件结构（改动总览）

**后端 `server/`**
| 文件 | 动作 | 职责 |
|---|---|---|
| `config/jwt.go` | 改 | 新增 `RefreshExTime` 字段 |
| `config.yaml` | 改 | jwt 段加 `refresh-ex-time: 168h` |
| `utils/jwt.go` | 改 | 加 access/refresh 双签 + ParseAccessToken/ParseRefreshToken + JoinBlacklist/IsBlacklisted；末任务删 CreateClaims/CreateTokenByOldToken/SetRedisJWT |
| `utils/claims.go` | 改 | 重写 GetToken(头→cookie→拒query)/GetClaims(ParseAccessToken)；加 RequestIsSecure；删 SetToken/ClearToken/LoginToken |
| `utils/cookie.go` | 新 | SetLoginCookies / ClearLoginCookies |
| `utils/helpers_test.go` | 新 | TestMain 初始化 global.OPS_CONFIG/BlackCache/OPS_LOG |
| `utils/jwt_test.go` | 新 | access/refresh audience 单测 |
| `utils/claims_test.go` | 新 | GetToken/RequestIsSecure 单测 |
| `model/common/response/response.go` | 改 | 加 NoAuthWithCode |
| `middleware/jwt.go` | 改 | 重写 JWTAuth（9999，删 new-token 头续期） |
| `middleware/cors.go` | 新 | Cors() credentials |
| `middleware/jwt_test.go` | 新 | 缺 token→9999、CORS 头单测 |
| `initialize/router.go` | 改 | 启用 middleware.Cors() |
| `service/system/sys_user.go` | 改 | Login 返回 access+refresh |
| `api/v1/system/sys_user.go` | 改 | Login 设 cookie+返回 expiresAt；新增 RefreshToken/Logout |
| `router/system/sys_base.go` | 改 | 挂 refreshToken/logout |
| `api/v1/system/sys_user_refresh_test.go` | 新 | refresh/logout cookie 行为单测 |

**前端 `web/`**
| 文件 | 动作 | 职责 |
|---|---|---|
| `src/service/request/index.ts` | 改 | withCredentials；onRequest 跳过 null Authorization |
| `src/service/request/shared.ts` | 改 | getAuthorization→null；handleRefreshToken 走 cookie；加主动刷新定时器 |
| `src/store/modules/auth/shared.ts` | 改 | getToken 基于 isAuthenticated；clearAuthStorage |
| `src/store/modules/auth/index.ts` | 改 | 登录/重置/初始化改 isAuthenticated+tokenExpiresAt 信号 |
| `src/service/api/auth.ts` | 改 | fetchLogin 返回 {expiresAt}；fetchRefreshToken 无 body |
| `src/typings/api/auth.d.ts` | 改 | LoginToken→{expiresAt}；PwdLoginForm 收敛 |
| `src/views/_builtin/login/modules/pwd-login.vue` | 改 | 入参对齐 {username,password} |

**文档**
| 文件 | 动作 |
|---|---|
| `aiDoc/frontend-backend/boundary.md` | 改：补 cookie 认证契约 |

---

## Task 1: 后端 JWT 双 token 与配置

**Files:**
- Modify: `server/config/jwt.go`
- Modify: `server/config.yaml`（jwt 段）
- Modify: `server/utils/jwt.go`
- Create: `server/utils/helpers_test.go`
- Create: `server/utils/jwt_test.go`

**Interfaces:**
- Consumes: `global.OPS_CONFIG.JWT.{SigningKey,ExpiresTime,RefreshExTime,Issuer}`、`global.BlackCache`
- Produces: `utils.JWT.CreateAccessToken(BaseClaims)(string,error)`、`CreateRefreshToken(BaseClaims)(string,error)`、`ParseAccessToken(string)(*CustomClaims,error)`、`ParseRefreshToken(string)(*CustomClaims,error)`、`utils.JoinBlacklist(string)`、`utils.IsBlacklisted(string)`、常量 `AudienceAccess="access"`/`AudienceRefresh="refresh"`

> 本任务只**新增**方法，保留旧 `CreateClaims/CreateTokenByOldToken/SetRedisJWT` 不动，保证中间编译通过；它们在 Task 4 末尾统一删除。

- [ ] **Step 1: 改 config/jwt.go 加 RefreshExTime**

整文件替换为：
```go
package config

type JWT struct {
	SigningKey    string `mapstructure:"signing-key" json:"signing-key" yaml:"signing-key"`           // jwt 签名
	ExpiresTime   string `mapstructure:"expires-time" json:"expires-time" yaml:"expires-time"`        // access token 过期时间
	RefreshExTime string `mapstructure:"refresh-ex-time" json:"refresh-ex-time" yaml:"refresh-ex-time"` // refresh token 过期时间
	BufferTime    string `mapstructure:"buffer-time" json:"buffer-time" yaml:"buffer-time"`           // 已废弃（原滑动续期），保留字段以免 OtherInit 解析 panic
	Issuer        string `mapstructure:"issuer" json:"issuer" yaml:"issuer"`                          // 签发者
}
```

- [ ] **Step 2: 改 config.yaml jwt 段加 refresh-ex-time**

把 `server/config.yaml` 的：
```yaml
jwt:
    signing-key: 2aa81975-ecfd-44bd-817b-809152efaafb
    expires-time: 7d
    buffer-time: 1d
    issuer: qmPlus
```
改为（在 `expires-time` 下一行插入 `refresh-ex-time`）：
```yaml
jwt:
    signing-key: 2aa81975-ecfd-44bd-817b-809152efaafb
    expires-time: 7d
    refresh-ex-time: 168h
    buffer-time: 1d
    issuer: qmPlus
```

- [ ] **Step 3: 改 utils/jwt.go，新增双 token 方法**

在 `server/utils/jwt.go` 文件**顶部**（`type JWT struct` 之前）插入常量与新增错误：
```go
const (
	AudienceAccess  = "access"
	AudienceRefresh = "refresh"
)
```
在 `var ( ... )` 错误块末尾追加：
```go
	TokenAudienceMismatch = errors.New("token受众不匹配")
```
在 `NewJWT` 函数之后、`CreateToken` 之前，插入：
```go
// createClaims 构造指定 audience 与过期时长的 claims。
func (j *JWT) createClaims(bc request.BaseClaims, audience string, exp time.Duration) request.CustomClaims {
	return request.CustomClaims{
		BaseClaims: bc,
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{audience},
			NotBefore: jwt.NewNumericDate(time.Now().Add(-1000 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(exp)),
			Issuer:    global.OPS_CONFIG.JWT.Issuer,
		},
	}
}

// CreateAccessToken 签发 access token（audience=access，业务接口鉴权用）。
func (j *JWT) CreateAccessToken(bc request.BaseClaims) (string, error) {
	ep, err := ParseDuration(global.OPS_CONFIG.JWT.ExpiresTime)
	if err != nil || ep <= 0 {
		ep = 7 * 24 * time.Hour
	}
	return j.CreateToken(j.createClaims(bc, AudienceAccess, ep))
}

// CreateRefreshToken 签发 refresh token（audience=refresh，仅 /auth/refreshToken 使用）。
func (j *JWT) CreateRefreshToken(bc request.BaseClaims) (string, error) {
	rp, err := ParseDuration(global.OPS_CONFIG.JWT.RefreshExTime)
	if err != nil || rp <= 0 {
		rp = 168 * time.Hour
	}
	return j.CreateToken(j.createClaims(bc, AudienceRefresh, rp))
}
```
在 `ParseToken` 函数之后，插入：
```go
// ParseAccessToken 解析并强制 audience=access；业务接口用，拒绝 refresh token。
func (j *JWT) ParseAccessToken(tokenString string) (*request.CustomClaims, error) {
	claims, err := j.ParseToken(tokenString)
	if err != nil {
		return nil, err
	}
	if !hasAudience(claims, AudienceAccess) {
		return nil, TokenAudienceMismatch
	}
	return claims, nil
}

// ParseRefreshToken 解析并强制 audience=refresh。
func (j *JWT) ParseRefreshToken(tokenString string) (*request.CustomClaims, error) {
	claims, err := j.ParseToken(tokenString)
	if err != nil {
		return nil, err
	}
	if !hasAudience(claims, AudienceRefresh) {
		return nil, TokenAudienceMismatch
	}
	return claims, nil
}

func hasAudience(claims *request.CustomClaims, audience string) bool {
	for _, a := range claims.Audience {
		if a == audience {
			return true
		}
	}
	return false
}

// JoinBlacklist 将 token 加入黑名单（内存缓存，进程级；进程重启失效）。
func JoinBlacklist(token string) {
	if token == "" {
		return
	}
	global.BlackCache.SetDefault(token, struct{}{})
}

// IsBlacklisted 判断 token 是否在黑名单。
func IsBlacklisted(token string) bool {
	if token == "" {
		return false
	}
	_, ok := global.BlackCache.Get(token)
	return ok
}
```

- [ ] **Step 4: 新建 utils/helpers_test.go（测试全局初始化）**

创建 `server/utils/helpers_test.go`：
```go
package utils

import (
	"os"
	"testing"
	"time"

	"github.com/hllkk/devops-admin/server/config"
	"github.com/hllkk/devops-admin/server/global"
	"github.com/songzhibin97/gkit/cache/local_cache"
	"go.uber.org/zap"
)

// TestMain 为 utils 包测试初始化全局依赖（NewJWT 读 OPS_CONFIG，黑名单读 BlackCache，GetClaims 读 OPS_LOG）。
func TestMain(m *testing.M) {
	global.OPS_CONFIG = config.Server{
		JWT: config.JWT{
			SigningKey:    "test-signing-key",
			ExpiresTime:   "1h",
			RefreshExTime: "168h",
			BufferTime:    "1d",
			Issuer:        "test",
		},
	}
	global.BlackCache = local_cache.NewCache(local_cache.SetDefaultExpire(time.Hour))
	global.OPS_LOG = zap.NewNop()
	os.Exit(m.Run())
}
```

- [ ] **Step 5: 新建 utils/jwt_test.go（audience 强制校验）**

创建 `server/utils/jwt_test.go`：
```go
package utils

import (
	"errors"

	"github.com/hllkk/devops-admin/server/model/system/request"
	"testing"
)

func TestAccessTokenAcceptedRefreshRejected(t *testing.T) {
	j := NewJWT()
	bc := request.BaseClaims{ID: 1, Username: "admin"}

	access, err := j.CreateAccessToken(bc)
	if err != nil {
		t.Fatalf("CreateAccessToken: %v", err)
	}
	refresh, err := j.CreateRefreshToken(bc)
	if err != nil {
		t.Fatalf("CreateRefreshToken: %v", err)
	}

	// access token 可被业务接口解析
	if _, err := j.ParseAccessToken(access); err != nil {
		t.Fatalf("ParseAccessToken(access) 应通过，got %v", err)
	}
	// refresh token 不得访问业务接口
	if _, err := j.ParseAccessToken(refresh); !errors.Is(err, TokenAudienceMismatch) {
		t.Fatalf("ParseAccessToken(refresh) 应返回 TokenAudienceMismatch，got %v", err)
	}
	// refresh token 仅 refresh 端点可解析
	if _, err := j.ParseRefreshToken(refresh); err != nil {
		t.Fatalf("ParseRefreshToken(refresh) 应通过，got %v", err)
	}
	// access token 不能当 refresh 用
	if _, err := j.ParseRefreshToken(access); !errors.Is(err, TokenAudienceMismatch) {
		t.Fatalf("ParseRefreshToken(access) 应返回 TokenAudienceMismatch，got %v", err)
	}
}

func TestBlacklist(t *testing.T) {
	JoinBlacklist("tok-1")
	if !IsBlacklisted("tok-1") {
		t.Fatal("IsBlacklisted 应命中 tok-1")
	}
	if IsBlacklisted("tok-2") {
		t.Fatal("IsBlacklisted 不应命中 tok-2")
	}
	if IsBlacklisted("") {
		t.Fatal("空 token 不应命中黑名单")
	}
}
```

- [ ] **Step 6: 跑测试，确认通过**

Run: `cd server && go test ./utils/ -run 'TestAccessTokenAcceptedRefreshRejected|TestBlacklist' -v`
Expected: PASS（两个用例通过）。

- [ ] **Step 7: 构建并提交**

Run: `cd server && go build ./...`
Expected: 无报错。

```bash
cd /home/devops-admin
git add server/config/jwt.go server/config.yaml server/utils/jwt.go server/utils/helpers_test.go server/utils/jwt_test.go
git commit -m "feat(jwt): 签发 access/refresh 双 token（audience 区分）+ 黑名单工具，新增 refresh-ex-time 配置"
```

---

## Task 2: 后端 token 提取与 cookie 工具

**Files:**
- Modify: `server/utils/claims.go`（整文件重写）
- Create: `server/utils/cookie.go`
- Create: `server/utils/claims_test.go`

**Interfaces:**
- Consumes: Task1 的 `NewJWT/ParseAccessToken`、`global.OPS_LOG`
- Produces: `utils.GetToken(c)(*gin.Context)(string,error)`、`utils.RequestIsSecure(c)bool`、`utils.GetClaims(c)(*CustomClaims,error)`、`utils.SetLoginCookies(c,access,refresh)`、`utils.ClearLoginCookies(c)`；保留 `GetUserID/GetUserUuid/GetUserAuthorityId/GetUserInfo/GetUserName`
- **本任务保留** 旧 `SetToken/ClearToken` 不删（middleware 仍调用它们），删除推迟到 Task 3（middleware 重写之后），保证每任务可独立构建。

- [ ] **Step 1: 整文件重写 utils/claims.go**

替换 `server/utils/claims.go` 全文为：
```go
package utils

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hllkk/devops-admin/server/global"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
)

// GetToken 取 access token：Authorization: Bearer 头优先 → token httpOnly cookie。
// 禁止从 URL 查询参数取 token，避免 JWT 进入 URL（Nginx 日志/浏览器历史/Referer 泄漏）。
func GetToken(c *gin.Context) (string, error) {
	if authorization := c.Request.Header.Get("Authorization"); authorization != "" {
		if !strings.HasPrefix(strings.ToLower(authorization), "bearer ") {
			return "", fmt.Errorf("invalid authorization header format, expected 'Bearer <token>', got: %q", authorization)
		}
		return authorization[7:], nil
	}
	if token, err := c.Cookie("token"); err == nil && token != "" {
		return token, nil
	}
	return "", errors.New("token not found in header or cookie")
}

// RequestIsSecure 判断请求是否经 HTTPS 传输，用于动态决定 cookie 的 Secure 标志。
// 优先 X-Forwarded-Proto（多层反代反映浏览器真实协议），回退 c.Request.TLS。
func RequestIsSecure(c *gin.Context) bool {
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		return strings.EqualFold(proto, "https")
	}
	return c.Request.TLS != nil
}

// GetClaims 取 access token 并解析为 claims（业务接口强制 access，拒绝 refresh token）。
func GetClaims(c *gin.Context) (*systemReq.CustomClaims, error) {
	token, err := GetToken(c)
	if err != nil {
		return nil, err
	}
	j := NewJWT()
	claims, err := j.ParseAccessToken(token)
	if err != nil {
		global.OPS_LOG.Error("从请求中解析 access token 失败，请检查 Authorization 头或 token cookie")
		return nil, err
	}
	return claims, nil
}

// GetUserID 从 Context 中获取 jwt 用户ID。
func GetUserID(c *gin.Context) uint {
	if claims, exists := c.Get("claims"); exists {
		return claims.(*systemReq.CustomClaims).BaseClaims.ID
	}
	if cl, err := GetClaims(c); err == nil {
		return cl.BaseClaims.ID
	}
	return 0
}

// GetUserUuid 从 Context 中获取 jwt 用户UUID。
func GetUserUuid(c *gin.Context) uuid.UUID {
	if claims, exists := c.Get("claims"); exists {
		return claims.(*systemReq.CustomClaims).UUID
	}
	if cl, err := GetClaims(c); err == nil {
		return cl.UUID
	}
	return uuid.UUID{}
}

// GetUserAuthorityId 从 Context 中获取 jwt 角色ID。
func GetUserAuthorityId(c *gin.Context) uint {
	if claims, exists := c.Get("claims"); exists {
		return claims.(*systemReq.CustomClaims).RoleId
	}
	if cl, err := GetClaims(c); err == nil {
		return cl.RoleId
	}
	return 0
}

// GetUserInfo 从 Context 中获取完整 claims。
func GetUserInfo(c *gin.Context) *systemReq.CustomClaims {
	if claims, exists := c.Get("claims"); exists {
		return claims.(*systemReq.CustomClaims)
	}
	if cl, err := GetClaims(c); err == nil {
		return cl
	}
	return nil
}

// GetUserName 从 Context 中获取用户名。
func GetUserName(c *gin.Context) string {
	if claims, exists := c.Get("claims"); exists {
		return claims.(*systemReq.CustomClaims).Username
	}
	if cl, err := GetClaims(c); err == nil {
		return cl.Username
	}
	return ""
}

// SetToken 已废弃（旧 x-token cookie 半成品）。仅临时保留供 middleware 过渡期编译，
// Task 3 重写 middleware 后会删除。新代码用 SetLoginCookies。
//
//nolint:unused
func SetToken(c *gin.Context, token string, maxAge int) {
	host, _, err := net.SplitHostPort(c.Request.Host)
	if err != nil {
		host = c.Request.Host
	}
	if net.ParseIP(host) != nil {
		c.SetCookie("x-token", token, maxAge, "/", "", false, false)
	} else {
		c.SetCookie("x-token", token, maxAge, "/", host, false, false)
	}
}

// ClearToken 已废弃（同 SetToken）。Task 3 删除。
//
//nolint:unused
func ClearToken(c *gin.Context) {
	host, _, err := net.SplitHostPort(c.Request.Host)
	if err != nil {
		host = c.Request.Host
	}
	if net.ParseIP(host) != nil {
		c.SetCookie("x-token", "", -1, "/", "", false, false)
	} else {
		c.SetCookie("x-token", "", -1, "/", host, false, false)
	}
}
```

- [ ] **Step 2: 新建 utils/cookie.go**

创建 `server/utils/cookie.go`：
```go
package utils

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/global"
)

// SetLoginCookies 写入 access/refresh 两个 httpOnly cookie。
// 属性：HttpOnly=true、SameSite=Strict、Secure=动态(RequestIsSecure)、Path=/、Domain=""。
func SetLoginCookies(c *gin.Context, accessToken, refreshToken string) {
	secure := RequestIsSecure(c)
	c.SetSameSite(http.SameSiteStrictMode)

	c.SetCookie("token", accessToken, int(accessExpirySeconds().Seconds()), "/", "", secure, true)
	c.SetCookie("refresh-token", refreshToken, int(refreshExpirySeconds().Seconds()), "/", "", secure, true)
}

// ClearLoginCookies 清除登录 cookie（与 Set 时同 SameSite/Secure，确保能被覆盖清除）。
func ClearLoginCookies(c *gin.Context) {
	secure := RequestIsSecure(c)
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("token", "", -1, "/", "", secure, true)
	c.SetCookie("refresh-token", "", -1, "/", "", secure, true)
}

func accessExpirySeconds() time.Duration {
	dr, err := ParseDuration(global.OPS_CONFIG.JWT.ExpiresTime)
	if err != nil || dr <= 0 {
		dr = 7 * 24 * time.Hour
	}
	return dr
}

func refreshExpirySeconds() time.Duration {
	dr, err := ParseDuration(global.OPS_CONFIG.JWT.RefreshExTime)
	if err != nil || dr <= 0 {
		dr = 168 * time.Hour
	}
	return dr
}
```

- [ ] **Step 3: 新建 utils/claims_test.go**

创建 `server/utils/claims_test.go`：
```go
package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newContext(req *http.Request) *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	return c
}

func TestGetTokenFromHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer abc123")
	c := newContext(req)
	got, err := GetToken(c)
	if err != nil || got != "abc123" {
		t.Fatalf("header 取值期望 abc123/noerr，got %q/%v", got, err)
	}
}

func TestGetTokenFromCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: "cookie-tok"})
	c := newContext(req)
	got, err := GetToken(c)
	if err != nil || got != "cookie-tok" {
		t.Fatalf("cookie 取值期望 cookie-tok/noerr，got %q/%v", got, err)
	}
}

func TestGetTokenRejectsQuery(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?token=qq", nil)
	c := newContext(req)
	if _, err := GetToken(c); err == nil {
		t.Fatal("query 参数取 token 必须被拒绝")
	}
}

func TestGetTokenMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := newContext(req)
	if _, err := GetToken(c); err == nil {
		t.Fatal("无 token 必须返回 error")
	}
}

func TestRequestIsSecure(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	if !RequestIsSecure(newContext(req)) {
		t.Fatal("X-Forwarded-Proto=https 应判定 secure")
	}
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("X-Forwarded-Proto", "http")
	if RequestIsSecure(newContext(req2)) {
		t.Fatal("X-Forwarded-Proto=http 应判定非 secure")
	}
}
```

- [ ] **Step 4: 跑测试**

Run: `cd server && go test ./utils/ -v`
Expected: 全部 PASS（含 Task1 的用例）。

- [ ] **Step 5: 构建并提交**

Run: `cd server && go build ./...`
Expected: 无报错（旧 SetToken/ClearToken 已在本任务保留，middleware 仍可编译）。

```bash
cd /home/devops-admin
git add server/utils/claims.go server/utils/cookie.go server/utils/claims_test.go
git commit -m "feat(auth): 重写 token 提取(头→cookie→拒query)+RequestIsSecure，新增 httpOnly cookie 写入/清除工具"
```

---

## Task 3: 后端鉴权失败契约 + 中间件 + CORS

**Files:**
- Modify: `server/model/common/response/response.go`
- Modify: `server/middleware/jwt.go`（整文件重写）
- Create: `server/middleware/cors.go`
- Modify: `server/initialize/router.go`
- Create: `server/middleware/jwt_test.go`

**Interfaces:**
- Consumes: Task1/2 的 `utils.GetToken/IsBlacklisted/NewJWT/ParseAccessToken`、`response.Result`
- Produces: `response.NoAuthWithCode(code,msg,c)`、`middleware.JWTAuth()`（返回 9999）、`middleware.Cors()`

- [ ] **Step 1: response.go 加 NoAuthWithCode**

在 `server/model/common/response/response.go` 的 `NoAuth` 函数之后插入：
```go
// NoAuthWithCode 鉴权失败：HTTP 200 + 业务 code（不用 401，以便前端按 code 分流刷新/登出）。
// code "9999"=EXPIRED_TOKEN_CODES（前端刷新）；"8888"=LOGOUT_CODES（前端登出）。
func NoAuthWithCode(code, message string, c *gin.Context) {
	Result(code, nil, message, c)
}
```

- [ ] **Step 2: 整文件重写 middleware/jwt.go**

替换 `server/middleware/jwt.go` 全文为：
```go
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/response"
	"github.com/hllkk/devops-admin/server/utils"
)

// expiredTokenCode 命中前端 VITE_SERVICE_EXPIRED_TOKEN_CODES，触发刷新；刷新失败由 refresh 端点返回 8888 登出。
const expiredTokenCode = "9999"

// JWTAuth 校验 access token：从 Authorization 头或 token cookie 取值，强制 audience=access，校验黑名单。
// 失败统一 HTTP 200 + code "9999"（前端据此刷新或登出）。不再下发 new-token 头（改由 refresh 端点轮换）。
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := utils.GetToken(c)
		if err != nil || token == "" {
			response.NoAuthWithCode(expiredTokenCode, "未登录或令牌失效，请登录", c)
			c.Abort()
			return
		}
		if utils.IsBlacklisted(token) {
			response.NoAuthWithCode(expiredTokenCode, "令牌已失效，请重新登录", c)
			c.Abort()
			return
		}
		j := utils.NewJWT()
		claims, err := j.ParseAccessToken(token)
		if err != nil {
			response.NoAuthWithCode(expiredTokenCode, "登录已过期，请重新登录", c)
			c.Abort()
			return
		}
		c.Set("claims", claims)
		c.Next()
	}
}
```

- [ ] **Step 3: 删除 utils/claims.go 中已废弃的 SetToken/ClearToken**

middleware 已在 Step 2 重写、不再调用 `utils.SetToken/ClearToken`，现在删除它们。打开 `server/utils/claims.go`，删除 `SetToken` 与 `ClearToken` 两个函数（Task 2 标注 `//nolint:unused` 的那两个），并移除随之不再使用的 `"net"` import。

验证无残留引用：
Run: `cd server && grep -rn "utils.SetToken\|utils.ClearToken\|\.SetToken(\|\.ClearToken(" --include=*.go . | grep -v _test`
Expected: 无输出。

- [ ] **Step 4: 新建 middleware/cors.go**

创建 `server/middleware/cors.go`：
```go
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Cors 放行跨域并允许携带 cookie（credentials）。
// Origin 动态回显请求来源，配合 httpOnly cookie + 前端 withCredentials。
// 生产环境建议在外层网关收敛白名单。
func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		if origin := c.Request.Header.Get("Origin"); origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Token,X-User-Id,x-request-id,apifoxtoken")
			c.Header("Access-Control-Allow-Methods", "POST,GET,PUT,DELETE,OPTIONS")
			c.Header("Access-Control-Expose-Headers", "Content-Length,Content-Type,New-Token,New-Expires-At,Download-Filename")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
```

- [ ] **Step 5: initialize/router.go 启用 CORS**

在 `server/initialize/router.go` 的 `Routers()` 中，把：
```go
	Router.Use(middleware.GinRecovery(true))
	if gin.Mode() == gin.DebugMode {
		Router.Use(gin.Logger())
	}
```
改为（在其后加一行启用 CORS）：
```go
	Router.Use(middleware.GinRecovery(true))
	if gin.Mode() == gin.DebugMode {
		Router.Use(gin.Logger())
	}
	Router.Use(middleware.Cors()) // 跨域带 cookie（credentials），httpOnly cookie 认证必需
```
（`middleware` 已在该文件 import，无需改 import。）

- [ ] **Step 6: 新建 middleware/jwt_test.go**

创建 `server/middleware/jwt_test.go`：
```go
package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/config"
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system/request"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/songzhibin97/gkit/cache/local_cache"
	"go.uber.org/zap"
)

func init() {
	global.OPS_CONFIG = config.Server{JWT: config.JWT{SigningKey: "mw-key", ExpiresTime: "1h", RefreshExTime: "168h", Issuer: "t"}}
	global.BlackCache = local_cache.NewCache(local_cache.SetDefaultExpire(3600 * 1e9))
	global.OPS_LOG = zap.NewNop()
}

func do(req *http.Request, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r := gin.New()
	r.Use(handler)
	r.Any("/x", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	r.ServeHTTP(w, req)
	return w
}

func TestJWTAuthMissingTokenReturns9999(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	w := do(req, JWTAuth())
	var body struct {
		Code string `json:"code"`
	}
	_ = json.NewDecoder(w.Body).Decode(&body)
	if body.Code != "9999" {
		t.Fatalf("缺 token 应返回 code 9999，got %s", body.Code)
	}
}

func TestJWTAuthValidAccessPasses(t *testing.T) {
	j := utils.NewJWT()
	tok, _ := j.CreateAccessToken(request.BaseClaims{ID: 7, Username: "u"})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := do(req, JWTAuth())
	if w.Code != http.StatusOK {
		t.Fatalf("有效 access token 应放行到下一 handler(200)，got %d body=%s", w.Code, w.Body.String())
	}
}

func TestJWTAuthRefreshTokenRejected(t *testing.T) {
	j := utils.NewJWT()
	tok, _ := j.CreateRefreshToken(request.BaseClaims{ID: 7, Username: "u"})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := do(req, JWTAuth())
	var body struct {
		Code string `json:"code"`
	}
	_ = json.NewDecoder(w.Body).Decode(&body)
	if body.Code != "9999" {
		t.Fatalf("refresh token 访问业务接口应被拒(9999)，got %s", body.Code)
	}
}

func TestCorsHeadersOnPreflight(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/x", nil)
	req.Header.Set("Origin", "http://localhost:8080")
	w := do(req, Cors())
	if w.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS 预检应 204，got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatal("预检应回显 Allow-Credentials: true")
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:8080" {
		t.Fatalf("预检应回显 Origin，got %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
}
```

- [ ] **Step 7: 跑测试与构建**

Run: `cd server && go test ./middleware/ ./utils/ -v`
Expected: 全 PASS。

Run: `cd server && go build ./...`
Expected: 无报错（middleware 已不再引用旧 SetToken/ClearToken）。

- [ ] **Step 8: 提交**

```bash
cd /home/devops-admin
git add server/model/common/response/response.go server/middleware/jwt.go server/middleware/cors.go server/middleware/jwt_test.go server/initialize/router.go server/utils/claims.go
git commit -m "feat(auth): 鉴权失败改 HTTP200+code(9999刷新/8888登出)，重写 JWTAuth，新增 CORS(credentials) 中间件"
```

---

## Task 4: 后端 Login/Refresh/Logout 端点 + 清理废弃 JWT 方法

**Files:**
- Modify: `server/service/system/sys_user.go`（Login）
- Modify: `server/api/v1/system/sys_user.go`（Login + 新增 RefreshToken/Logout）
- Modify: `server/router/system/sys_base.go`
- Modify: `server/utils/jwt.go`（删 CreateClaims/CreateTokenByOldToken/SetRedisJWT）
- Create: `server/api/v1/system/sys_user_refresh_test.go`

**Interfaces:**
- Consumes: `userService.Login(username,password)(access,refresh,user,err)`、`utils.SetLoginCookies/ClearLoginCookies/JoinBlacklist/GetToken/ParseRefreshToken/CreateAccessToken/CreateRefreshToken`
- Produces: HTTP `POST /auth/login`→`{expiresAt}`、`POST /auth/refreshToken`→`{expiresAt}`（失败 8888）、`POST /auth/logout`→ok

- [ ] **Step 1: 改 service/system/sys_user.go 的 Login**

把 `server/service/system/sys_user.go` 中 `Login` 函数体（从 `func (s *UserService) Login(...)` 到该函数结束 `}`）替换为：
```go
// Login 校验用户名密码，签发 access + refresh token，返回两串 + 用户实体。
func (s *UserService) Login(username, password string) (accessToken, refreshToken string, user system.SysUser, err error) {
	if err = global.OPS_DB.Where("user_name = ?", username).First(&user).Error; err != nil {
		return "", "", system.SysUser{}, errors.New("用户不存在或密码错误")
	}
	if user.Status != "0" {
		return "", "", system.SysUser{}, errors.New("账号已停用")
	}
	if !utils.BcryptCheck(password, user.Password) {
		return "", "", system.SysUser{}, errors.New("用户不存在或密码错误")
	}
	_, isSuper, firstRoleId := s.getUserRoleIds(int64(user.UserId))
	bc := request.BaseClaims{
		ID:         uint(user.UserId),
		Username:   user.UserName,
		NickName:   user.NickName,
		RoleId:     firstRoleId,
		SuperAdmin: isSuper,
	}
	j := utils.NewJWT()
	if accessToken, err = j.CreateAccessToken(bc); err != nil {
		return "", "", system.SysUser{}, err
	}
	if refreshToken, err = j.CreateRefreshToken(bc); err != nil {
		return "", "", system.SysUser{}, err
	}
	return accessToken, refreshToken, user, nil
}
```
并把该函数上方两段注释（`// Login 校验用户名密码...` 之前的对照说明注释块）替换为一行：
```go
// Login 校验用户名密码，签发 access + refresh 双 token（httpOnly cookie 由 API 层写入）。
```
（确保不再出现 `refreshToken = token` 占位。）

- [ ] **Step 2: 改 api/v1/system/sys_user.go 的 Login + 新增 RefreshToken/Logout**

把 `server/api/v1/system/sys_user.go` 中 `Login` 函数（含其 Swagger 注释）整体替换为：
```go
const logoutTokenCode = "8888"

// Login
// @Tags     Base
// @Summary  用户登录
// @Produce   application/json
// @Param    data  body      systemReq.Login  true  "用户名, 密码"
// @Success  200   {object}  response.Response{data=object,msg=string}  "access/refresh 写入 httpOnly cookie，返回 expiresAt(毫秒)"
// @Router   /auth/login [post]
func (b *BaseApi) Login(c *gin.Context) {
	var l systemReq.Login
	if err := c.ShouldBindJSON(&l); err != nil {
		response.FailWithMessage("参数校验不通过", c)
		return
	}
	if err := utils.Verify(l, utils.LoginVerify); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	access, refresh, _, err := userService.Login(l.Username, l.Password)
	if err != nil {
		global.OPS_LOG.Warn("登录失败", zap.String("user", l.Username), zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	utils.SetLoginCookies(c, access, refresh)
	j := utils.NewJWT()
	claims, _ := j.ParseAccessToken(access)
	expiresAt := int64(0)
	if claims != nil && claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Unix() * 1000
	}
	response.OkWithDetailed(gin.H{"expiresAt": expiresAt}, "登录成功", c)
}

// RefreshToken
// @Tags     Base
// @Summary  刷新令牌
// @Produce   application/json
// @Success  200  {object}  response.Response{data=object,msg=string}  "用 refresh-token cookie 换发新 access/refresh，返回 expiresAt(毫秒)；失败 code=8888"
// @Router   /auth/refreshToken [post]
func (b *BaseApi) RefreshToken(c *gin.Context) {
	refresh, err := c.Cookie("refresh-token")
	if err != nil || refresh == "" {
		response.NoAuthWithCode(logoutTokenCode, "refresh token 不存在，请重新登录", c)
		return
	}
	j := utils.NewJWT()
	claims, err := j.ParseRefreshToken(refresh)
	if err != nil || utils.IsBlacklisted(refresh) {
		response.NoAuthWithCode(logoutTokenCode, "refresh token 已失效，请重新登录", c)
		return
	}
	bc := claims.BaseClaims
	access, err := j.CreateAccessToken(bc)
	if err != nil {
		response.NoAuthWithCode(logoutTokenCode, "令牌刷新失败，请重新登录", c)
		return
	}
	newRefresh, err := j.CreateRefreshToken(bc)
	if err != nil {
		response.NoAuthWithCode(logoutTokenCode, "令牌刷新失败，请重新登录", c)
		return
	}
	utils.JoinBlacklist(refresh)
	if oldAccess, e := utils.GetToken(c); e == nil && oldAccess != "" {
		utils.JoinBlacklist(oldAccess)
	}
	utils.SetLoginCookies(c, access, newRefresh)
	expiresAt := int64(0)
	if ac, perr := j.ParseAccessToken(access); perr == nil && ac != nil && ac.ExpiresAt != nil {
		expiresAt = ac.ExpiresAt.Unix() * 1000
	}
	response.OkWithDetailed(gin.H{"expiresAt": expiresAt}, "刷新成功", c)
}

// Logout
// @Tags     Base
// @Summary  退出登录
// @Produce   application/json
// @Success  200  {object}  response.Response  "清除登录 cookie，当前 token 入黑名单"
// @Router   /auth/logout [post]
func (b *BaseApi) Logout(c *gin.Context) {
	utils.ClearLoginCookies(c)
	if token, err := utils.GetToken(c); err == nil && token != "" {
		utils.JoinBlacklist(token)
	}
	response.OkWithMessage("退出成功", c)
}
```
> `systemReq`、`utils`、`zap`、`global`、`response` 已在原文件 import；`gin.H` 来自已有的 `gin` import。无需改 import 块。

- [ ] **Step 3: 改 router/system/sys_base.go 挂新路由**

把 `server/router/system/sys_base.go` 的 `InitBaseRouter` 函数体替换为：
```go
func (s *BaseRouter) InitBaseRouter(public, private *gin.RouterGroup) {
	pub := public.Group("auth")
	{
		pub.POST("login", baseApi.Login)
		pub.POST("code", baseApi.Captcha)
		pub.POST("refreshToken", baseApi.RefreshToken)
		pub.POST("logout", baseApi.Logout)
	}
	// private 组的 /auth/getUserInfo（需鉴权）
	pri := private.Group("auth")
	{
		pri.GET("getUserInfo", baseApi.GetUserInfo)
	}
}
```

- [ ] **Step 4: 删除 utils/jwt.go 中已无引用的旧方法**

从 `server/utils/jwt.go` 删除以下三个函数（Task 1 保留、现调用方均已切换）：
- `CreateClaims(baseClaims request.BaseClaims) request.CustomClaims`（整段，含其上注释）
- `CreateTokenByOldToken(oldToken string, claims request.CustomClaims) (string, error)`（整段，含注释）
- `SetRedisJWT(jwt string, userName string) (err error)`（整段，含注释）

删除后验证无残留引用：
Run: `cd server && grep -rn "CreateClaims\|CreateTokenByOldToken\|SetRedisJWT" --include=*.go . | grep -v _test`
Expected: 无输出（`GetRedisJWT` 在 `service/system/jwt_black_list.go` 是 JwtService 方法、与 utils 无关，保留）。

- [ ] **Step 5: 新建 api/v1/system/sys_user_refresh_test.go（refresh/logout cookie 行为）**

创建 `server/api/v1/system/sys_user_refresh_test.go`：
```go
package system

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/config"
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/middleware"
	"github.com/hllkk/devops-admin/server/model/system/request"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/songzhibin97/gkit/cache/local_cache"
	"go.uber.org/zap"
)

func init() {
	global.OPS_CONFIG = config.Server{JWT: config.JWT{SigningKey: "api-key", ExpiresTime: "1h", RefreshExTime: "168h", Issuer: "t"}}
	global.BlackCache = local_cache.NewCache(local_cache.SetDefaultExpire(3600 * 1e9))
	global.OPS_LOG = zap.NewNop()
}

func decodeCode(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("解码响应失败: %v body=%s", err, w.Body.String())
	}
	return body.Code
}

// RefreshToken 无 refresh cookie 必须返回 8888（不查库，纯 cookie 校验路径）。
func TestRefreshTokenMissingCookieReturns8888(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/auth/refreshToken", (&BaseApi{}).RefreshToken)

	req := httptest.NewRequest(http.MethodPost, "/auth/refreshToken", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if code := decodeCode(t, w); code != "8888" {
		t.Fatalf("无 refresh cookie 应返回 8888，got %s", code)
	}
}

// Logout 必须清除 cookie 且响应成功。
func TestLogoutClearsCookies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/auth/logout", (&BaseApi{}).Logout)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if code := decodeCode(t, w); code != "0000" {
		t.Fatalf("logout 应返回 0000，got %s", code)
	}
	setCookies := w.Result().Header["Set-Cookie"]
	if len(setCookies) < 2 {
		t.Fatalf("logout 应下发至少 2 个 Set-Cookie 清除 token/refresh-token，got %v", setCookies)
	}
}

// RefreshToken 带 access token（非 refresh）必须返回 8888。
func TestRefreshTokenWithAccessTokenRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	j := utils.NewJWT()
	access, _ := j.CreateAccessToken(request.BaseClaims{ID: 1, Username: "u"})

	r := gin.New()
	r.POST("/auth/refreshToken", (&BaseApi{}).RefreshToken)
	req := httptest.NewRequest(http.MethodPost, "/auth/refreshToken", nil)
	req.AddCookie(&http.Cookie{Name: "refresh-token", Value: access})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if code := decodeCode(t, w); code != "8888" {
		t.Fatalf("用 access 当 refresh 应返回 8888，got %s", code)
	}
	_ = middleware.JWTAuth // 保持 middleware import（防 lint 删 import）
}
```
> 若 Go 编译器因 `middleware` 未实际使用而报 imported-and-not-used，删除该 import 与对应 `_ = middleware.JWTAuth` 行即可。

- [ ] **Step 6: 跑测试与构建**

Run: `cd server && go test ./api/v1/system/ ./middleware/ ./utils/ -v`
Expected: 全 PASS（Login 端点依赖 DB，其端到端验证在 Task 8 手动进行）。

Run: `cd server && go build ./...`
Expected: 无报错。

- [ ] **Step 7: 提交**

```bash
cd /home/devops-admin
git add server/service/system/sys_user.go server/api/v1/system/sys_user.go server/api/v1/system/sys_user_refresh_test.go server/router/system/sys_base.go server/utils/jwt.go
git commit -m "feat(auth): Login 设 httpOnly cookie 返回 expiresAt，新增 /auth/refreshToken、/auth/logout，清理废弃 JWT 方法"
```

---

## Task 5: 前端请求层（withCredentials + cookie 刷新）

**Files:**
- Modify: `web/src/service/request/index.ts`
- Modify: `web/src/service/request/shared.ts`

**Interfaces:**
- Consumes: `fetchRefreshToken()`（Task 7 改为无参）、`localStg`、`useAuthStore`
- Produces: `request`（withCredentials=true）、`getAuthorization():null`、`handleExpiredRequest`、`scheduleProactiveRefresh/clearProactiveRefreshTimer`

- [ ] **Step 1: 改 service/request/shared.ts**

把 `web/src/service/request/shared.ts` 整文件替换为：
```typescript
import { useAuthStore } from '@/store/modules/auth';
import { getToken } from '@/store/modules/auth/shared';
import { localStg } from '@/utils/storage';
import { fetchRefreshToken } from '../api';
import type { RequestInstanceState } from './type';

// Cookie 鉴权模式：token 仅存 httpOnly cookie，前端不持有也不下发 Authorization。
export function getAuthorization(): null {
  return null;
}

let proactiveRefreshTimer: ReturnType<typeof setTimeout> | null = null;
let scheduledExpiresAt: number | null = null;

/** Schedule proactive token refresh at 80% of token lifetime. */
export function scheduleProactiveRefresh(expiresAtMs: number) {
  if (!expiresAtMs) return;
  if (scheduledExpiresAt === expiresAtMs && proactiveRefreshTimer) return;

  clearProactiveRefreshTimer();

  const now = Date.now();
  const tokenLifetime = expiresAtMs - now;
  if (tokenLifetime <= 60000) return;

  const delay = tokenLifetime * 0.8;
  if (delay <= 0) return;

  scheduledExpiresAt = expiresAtMs;
  proactiveRefreshTimer = setTimeout(async () => {
    proactiveRefreshTimer = null;
    scheduledExpiresAt = null;
    if (!getToken()) return;
    try {
      const { error, data } = await fetchRefreshToken();
      if (!error && data?.expiresAt) {
        localStg.set('tokenExpiresAt', data.expiresAt);
        scheduleProactiveRefresh(data.expiresAt);
      }
    } catch {
      // 静默失败：响应式刷新路径会兜底
    }
  }, delay);
}

export function clearProactiveRefreshTimer() {
  if (proactiveRefreshTimer) {
    clearTimeout(proactiveRefreshTimer);
    proactiveRefreshTimer = null;
  }
  scheduledExpiresAt = null;
}

/** refresh token —— refresh token 由浏览器经 httpOnly cookie 自动携带。 */
async function handleRefreshToken() {
  const { resetStore } = useAuthStore();
  const { data, error } = await fetchRefreshToken();
  if (!error && data) {
    if (data.expiresAt) {
      localStg.set('tokenExpiresAt', data.expiresAt);
      scheduleProactiveRefresh(data.expiresAt);
    }
    return true;
  }
  await resetStore('session_expired');
  return false;
}

export async function handleExpiredRequest(state: RequestInstanceState) {
  if (!state.refreshTokenPromise) {
    state.refreshTokenPromise = handleRefreshToken();
  }
  const success = await state.refreshTokenPromise;
  setTimeout(() => {
    state.refreshTokenPromise = null;
  }, 1000);
  return success;
}

export function showErrorMsg(state: RequestInstanceState, message: string) {
  if (!state.errMsgStack?.length) {
    state.errMsgStack = [];
  }
  const isExist = state.errMsgStack.includes(message);
  if (!isExist) {
    state.errMsgStack.push(message);
    window.$message?.error(message, {
      onLeave: () => {
        state.errMsgStack = state.errMsgStack.filter(msg => msg !== message);
        setTimeout(() => {
          state.errMsgStack = [];
        }, 5000);
      }
    });
  }
}
```

- [ ] **Step 2: 改 service/request/index.ts（withCredentials + 跳过 null 头）**

在 `web/src/service/request/index.ts` 中，把 `createFlatRequest` 的配置对象：
```typescript
export const request = createFlatRequest(
  {
    baseURL,
    headers: {
      apifoxToken: 'XL299LiMEDZ0H5h3A29PxwQXdMJqWyY2'
    }
  },
```
改为（加 `withCredentials: true`）：
```typescript
export const request = createFlatRequest(
  {
    baseURL,
    withCredentials: true,
    headers: {
      apifoxToken: 'XL299LiMEDZ0H5h3A29PxwQXdMJqWyY2'
    }
  },
```
把 `onRequest`：
```typescript
    async onRequest(config) {
      const Authorization = getAuthorization();
      Object.assign(config.headers, { Authorization });

      return config;
    },
```
改为（Authorization 为 null 时不设头）：
```typescript
    async onRequest(config) {
      const Authorization = getAuthorization();
      // Cookie 鉴权模式下 Authorization 为 null，不设置 Header（否则 axios 可能发送字面量 "null"）
      if (Authorization) {
        Object.assign(config.headers, { Authorization });
      }
      return config;
    },
```
把 `onBackendFail` 中刷新重试段：
```typescript
        const success = await handleExpiredRequest(request.state);
        if (success) {
          const Authorization = getAuthorization();
          Object.assign(response.config.headers, { Authorization });

          return instance.request(response.config) as Promise<AxiosResponse>;
        }
```
改为：
```typescript
        const success = await handleExpiredRequest(request.state);
        if (success) {
          const Authorization = getAuthorization();
          if (Authorization) {
            Object.assign(response.config.headers, { Authorization });
          }
          return instance.request(response.config) as Promise<AxiosResponse>;
        }
```

- [ ] **Step 3: 类型检查**

Run: `pnpm -C web typecheck`
Expected: 无报错（此时 `fetchRefreshToken` 仍为旧签名，Task 7 改；若报错指向 Task 7 尚未改的调用，记下并继续——本步允许在 Task 7 前存在跨文件类型波动，Task 7 完成后整体通过）。

> 若 typecheck 因 `fetchRefreshToken(refreshToken)` 旧签名与 shared.ts 新调用冲突而报错，属预期跨任务依赖；完成 Task 7 后再次 typecheck 必须通过。本步**暂不提交**，与 Task 6、7 合并验证后统一提交。

---

## Task 6: 前端 auth store 改 isAuthenticated 信号

**Files:**
- Modify: `web/src/store/modules/auth/shared.ts`
- Modify: `web/src/store/modules/auth/index.ts`

**Interfaces:**
- Consumes: `fetchLogin`（响应 `{expiresAt}`，Task 7）、`fetchGetUserInfo`、`fetchLogout`、`scheduleProactiveRefresh/clearProactiveRefreshTimer`、`localStg`
- Produces: `getToken()` 基于 `isAuthenticated`、`clearAuthStorage`、`login/loginByToken/getUserInfo/initUserInfo/resetStore`

- [ ] **Step 1: 改 store/modules/auth/shared.ts**

把 `web/src/store/modules/auth/shared.ts` 整文件替换为：
```typescript
import { localStg } from '@/utils/storage';

/** 登录态信号：httpOnly cookie 模式下 token 不进 JS，用 isAuthenticated 布尔判定是否已登录。 */
export function getToken(): string {
  const isAuthenticated = localStg.get('isAuthenticated');
  return isAuthenticated ? 'authenticated' : '';
}

/** 清除登录态本地信号（真正 token 由后端清 httpOnly cookie）。 */
export function clearAuthStorage() {
  localStg.remove('isAuthenticated');
  localStg.remove('tokenExpiresAt');
}
```

- [ ] **Step 2: 改 store/modules/auth/index.ts 的 import**

把 `web/src/store/modules/auth/index.ts` 顶部的 import 区：
```typescript
import { fetchGetUserInfo, fetchLogin, fetchLogout } from '@/service/api';
import { useRouterPush } from '@/hooks/common/router';
import { localStg } from '@/utils/storage';
import { SetupStoreId } from '@/enum';
import { $t } from '@/locales';
import { useRouteStore } from '../route';
import { useTabStore } from '../tab';
import { useNoticeStore } from '../notice';
import { clearAuthStorage, getToken } from './shared';
```
改为（加主动刷新定时器 import；移除不再需要的 useRoute/useTabStore/useNoticeStore 若下文未用——见 Step 3-5 决定）：
```typescript
import { fetchGetUserInfo, fetchLogin, fetchLogout } from '@/service/api';
import { useRouterPush } from '@/hooks/common/router';
import { localStg } from '@/utils/storage';
import { SetupStoreId } from '@/enum';
import { $t } from '@/locales';
import { useRouteStore } from '../route';
import { useTabStore } from '../tab';
import { useNoticeStore } from '../notice';
import { clearAuthStorage, getToken } from './shared';
import { clearProactiveRefreshTimer, scheduleProactiveRefresh } from '@/service/request/shared';
```

- [ ] **Step 3: 加 tokenExpiry 辅助 + 改 resetStore**

在 `const token = ref('');` 之前插入辅助函数：
```typescript
function storeTokenExpiry(expiresAt: number | undefined) {
  if (expiresAt) {
    localStg.set('tokenExpiresAt', expiresAt);
    scheduleProactiveRefresh(expiresAt);
  }
}
```
把 `resetStore` 函数整体替换为：
```typescript
  async function resetStore() {
    recordUserId();

    clearProactiveRefreshTimer();
    localStg.remove('tokenExpiresAt');

    try {
      await fetchLogout();
    } catch {
      // token 可能已失效，忽略登出接口错误
    }

    clearAuthStorage();
    authStore.$reset();

    if (!route.meta.constant) {
      await toLogin();
    }

    noticeStore.clearNotice();
    tabStore.cacheTabs();
    routeStore.resetStore();
  }
```

- [ ] **Step 4: 改 login / loginByToken**

把 `login` 函数整体替换为：
```typescript
  async function login(loginForm: Api.Auth.PwdLoginForm | Api.Auth.SocialLoginForm, redirect = true) {
    startLoading();

    // 仅取 username/password（社交登录为非目标，保留联合类型以免 social-callback 编译断裂）
    const { username, password } = loginForm as Api.Auth.PwdLoginForm;
    const { data, error } = await fetchLogin({ username, password });

    if (!error && data) {
      localStg.set('isAuthenticated', true);
      token.value = 'authenticated';
      storeTokenExpiry(data.expiresAt);

      const pass = await getUserInfo();
      if (pass) {
        const isClear = checkTabClear();
        let needRedirect = redirect;
        if (isClear) {
          needRedirect = false;
        }
        await redirectFromLogin(needRedirect);

        window.$notification?.success({
          title: $t('page.login.common.loginSuccess'),
          content: $t('page.login.common.welcomeBack', { userName: userInfo.user?.userName || '' }),
          duration: 4500
        });
      } else {
        resetStore();
      }
    } else {
      resetStore();
    }

    endLoading();
  }
```
删除原 `loginByToken` 函数（已被 login 内联取代）。

- [ ] **Step 5: 改 initUserInfo**

把 `initUserInfo` 函数整体替换为：
```typescript
  async function initUserInfo() {
    const maybeToken = getToken();

    if (maybeToken) {
      token.value = maybeToken;
      const pass = await getUserInfo();

      if (!pass) {
        resetStore();
      }
    }
  }
```

- [ ] **Step 6: 校验未使用的 import**

确认 `useTabStore`/`useNoticeStore`/`useRouteStore` 在 `resetStore` 中仍被使用（`tabStore.cacheTabs()`、`noticeStore.clearNotice()`、`routeStore.resetStore()`）——保留。`useRoute`/`route` 变量在 `resetStore` 中用 `route.meta.constant`——保留。无需删 import。

Run: `pnpm -C web typecheck`
Expected: 与 Task 5 同样，可能因 Task 7 的 `fetchLogin` 返回类型未改而波动；完成 Task 7 后必须通过。

---

## Task 7: 前端 auth api + 类型 + 登录页入参

**Files:**
- Modify: `web/src/service/api/auth.ts`
- Modify: `web/src/typings/api/auth.d.ts`
- Modify: `web/src/views/_builtin/login/modules/pwd-login.vue`

**Interfaces:**
- Consumes: `request`
- Produces: `fetchLogin(data: {username;password})`→`Api.Auth.LoginToken({expiresAt})`、`fetchRefreshToken()`→`Api.Auth.LoginToken`、`fetchLogout`、`fetchGetUserInfo`

- [ ] **Step 1: 改 typings/api/auth.d.ts**

把 `web/src/typings/api/auth.d.ts` 中：
```typescript
    interface LoginToken {
      token: string;
      refreshToken: string;
    }
```
替换为：
```typescript
    /** 登录/刷新响应：token 仅存 httpOnly cookie 不回传，expiresAt 为 access token 过期毫秒时间戳 */
    interface LoginToken {
      expiresAt: number;
    }
```
把 `PwdLoginForm`：
```typescript
    /** password login form */
    interface PwdLoginForm extends LoginForm {
      /** 用户名 */
      username?: string;
      /** 密码 */
      password?: string;
    }
```
替换为（cookie 模式仅用户名密码；client/grant/captcha 残留移除）：
```typescript
    /** password login form（httpOnly cookie 模式：仅用户名密码） */
    interface PwdLoginForm {
      /** 用户名 */
      username?: string;
      /** 密码 */
      password?: string;
    }
```

- [ ] **Step 2: 改 service/api/auth.ts**

把 `web/src/service/api/auth.ts` 的 `fetchLogin` 替换为：
```typescript
/**
 * Login（httpOnly cookie 模式：access/refresh 由后端写入 cookie，响应只回 expiresAt）
 *
 * @param data 用户名 + 密码
 */
export function fetchLogin(data: Api.Auth.PwdLoginForm) {
  return request<Api.Auth.LoginToken>({
    url: '/auth/login',
    method: 'post',
    data
  });
}
```
把 `fetchRefreshToken` 替换为（无 body，refresh token 经 cookie 携带）：
```typescript
/**
 * Refresh token —— refresh token 由浏览器经 httpOnly cookie 自动携带，无需传参。
 */
export function fetchRefreshToken() {
  return request<Api.Auth.LoginToken>({
    url: '/auth/refreshToken',
    method: 'post',
    data: {}
  });
}
```
保留 `fetchGetUserInfo`、`fetchLogout`。删除 `fetchRegister`、`fetchSocialLoginCallback`、`fetchCustomBackendError` 三个 Soybean 示例函数（后端无对应端点，避免误用）。删除后检查 `src/service/api/index.ts` 是否 re-export 了被删函数，若有则同步移除对应导出行。

Run: `cd /home/devops-admin && grep -rn "fetchRegister\|fetchSocialLoginCallback\|fetchCustomBackendError" web/src --include=*.ts --include=*.vue`
Expected: 仅命中 `auth.ts` 自身与可能的 `index.ts` re-export；逐一清理后再跑应仅剩（或无）`auth.ts` 内已删除前的引用=0。

- [ ] **Step 3: 改 pwd-login.vue 入参与残留 header**

`web/src/views/_builtin/login/modules/pwd-login.vue` 中 `model` 已是 `{username, password}`（带默认 admin/admin123），`authStore.login(model)` 调用不变（login 内部只取 username/password）。本步**无需改动 pwd-login.vue 逻辑**；仅确认 `model` 不再携带 `code/uuid/clientId/grantType`（当前不带，OK）。

> 若 typecheck 报 `Api.Auth.PwdLoginForm` 上访问 `code/uuid` 的错误：pwd-login.vue 中 `model.code`/`model.uuid` 仅在 `captchaEnabled` 分支使用，而 captcha 当前关闭（`/auth/code` 返回 `captchaEnabled=false`）。把这些访问改为可选链或保留 `code?: string; uuid?: string;` 在 `PwdLoginForm` 内即可。默认按"保留为可选字段不报错"处理：在 Step 1 的 `PwdLoginForm` 里**追加可选** `code?: string; uuid?: string;`（仅类型，不发后端）。如 typecheck 通过则不必追加。

- [ ] **Step 4: 前端整体类型检查 + 构建**

Run: `pnpm -C web typecheck`
Expected: 无报错。

Run: `pnpm -C web build`
Expected: 构建成功。

- [ ] **Step 5: 提交（Task 5/6/7 合并）**

```bash
cd /home/devops-admin
git add web/src/service/request/index.ts web/src/service/request/shared.ts web/src/store/modules/auth/shared.ts web/src/store/modules/auth/index.ts web/src/service/api/auth.ts web/src/service/api/index.ts web/src/typings/api/auth.d.ts web/src/views/_builtin/login/modules/pwd-login.vue
git commit -m "feat(web): 改 httpOnly cookie 鉴权——withCredentials、getAuthorization 返回 null、isAuthenticated 信号、cookie 主动/响应式刷新、登录入参收敛"
```

---

## Task 8: 文档契约 + 全量验证（端到端）

**Files:**
- Modify: `aiDoc/frontend-backend/boundary.md`

- [ ] **Step 1: boundary.md 补 cookie 认证契约**

在 `aiDoc/frontend-backend/boundary.md` 的「## 初始化向导（/init/*）」段之后，新增一节：
```markdown
## 认证（httpOnly cookie）

- 认证载体：access + refresh 双 **httpOnly cookie**，token 不进 JS。
  - cookie 名：`token`（access）、`refresh-token`（refresh）
  - 属性：`HttpOnly=true`、`SameSite=Strict`、`Secure=RequestIsSecure(X-Forwarded-Proto→TLS)`、`Path=/`、`Domain=""`
- 取 token：后端 `utils.GetToken` 优先 `Authorization: Bearer` 头，其次 `token` cookie；**禁止从 query 取 token**。
- 鉴权失败契约：统一 **HTTP 200 + 业务 code**（不再 401）：
  - `9999`（`VITE_SERVICE_EXPIRED_TOKEN_CODES`）：access 失效，前端调 `/auth/refreshToken` 刷新并重试
  - `8888`（`VITE_SERVICE_LOGOUT_CODES`）：refresh 也失效（仅 `/auth/refreshToken` 返回），前端登出
  - `/auth/refreshToken` **禁止返回 9999**，防前端死循环刷新
- 登录响应体只回 `{ expiresAt }`（毫秒），**不回 token**。
- 端点：`POST /auth/login`、`POST /auth/refreshToken`、`POST /auth/logout`（public）；`GET /auth/getUserInfo`（private）。
- 前端：所有请求 `withCredentials: true`；`getAuthorization()` 返回 `null`；登录态信号 `isAuthenticated`+`tokenExpiresAt`（localStorage，可读非敏感）。
- CORS：`middleware.Cors()` 回显 Origin + `Allow-Credentials: true`。
```

- [ ] **Step 2: 后端全量构建 + 测试**

Run: `cd server && go build ./... && go test ./...`
Expected: 构建无错；测试全 PASS（Login 端点的 DB 依赖用例若无测试 DB 会跳过/不涉及，本次未写 DB 用例）。

- [ ] **Step 3: 前端全量构建**

Run: `pnpm -C web build`
Expected: 构建成功。

- [ ] **Step 4: 端到端手动验证（启动后端 + 前端）**

> 需要已初始化的 DB 与种子超管账号（admin/admin123，见权限基座闭环 spec）。在两个终端：
1. 后端：`cd server && go run .`（监听 :8888）
2. 前端：`pnpm -C web dev`（dev proxy 指向 :8888）

用 curl 验证后端（注意 `--cookie`/`-c`/`-b`）：
```bash
# 登录：应返回 {expiresAt} 且 Set-Cookie 含 token/refresh-token（HttpOnly）
curl -i -s -c /tmp/cj.txt -X POST http://localhost:8888/auth/login \
  -H 'Content-Type: application/json' -d '{"username":"admin","password":"admin123"}'

# 带 cookie 访问需鉴权接口：应 200 {code:0000}
curl -s -b /tmp/cj.txt http://localhost:8888/auth/getUserInfo

# 不带 cookie 访问：应返回 code 9999
curl -s http://localhost:8888/auth/getUserInfo

# 刷新：用 cookie 换新 token，应 200 {expiresAt}
curl -i -s -b /tmp/cj.txt -c /tmp/cj.txt -X POST http://localhost:8888/auth/refreshToken

# 登出：应清 cookie + 200
curl -i -s -b /tmp/cj.txt -X POST http://localhost:8888/auth/logout
```
Expected：
- 登录响应体**无** token 字段；`Set-Cookie` 含 `token=...; HttpOnly; SameSite=Strict` 与 `refresh-token=...; HttpOnly`
- 带 cookie 的 getUserInfo 返回 `code:"0000"`
- 不带 cookie 返回 `code:"9999"`
- refreshToken 成功返回 `{expiresAt}` 并下发新 cookie
- logout 下发清除 cookie 的 `Set-Cookie`

浏览器验证（前端 dev）：
- 登录后刷新页面仍保持登录态
- 等待/触发 access 过期后业务请求自动刷新重试（可临时把 `expires-time` 调短如 `90s` 验证）
- 登出后跳登录页且 cookie 被清

- [ ] **Step 5: 提交文档**

```bash
cd /home/devops-admin
git add aiDoc/frontend-backend/boundary.md
git commit -m "docs(boundary): 补 httpOnly cookie 认证契约（双 cookie/失败 code/端点/CORS）"
```

---

## Self-Review（写计划后自检）

**Spec 覆盖**：
- §4.1 双 token/claims → Task 1 ✓
- §4.2 取 token/cookie 工具 → Task 2 ✓
- §4.3 NoAuthWithCode 契约 → Task 3 ✓
- §4.4 中间件 + CORS → Task 3 ✓
- §4.5 Login/Refresh/Logout + 路由 → Task 4 ✓
- §4.6 删除残留（SetToken/ClearToken/new-token 头/BufferTime 逻辑/CreateTokenByOldToken/SetRedisJWT/CreateClaims）→ Task 2/3/4 ✓
- §5 前端请求层/store/api/类型/登录页 → Task 5/6/7 ✓
- §9 boundary.md → Task 8 ✓
- §8 测试 → Task 1/2/3/4 单测 + Task 8 端到端 ✓

**占位符**：无 TBD/TODO；每步均含完整代码或精确命令。

**类型一致**：
- `LoginToken` 全链路统一为 `{ expiresAt: number }`（d.ts、api、store）✓
- `userService.Login` 返回 `(accessToken, refreshToken, user, err)` 与 API 层调用一致 ✓
- `getAuthorization(): null`、`getToken(): string`（'authenticated'|''）一致 ✓
- `SetLoginCookies(c, access, refresh)` / `ClearLoginCookies(c)` 签名前后一致 ✓
- 失败 code 常量 `expiredTokenCode="9999"`（middleware）、`logoutTokenCode="8888"`（api）与前端 `.env` 一致 ✓
