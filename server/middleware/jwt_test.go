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
