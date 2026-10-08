package server

import (
	"crypto/sha256"
	"encoding/hex"
)

// sha256Hex 明文 → sha256 hex（安装任务ID生成用）。
func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
