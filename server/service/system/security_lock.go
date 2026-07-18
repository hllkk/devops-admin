package system

import (
	"context"
	"time"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
)

// 登录失败计数 / 账号锁定(对齐 GVA service/system/security_lock.go)
const (
	loginFailKeyPrefix = "login_fail:"
	loginLockKeyPrefix = "login_lock:"
)

func loginFailKey(username string) string { return loginFailKeyPrefix + username }
func loginLockKey(username string) string { return loginLockKeyPrefix + username }

// IsAccountLocked 账号是否处于锁定状态
func IsAccountLocked(_ context.Context, username string) bool {
	if username == "" {
		return false
	}
	return global.OPS_CACHE.Exists(loginLockKey(username))
}

// RecordLoginFail 记录一次登录失败 计数滚动窗口 TTL=锁定时长 达阈值置锁。
// 当前 model 无独立 LockEnable 开关(GVA 有),以 LoginFailLockCount>0 视为开启。
func RecordLoginFail(_ context.Context, username string, cfg system.SysSecurityConfig) {
	if username == "" || cfg.LoginFailLockCount <= 0 {
		return
	}
	n, err := global.OPS_CACHE.IncrementWithExpire(loginFailKey(username), 1, cfg.LockDurationTimeout())
	if err != nil {
		return
	}
	if int(n) >= cfg.LoginFailLockCount {
		global.OPS_CACHE.Set(loginLockKey(username), 1, cfg.LockDurationTimeout())
	}
}

// ClearLoginFail 清除失败计数与锁 登录成功调用
func ClearLoginFail(_ context.Context, username string) {
	if username == "" {
		return
	}
	global.OPS_CACHE.Delete(loginFailKey(username))
	global.OPS_CACHE.Delete(loginLockKey(username))
}

// IsPasswordExpired 密码是否过期(对齐 GVA:nil 视为不过期——未知最后修改时间不强制改密)
func IsPasswordExpired(_ context.Context, passwordUpdatedAt *time.Time, cfg system.SysSecurityConfig, now time.Time) bool {
	if !cfg.PwdExpireEnable || cfg.PwdExpireDays <= 0 {
		return false
	}
	if passwordUpdatedAt == nil {
		return false
	}
	deadline := passwordUpdatedAt.AddDate(0, 0, cfg.PwdExpireDays)
	return now.After(deadline)
}
