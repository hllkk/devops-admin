package initialize

import (
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// Reload 优雅地重新加载系统配置
func Reload() error {
	logger.Bg().Mod("system").Info("正在重新加载系统配置...")

	// 重新加载配置文件
	if err := global.OPS_VP.ReadInConfig(); err != nil {
		logger.Bg().Mod("system").Err(err).Error("重新读取配置文件失败!")
		return err
	}
	// ReadInConfig 后必须 Unmarshal 刷新内存(修复历史 bug:只读不刷),再应用 env 覆盖与安全校验
	if err := global.OPS_VP.Unmarshal(&global.OPS_CONFIG); err != nil {
		logger.Bg().Mod("system").Err(err).Error("重新 Unmarshal 配置失败!")
		return err
	}
	global.ApplySensitiveEnvAndValidate()

	// 重新初始化数据库连接
	if global.OPS_DB != nil {
		db, _ := global.OPS_DB.DB()
		err := db.Close()
		if err != nil {
			logger.Bg().Mod("system").Err(err).Error("关闭原数据库连接失败!")
			return err
		}
	}

	// 重新建立数据库连接
	global.OPS_DB = Gorm()

	// 重新初始化其他配置
	OtherInit()
	DBList()

	if global.OPS_DB != nil {
		// 重新初始化雪花算法并注册 GORM 回调
		InitSnowflake()
		// 重新注册数据权限 GORM 回调
		RegisterDataScopeCallbacks()
		// 注册雪花 ID GORM 回调(须在建表前)
		RegisterSnowflakeCallbacks()
		// 确保数据库表结构是最新的
		RegisterTables()
	}

	// 重新初始化定时任务
	Timer()
	LoadTimedTasks()

	logger.Bg().Mod("system").Info("系统配置重新加载完成")
	return nil
}
