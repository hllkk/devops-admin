package initialize

import (
	"os"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	sysService "github.com/hllkk/devops-admin/server/service/system"

	"go.uber.org/zap"
	"gorm.io/gorm"
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
	// 无 initializer 管理的表在此显式迁移;其余表复用已注册 initializer 的 MigrateTable,
	// 与 /initdb 共用单一真源,避免两套建表清单漂移(详见 service/system.MigrateRegisteredTables)。
	if err := db.AutoMigrate(
		system.JwtBlacklist{},
		system.SysError{},
	); err != nil {
		global.OPS_LOG.Error("register table failed", zap.Error(err))
		os.Exit(1)
	}

	if err := sysService.MigrateRegisteredTables(db); err != nil {
		global.OPS_LOG.Error("register table via initializers failed", zap.Error(err))
		os.Exit(1)
	}

	if err := bizModel(); err != nil {
		global.OPS_LOG.Error("register biz_table failed", zap.Error(err))
		os.Exit(1)
	}
	global.OPS_LOG.Info("register table success")
}
