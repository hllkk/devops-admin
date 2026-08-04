package initialize

import (
	"os"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/media"
	"github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/utils/logger"

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
		logger.Bg().Mod("system").Info("auto-migrate is disabled, skipping table registration")
		return
	}

	db := global.OPS_DB
	// 重启路径独立建表清单(与 /initdb 的 initializer 清单分离, 内容保持等价)
	err := db.AutoMigrate(
		system.SysUser{},
		system.SysMenu{},
		system.JwtBlacklist{},
		system.SysRole{},
		system.SysDepartment{},
		system.SysPost{},
		system.SysLoginLog{},
		system.SysOperLog{},
		system.SysDataAccessLog{},
		system.SysRoleDepartment{},
		system.SysDictType{},
		system.SysDictData{},
		system.SysRoleMenu{},
		system.SysSocial{},
		system.SysError{},
		system.SysSecurityConfig{},
		system.SysGeneralConfig{},
		system.SysLdapConfig{},
		system.SysNotifyConfig{},
		system.SysAuthConfig{},
		system.SysTimedTask{},
		system.SysTimedTaskLog{},

		media.MediaUpload{},
		media.MediaUploadChunk{},
		media.FileUploadAndDownload{},
		media.AttachmentCategory{},
	)

	if err != nil {
		logger.Bg().Mod("system").Err(err).Error("register table failed")
		os.Exit(1)
	}

	err = bizModel()

	if err != nil {
		logger.Bg().Mod("system").Err(err).Error("register biz_table failed")
		os.Exit(1)
	}
	logger.Bg().Mod("system").Info("register table success")
}
