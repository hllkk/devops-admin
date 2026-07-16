package initialize

import (
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/service/system"
	"github.com/hllkk/devops-admin/server/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 初始化全局函数
func SetupHandlers() {
	// 注册系统重载处理函数
	utils.GlobalSystemEvents.RegisterReloadHandler(func() error {
		return Reload()
	})
	// 注入首次初始化数据库路径（/init/initdb）的回调：
	//  1. db 创建后、建表前注册雪花回调（SetDBCallbacksCallback），确保首初始化阶段
	//     依赖雪花主键的 seed（如 sys_setting）能正确生成主键——InitData 在 dbReadyCallback
	//     之前执行，回调必须更早注册；
	//  2. 向导已写入 OPS_CONFIG.Redis + System.UseRedis，流程末尾即时连接 OPS_REDIS
	//     （ping 已在向导测过；失败仅告警不 panic，重启后由 RunServer 兜底）。
	system.SetDBCallbacksCallback(func(db *gorm.DB) {
		RegisterCallbacks(db)
	})
	system.SetDBReadyCallback(func() {
		if global.OPS_CONFIG.System.UseRedis && global.OPS_REDIS == nil {
			if client, err := DialRedis(global.OPS_CONFIG.Redis); err != nil {
				global.OPS_LOG.Warn("init 后即时连接 Redis 失败，重启后将自动重试", zap.Error(err))
			} else {
				global.OPS_REDIS = client
			}
		}
	})
}
