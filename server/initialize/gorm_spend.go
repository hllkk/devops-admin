package initialize

import (
	"fmt"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/utils/logger"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// spendIndexes 启动时幂等确保的表达式索引，解决 COALESCE 导致 ORDER BY / 游标全表扫描问题。
var spendIndexes = []string{
	`CREATE INDEX IF NOT EXISTS idx_spend_logs_effective_time ON "LiteLLM_SpendLogs" (COALESCE("endTime","startTime"), request_id)`,
}

// ensureSpendIndexes 幂等建索引：已存在则跳过（IF NOT EXISTS），LiteLLM 重建表后重启自动补建。
func ensureSpendIndexes(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		return
	}
	for _, ddl := range spendIndexes {
		if _, err := sqlDB.Exec(ddl); err != nil {
			logger.Bg().Mod("system").Warn(fmt.Sprintf("ensure spend index 失败(不影响启动): %v", err))
		}
	}
}

// GormSpend 初始化 LiteLLM spend logs 只读连接。
// litellm.spend-dsn 留空时复用主库 OPS_DB（prod 共享 devops_admin 库场景）；
// dev 的 litellm 独立库需配 spend-dsn。连接用于查 public."LiteLLM_SpendLogs" 表（只读，不 AutoMigrate）。
func GormSpend() *gorm.DB {
	dsn := global.OPS_CONFIG.Litellm.SpendDSN
	if dsn == "" {
		logger.Bg().Mod("system").Info("litellm.spend-dsn 未配置，用量回流复用主库(适用 prod 共享库)")
		ensureSpendIndexes(global.OPS_DB)
		return global.OPS_DB
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		logger.Bg().Mod("system").Err(err).Error("litellm spend db 连接失败，用量回流将不可用")
		fmt.Printf("litellm spend db 连接失败: %v\n", err)
		ensureSpendIndexes(global.OPS_DB) // 降级复用主库，避免阻断启动
		return global.OPS_DB
	}
	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.SetMaxOpenConns(20)
		sqlDB.SetMaxIdleConns(5)
	}
	logger.Bg().Mod("system").Info("litellm spend db 连接成功")
	ensureSpendIndexes(db)
	return db
}
