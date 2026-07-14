package initialize

import (
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/service/system"
	"github.com/hllkk/devops-admin/server/utils"
	"go.uber.org/zap"
)

// 初始化全局函数
func SetupHandlers() {
	// 注册系统重载处理函数
	utils.GlobalSystemEvents.RegisterReloadHandler(func() error {
		return Reload()
	})
	// 注入首次初始化数据库路径（/init/initdb）的回调：
	//  1. OPS_DB 就绪后注册雪花回调，确保首初始化创建的表也能生成雪花主键；
	//  2. 向导已写入 OPS_CONFIG.Redis + System.UseRedis，此处即时连接 OPS_REDIS
	//     （ping 已在向导测过；失败仅告警不 panic，重启后由 RunServer 兜底）。
	system.SetDBReadyCallback(func() {
		RegisterCallbacks(global.OPS_DB)
		if global.OPS_CONFIG.System.UseRedis && global.OPS_REDIS == nil {
			if client, err := DialRedis(global.OPS_CONFIG.Redis); err != nil {
				global.OPS_LOG.Warn("init 后即时连接 Redis 失败，重启后将自动重试", zap.Error(err))
			} else {
				global.OPS_REDIS = client
			}
		}
	})
}
