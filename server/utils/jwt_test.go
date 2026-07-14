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
