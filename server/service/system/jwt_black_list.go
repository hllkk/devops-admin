package system

import (
	"context"
	"time"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

type JwtService struct{}

var JwtServiceApp = new(JwtService)

// JsonInBlacklist 将 token 入黑名单(DB + Redis)。
//   - DB:create_time 由 gorm 维护,ClearDB 定时任务(@daily)按 create_time 7 天物理清理,
//     与 token 最长生命周期(默认 7d)对齐,保证表有界。
//   - Redis:按 token 剩余有效期带 TTL 写入,token 过期后缓存自动回收,避免只增不减。
//   - token 无效或已过期则不拉黑(本就已失效,无需再占黑名单席位)。
func (jwtService *JwtService) JsonInBlacklist(ctx context.Context, jwtList system.JwtBlacklist) (err error) {
	ttl, ok := blacklistTTL(jwtList.Jwt)
	if !ok {
		return nil
	}
	if err = global.OPS_DB.WithContext(ctx).Create(&jwtList).Error; err != nil {
		return
	}
	global.OPS_CACHE.Set(jwtList.Jwt, "1", ttl)
	return
}

// GetRedisJWT 多端登录模式下取用户当前活跃 jwt。
func (jwtService *JwtService) GetRedisJWT(ctx context.Context, userName string) (redisJWT string, err error) {
	redisJWT, err = global.OPS_REDIS.Get(ctx, userName).Result()
	return redisJWT, err
}

// LoadAll 启动时把 DB 中的黑名单加载进 OPS_CACHE。
// 仅加载尚未过期的 token,并按剩余有效期带 TTL 写入,与 token 生命周期对齐;
// 已过期(尚未被 ClearDB 清走的)跳过,顺带避免无谓占内存。
func LoadAll(ctx context.Context) {
	var data []string
	err := global.OPS_DB.WithContext(ctx).Model(&system.JwtBlacklist{}).Select("jwt").Find(&data).Error
	if err != nil {
		logger.WithCtx(ctx).Mod("biz").Err(err).Error("加载数据库jwt黑名单失败!")
		return
	}
	for i := 0; i < len(data); i++ {
		if ttl, ok := blacklistTTL(data[i]); ok {
			global.OPS_CACHE.Set(data[i], "1", ttl)
		}
	}
}

// blacklistTTL 解析 token 的剩余有效期。ok=false 表示 token 无效或已过期,无需入黑名单。
func blacklistTTL(token string) (time.Duration, bool) {
	claims, err := utils.NewJWT().ParseToken(token)
	if err != nil {
		return 0, false
	}
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		return 0, false
	}
	return ttl, true
}
