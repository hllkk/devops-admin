package system

import (
	"context"
	"sync"
	"time"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/utils"
	"go.uber.org/zap"
)

type JwtService struct{}

var JwtServiceApp = new(JwtService)

// JsonInBlacklist 加入黑名单（兼容旧调用，委托到 utils.NewJWT().JoinBlacklist）
func (jwtService *JwtService) JsonInBlacklist(jwtList system.JwtBlacklist) (err error) {
	return utils.NewJWT().JoinBlacklist(jwtList.Jwt)
}

// GetRedisJWT 从redis取jwt
func (jwtService *JwtService) GetRedisJWT(userName string) (redisJWT string, err error) {
	redisJWT, err = global.OPS_REDIS.Get(context.Background(), userName).Result()
	return redisJWT, err
}

// blacklistCleanInterval 黑名单过期记录清理周期
const blacklistCleanInterval = time.Hour

// cleanerStartOnce 保证清理 goroutine 只启动一次（reload 安全）
var cleanerStartOnce sync.Once

// CleanExpiredBlacklist 物理删除已过期的 JWT 黑名单 DB 记录。
// Redis 黑名单自带 TTL 自动过期，此处只清 DB 兜底记录。
func CleanExpiredBlacklist() error {
	if global.OPS_DB == nil {
		return nil
	}
	return global.OPS_DB.Unscoped().
		Where("expires_at < ? AND expires_at > ?", time.Now(), time.Unix(0, 0)).
		Delete(&system.JwtBlacklist{}).Error
}

// StartBlacklistCleaner 启动 JWT 黑名单过期记录定时清理（sync.Once 幂等，reload 安全）。
// 必须在 DB 初始化完成、表建好之后调用。
func StartBlacklistCleaner() {
	cleanerStartOnce.Do(func() {
		go runBlacklistCleaner()
		global.OPS_LOG.Info("JWT 黑名单清理任务已启动", zap.Duration("interval", blacklistCleanInterval))
	})
}

// runBlacklistCleaner 周期性清理过期黑名单记录
func runBlacklistCleaner() {
	ticker := time.NewTicker(blacklistCleanInterval)
	defer ticker.Stop()
	for range ticker.C {
		if global.OPS_DB == nil {
			continue
		}
		if err := CleanExpiredBlacklist(); err != nil {
			global.OPS_LOG.Warn("清理过期 JWT 黑名单失败", zap.Error(err))
		}
	}
}

func LoadAll() {
	// Redis+DB 双写模式下，Redis 是黑名单主存储且自带 TTL 过期；
	// 本地缓存 BlackCache 仅作进程级兜底，启动时不再从 DB 全量加载。
	// DB 记录仅供 Redis miss 时的回退查询，由 StartBlacklistCleaner 定时清理过期数据。
	global.OPS_LOG.Info("JWT 黑名单使用 Redis+DB 双写模式，本地缓存仅作兜底")
}
