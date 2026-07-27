package middleware

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/common/response"
	"github.com/hllkk/devops-admin/server/service"
	"github.com/hllkk/devops-admin/server/utils/logger"
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
	// 无缓存句柄（极端启动期）时 fail-open，避免误伤
	if global.OPS_CACHE == nil {
		return nil
	}
	if err = SetLimitWithTime(key, limit, time.Duration(expire)*time.Second); err != nil {
		logger.Bg().Mod("system").Err(err).Error("limit")
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

// SetLimitWithTime 设置访问次数：窗口内计数到达 limit 即拒绝。
func SetLimitWithTime(key string, limit int, expiration time.Duration) error {
	count, err := global.OPS_CACHE.IncrementWithExpire(key, 1, expiration)
	if err != nil {
		// 运行时缓存异常：记录日志并 fail-open 放行
		logger.Bg().Mod("system").Err(err).Error("limit increment")
		return nil
	}
	if count > int64(limit) {
		return errors.New("请求太过频繁，请稍后再试")
	}
	return nil
}

// CacheCheckOrMark 基于 GVA_CACHE 的限流计数 超限返回错误 cache 异常 fail-open
func CacheCheckOrMark(key string, expire int, limit int) error {
	if global.OPS_CACHE == nil {
		return nil
	}
	n, err := global.OPS_CACHE.IncrementWithExpire(key, 1, time.Duration(expire)*time.Second)
	if err != nil {
		logger.Bg().Mod("system").Err(err).Error("limit")
		return nil // fail-open
	}
	if int(n) > limit {
		return errors.New("请求太过频繁，请稍后再试")
	}
	return nil
}

// SecurityLimit 按安全配置对登录/敏感接口做 IP 黑白名单校验 + 限流。
//   - IP 黑白名单(IpValidationEnabled 为真即生效,独立于限流开关):blacklist 命中拒绝、whitelist 未命中拒绝
//   - 限流(LimitEnable 为真即生效):按 IP+路由 在窗口内计数,超限拒绝;未开启或缓存异常 fail-open 放行
func SecurityLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg := service.ServiceGroupApp.SystemServiceGroup.SecurityConfigService.Current(c.Request.Context())
		if cfg.IpValidationEnabled {
			if ok, msg := ipAllowed(cfg.IpValidationMode, cfg.IpBlacklist, cfg.IpWhitelist, c.ClientIP()); !ok {
				c.JSON(http.StatusOK, gin.H{"code": response.ERROR, "msg": msg})
				c.Abort()
				return
			}
		}
		if !cfg.LimitEnable {
			c.Next()
			return
		}
		key := "OPS_SecLimit" + c.ClientIP() + c.FullPath()
		if err := CacheCheckOrMark(key, cfg.LimitWindow, cfg.LimitCount); err != nil {
			c.JSON(http.StatusOK, gin.H{"code": response.ERROR, "msg": err.Error()})
			c.Abort()
			return
		}
		c.Next()
	}
}

// ipAllowed 按 IpValidationMode 判定 ip 是否放行。
//   - whitelist:ip 不在白名单则拒绝
//   - blacklist(默认/空):ip 命中黑名单则拒绝
//
// 条目支持单 IP 或 CIDR(如 10.0.0.0/8),逗号/换行分隔,空白条目忽略。
func ipAllowed(mode, blacklist, whitelist, ipStr string) (bool, string) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false, "无效的客户端 IP"
	}
	if mode == "whitelist" {
		if !matchIPEntry(ip, splitIPEntries(whitelist)) {
			return false, "当前 IP 不在访问白名单"
		}
		return true, ""
	}
	if matchIPEntry(ip, splitIPEntries(blacklist)) {
		return false, "当前 IP 已被加入黑名单"
	}
	return true, ""
}

func splitIPEntries(raw string) []string {
	out := make([]string, 0, 8)
	for _, p := range strings.Split(strings.ReplaceAll(raw, "\n", ","), ",") {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func matchIPEntry(ip net.IP, entries []string) bool {
	for _, e := range entries {
		if strings.Contains(e, "/") {
			if _, cidr, err := net.ParseCIDR(e); err == nil && cidr.Contains(ip) {
				return true
			}
			continue
		}
		if eip := net.ParseIP(e); eip != nil && eip.Equal(ip) {
			return true
		}
	}
	return false
}
