package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hllkk/devops-admin/server/config"
	"github.com/hllkk/devops-admin/server/global"
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
	// 保存并恢复全局配置，避免污染同包其他测试
	orig := global.OPS_CONFIG
	defer func() { global.OPS_CONFIG = orig }()

	// 未配置可信代理：不信任 X-Forwarded-Proto，回退 c.Request.TLS（此处无 TLS → false）
	global.OPS_CONFIG = config.Server{}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	if RequestIsSecure(newContext(req)) {
		t.Fatal("未配置可信代理时不应信任 X-Forwarded-Proto")
	}

	// 配置可信代理：信任 X-Forwarded-Proto
	global.OPS_CONFIG.System.TrustedProxies = []string{"127.0.0.1"}
	if !RequestIsSecure(newContext(req)) {
		t.Fatal("配置可信代理后 X-Forwarded-Proto=https 应判定 secure")
	}
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("X-Forwarded-Proto", "http")
	if RequestIsSecure(newContext(req2)) {
		t.Fatal("配置可信代理时 X-Forwarded-Proto=http 应判定非 secure")
	}
}
