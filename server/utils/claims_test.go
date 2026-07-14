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
