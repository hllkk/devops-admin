package middleware

import (
	"context"
	"errors"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/common/response"
)

type LimitConfig struct {
	// GenerationKey 根据业务生成key 下面CheckOrMark查询生成
	GenerationKey func(c *gin.Context) string
	// 检查函数,用户可修改具体逻辑,更加灵活
	CheckOrMark func(key string, expire int, limit int) error
	// Expire key 过期时间
	Expire int
	// Limit 周期时间
	Limit int
}

func (l LimitConfig) LimitWithTime() gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := l.CheckOrMark(l.GenerationKey(c), l.Expire, l.Limit); err != nil {
			c.JSON(http.StatusOK, gin.H{"code": response.ERROR, "msg": err.Error()})
			c.Abort()
			return
		} else {
			c.Next()
		}
	}
}

// DefaultGenerationKey 默认生成key
func DefaultGenerationKey(c *gin.Context) string {
	return "GVA_Limit" + c.ClientIP()
}

func DefaultCheckOrMark(key string, expire int, limit int) (err error) {
	// 判断是否开启redis
	if global.OPS_REDIS == nil {
		return err
	}
	if err = SetLimitWithTime(key, limit, time.Duration(expire)*time.Second); err != nil {
		global.OPS_LOG.Error("limit", zap.Error(err))
	}
	return err
}

func DefaultLimit() gin.HandlerFunc {
	return LimitConfig{
		GenerationKey: DefaultGenerationKey,
		CheckOrMark:   DefaultCheckOrMark,
		Expire:        global.OPS_CONFIG.System.LimitTimeIP,
		Limit:         global.OPS_CONFIG.System.LimitCountIP,
	}.LimitWithTime()
}

// LoginLimit 登录端点 IP 限流：按 ClientIP 限制时间窗口内的登录尝试次数，
// 超限返回失败（code "0001"）；Redis 不可用或命令异常时降级放行，避免限流组件故障锁死登录。
// 阈值取 system.login-limit-count / login-limit-window（<=0 走默认 10 次 / 60 秒）。
// 与验证码触发互补：验证码挡“需要人机校验的重复尝试”，IP 限流挡“无视验证码的高频请求/资源消耗”。
func LoginLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		expire := global.OPS_CONFIG.System.LoginLimitWindow
		if expire <= 0 {
			expire = 60
		}
		limit := global.OPS_CONFIG.System.LoginLimitCount
		if limit <= 0 {
			limit = 10
		}
		if global.OPS_REDIS == nil {
			c.Next()
			return
		}
		ctx := context.Background()
		key := "login_limit:" + c.ClientIP()
		// INCR 原子自增；仅首次（结果为 1）时设置过期，构成固定时间窗口
		times, err := global.OPS_REDIS.Incr(ctx, key).Result()
		if err != nil {
			// Redis 命令异常：降级放行，不因限流组件故障锁死登录
			c.Next()
			return
		}
		if times == 1 {
			global.OPS_REDIS.Expire(ctx, key, time.Duration(expire)*time.Second)
		}
		if times > int64(limit) {
			response.FailWithMessage("登录尝试过于频繁，请稍后再试", c)
			c.Abort()
			return
		}
		c.Next()
	}
}

// SetLimitWithTime 设置访问次数
func SetLimitWithTime(key string, limit int, expiration time.Duration) error {
	count, err := global.OPS_REDIS.Exists(context.Background(), key).Result()
	if err != nil {
		return err
	}
	if count == 0 {
		pipe := global.OPS_REDIS.TxPipeline()
		pipe.Incr(context.Background(), key)
		pipe.Expire(context.Background(), key, expiration)
		_, err = pipe.Exec(context.Background())
		return err
	} else {
		// 次数
		if times, err := global.OPS_REDIS.Get(context.Background(), key).Int(); err != nil {
			return err
		} else {
			if times >= limit {
				if t, err := global.OPS_REDIS.PTTL(context.Background(), key).Result(); err != nil {
					return errors.New("请求太过频繁，请稍后再试")
				} else {
					return errors.New("请求太过频繁, 请 " + t.String() + " 秒后尝试")
				}
			} else {
				return global.OPS_REDIS.Incr(context.Background(), key).Err()
			}
		}
	}
}
