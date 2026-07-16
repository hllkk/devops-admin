package initialize

import (
	"os"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"

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
	err := db.AutoMigrate(

		// system.SysApi{},
		// system.SysIgnoreApi{},
		// system.SysUser{},
		// system.SysBaseMenu{},
		// system.JwtBlacklist{},
		// system.SysAuthority{},
		// system.SysDictionary{},
		// system.SysOperationRecord{},
		// system.SysAutoCodeHistory{},
		// system.SysDictionaryDetail{},
		// system.SysBaseMenuParameter{},
		// system.SysBaseMenuBtn{},
		// system.SysAuthorityBtn{},
		// system.SysAutoCodePackage{},
		// system.SysExportTemplate{},
		// system.Condition{},
		// system.JoinTemplate{},
		// system.SysParams{},
		// system.SysVersion{},
		system.SysError{},
		// system.SysApiToken{},
		// system.SysLoginLog{},

		// example.ExaFile{},
		// example.ExaCustomer{},
		// example.ExaFileChunk{},
		// example.ExaFileUploadAndDownload{},
		// example.ExaAttachmentCategory{},
	)
	if err != nil {
		global.OPS_LOG.Error("register table failed", zap.Error(err))
		os.Exit(1)
	}

	err = bizModel()

	if err != nil {
		global.OPS_LOG.Error("register biz_table failed", zap.Error(err))
		os.Exit(1)
	}
	global.OPS_LOG.Info("register table success")
}
