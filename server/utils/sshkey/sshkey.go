package sshkey

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/crypto/ssh"
)

// 平台 SSH 密钥对管理：启动期自动生成 ed25519 密钥对（幂等），安装 agent 时
// 公钥部署到目标机 authorized_keys，私钥供平台后续 SSH 访问（P4 用户公钥下发
// 复用同套密钥读写，写任意公钥的能力在 service/server/agent_install.go）。
// 密钥落文件系统（部署持久化由数据卷保证，与 uploads 同理），不落 DB。

const (
	privKeyName = "aiops_platform_ed25519"
	pubKeyName  = privKeyName + ".pub"
)

var (
	mu    sync.Mutex
	cache struct {
		dir     string
		privPEM []byte
		pubKey  string
	}
)

// Ensure 生成或加载平台密钥对（幂等；并发安全；目录不存在则创建）。
// 返回私钥 PEM 与公钥 authorized_keys 单行格式。
func Ensure(dir string) (privPEM []byte, pubKey string, err error) {
	mu.Lock()
	defer mu.Unlock()
	if dir == "" {
		dir = "resource/ssh"
	}
	if cache.dir == dir && len(cache.privPEM) > 0 && cache.pubKey != "" {
		return cache.privPEM, cache.pubKey, nil
	}
	privPath := filepath.Join(dir, privKeyName)
	pubPath := filepath.Join(dir, pubKeyName)

	if _, err := os.Stat(privPath); err == nil {
		privPEM, err = os.ReadFile(privPath)
		if err != nil {
			return nil, "", fmt.Errorf("读平台私钥失败: %w", err)
		}
		pubBytes, err := os.ReadFile(pubPath)
		if err != nil || len(pubBytes) == 0 {
			// 私钥在公钥丢：从私钥重导公钥
			derived, err := derivePub(privPEM)
			if err != nil {
				return nil, "", err
			}
			pubBytes = []byte(derived)
			if werr := os.WriteFile(pubPath, pubBytes, 0o644); werr != nil {
				return nil, "", fmt.Errorf("回写平台公钥失败: %w", werr)
			}
		}
		cache.dir, cache.privPEM, cache.pubKey = dir, privPEM, string(pubBytes)
		return cache.privPEM, cache.pubKey, nil
	}

	// 不存在则生成
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, "", fmt.Errorf("生成 ed25519 密钥对失败: %w", err)
	}
	privPEM, err = marshalPriv(priv)
	if err != nil {
		return nil, "", err
	}
	pub, err := ssh.NewPublicKey(priv.Public())
	if err != nil {
		return nil, "", fmt.Errorf("派生平台公钥失败: %w", err)
	}
	pubKey = string(ssh.MarshalAuthorizedKey(pub))

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, "", fmt.Errorf("创建密钥目录失败: %w", err)
	}
	if err := os.WriteFile(privPath, privPEM, 0o600); err != nil {
		return nil, "", fmt.Errorf("写平台私钥失败: %w", err)
	}
	if err := os.WriteFile(pubPath, []byte(pubKey), 0o644); err != nil {
		return nil, "", fmt.Errorf("写平台公钥失败: %w", err)
	}
	cache.dir, cache.privPEM, cache.pubKey = dir, privPEM, pubKey
	return privPEM, pubKey, nil
}

// PublicKey 获取平台公钥（authorized_keys 单行格式）。
func PublicKey(dir string) (string, error) {
	_, pub, err := Ensure(dir)
	return pub, err
}

// Signer 获取平台私钥签名器（SSH 客户端认证用）。
func Signer(dir string) (ssh.Signer, error) {
	privPEM, _, err := Ensure(dir)
	if err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKey(privPEM)
}

// derivePub 从私钥 PEM 重导公钥行。
func derivePub(privPEM []byte) (string, error) {
	signer, err := ssh.ParsePrivateKey(privPEM)
	if err != nil {
		return "", fmt.Errorf("解析平台私钥失败: %w", err)
	}
	return string(ssh.MarshalAuthorizedKey(signer.PublicKey())), nil
}

// marshalPriv ed25519 私钥转 OpenSSH PEM（无口令）。
func marshalPriv(priv ed25519.PrivateKey) ([]byte, error) {
	block, err := ssh.MarshalPrivateKey(priv, "aiops-platform")
	if err != nil {
		return nil, fmt.Errorf("序列化平台私钥失败: %w", err)
	}
	return pem.EncodeToMemory(block), nil
}
