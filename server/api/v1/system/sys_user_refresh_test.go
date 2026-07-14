package system

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
}
