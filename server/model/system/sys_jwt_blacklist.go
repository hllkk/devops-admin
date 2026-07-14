package system

import (
	"time"

	"github.com/hllkk/devops-admin/server/global"
)

// JwtBlacklist JWT 黑名单记录（Redis 优先查询，DB 兜底审计）
// Redis 使用 blacklist:<token> 键（带 TTL 自动过期），DB 记录用于 Redis miss 时回退查询和审计。
type JwtBlacklist struct {
	ID        int64           `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	global.OPS_MODEL
	Jwt       string          `gorm:"type:text;comment:jwt token 原文" json:"jwt"`
	JwtHash   string          `gorm:"type:varchar(64);uniqueIndex;comment:jwt token SHA256 摘要（加速 DB 回退查询）" json:"jwtHash"`
	ExpiresAt time.Time       `gorm:"index;comment:token 过期时间（供定时清理按过期时间删除）" json:"expiresAt"`
}
