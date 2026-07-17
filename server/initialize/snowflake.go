package initialize

import (
	"github.com/hllkk/devops-admin/server/global"
	"go.uber.org/zap"
)

// InitSnowflake 按当前配置初始化 yitter IdGenerator。
// workerID 来自 config.System.WorkerID(0-63);多副本部署须保证各实例唯一。
func InitSnowflake() {
	if err := global.InitSnowflake(global.OPS_CONFIG.System.WorkerID); err != nil && global.OPS_LOG != nil {
		global.OPS_LOG.Error("snowflake init failed", zap.Error(err))
	}
}

// RegisterSnowflakeCallbacks 为主库(OPS_DB)及所有多库连接(OPS_DBList)注册雪花 ID 回调。
// 需在 DB 初始化完成后、RegisterTables 之前调用,与 RegisterDataScopeCallbacks 同位置。
func RegisterSnowflakeCallbacks() {
	if global.OPS_DB != nil {
		if err := global.RegisterSnowflakeCallbacks(global.OPS_DB); err != nil && global.OPS_LOG != nil {
			global.OPS_LOG.Error("register snowflake callback on OPS_DB failed", zap.Error(err))
		}
	}
	for name, db := range global.OPS_DBList {
		if err := global.RegisterSnowflakeCallbacks(db); err != nil && global.OPS_LOG != nil {
			global.OPS_LOG.Error("register snowflake callback on dblist failed", zap.String("db", name), zap.Error(err))
		}
	}
}
