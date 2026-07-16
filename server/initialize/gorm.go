package initialize

import (
	"os"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/hllkk/devops-admin/server/global"
	sysSvc "github.com/hllkk/devops-admin/server/service/system"

	_ "github.com/hllkk/devops-admin/server/source/system" // 触发 seed initializer 的 init() 自注册（建表清单的唯一真相源）
)

func Gorm() *gorm.DB {
	switch global.OPS_CONFIG.System.DbType {
	case "mysql":
		global.OPS_ACTIVE_DBNAME = &global.OPS_CONFIG.Mysql.Dbname
		return GormMysql()
	case "pgsql":
		global.OPS_ACTIVE_DBNAME = &global.OPS_CONFIG.Pgsql.Dbname
		return GormPgSql()
	case "oracle":
		global.OPS_ACTIVE_DBNAME = &global.OPS_CONFIG.Oracle.Dbname
		return GormOracle()
	case "mssql":
		global.OPS_ACTIVE_DBNAME = &global.OPS_CONFIG.Mssql.Dbname
		return GormMssql()
	case "sqlite":
		global.OPS_ACTIVE_DBNAME = &global.OPS_CONFIG.Sqlite.Dbname
		return GormSqlite()
	default:
		global.OPS_ACTIVE_DBNAME = &global.OPS_CONFIG.Mysql.Dbname
		return GormMysql()
	}
}

func RegisterTables() {
	if global.OPS_CONFIG.System.DisableAutoMigrate {
		global.OPS_LOG.Info("auto-migrate is disabled, skipping table registration")
		return
	}

	db := global.OPS_DB
	// 建表清单的唯一真相源是 source/system 下各 initializer 的 MigrateTable（有数据
	// 的 seed initializer + initAutoMigrate 收纳的无数据表）；与首次 InitDB 的
	// createTables 共用 sysSvc.MigrateRegisteredTables，不再维护独立的全量清单。
	if err := sysSvc.MigrateRegisteredTables(db); err != nil {
		global.OPS_LOG.Error("register table failed", zap.Error(err))
		os.Exit(1)
	}

	global.OPS_LOG.Info("register table success")

	// 启动时幂等数据迁移：为「已初始化」的旧库补齐后续新增的 seed 内容（菜单/字典/权限）
	// 或修复历史数据；未初始化（sys_users 为空）则跳过，交由前端向导触发 InitDB。
	runDataMigrations(db)
}
