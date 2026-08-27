package initialize

import (
	"os"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/gateway"
	"github.com/hllkk/devops-admin/server/model/media"
	"github.com/hllkk/devops-admin/server/model/system"
	sourceGateway "github.com/hllkk/devops-admin/server/source/gateway"
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

		gateway.Provider{},
		gateway.Credential{},
		gateway.ProviderPrefix{},
		gateway.Model{},
		gateway.ModelDeployment{},
		gateway.ModelVisibility{},
		gateway.ModelVisibilityUser{},
		gateway.AiKey{},
		gateway.KeyScenario{},
		gateway.LlmLog{},
		gateway.SyncState{},
		gateway.CostSummaryDaily{},
		gateway.RouterSettings{},
		gateway.ProviderBalance{},
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

	// 供应商前缀差异表种子兜底（功能依赖数据，幂等 OnConflict DoNothing，已有环境重启自愈；
	// /initdb 路径由 source/gateway 初始化器链覆盖）。失败仅记日志不阻断启动。
	if err := sourceGateway.SeedProviderPrefix(db); err != nil {
		logger.Bg().Mod("gateway").Err(err).Error("seed gateway_provider_prefix failed")
	}
	// AiKey 同类归属下名称唯一索引兜底（防 LiteLLM key_alias 撞车；部分索引 WHERE deleted_at IS NULL，
	// 软删记录不占约束。/initdb 路径由 source/gateway 初始化器链覆盖）。失败仅记日志不阻断启动——
	// 存量脏数据环境重启不崩，名称查重由 service 层兜底。
	if err := sourceGateway.EnsureAiKeyUniqueIndex(db); err != nil {
		logger.Bg().Mod("gateway").Err(err).Error("ensure gateway_ai_key unique index failed")
	}
	// KeyScenario 名称唯一索引兜底(同上：软删行不占名，停用行占名防同名二义)。
	if err := sourceGateway.EnsureKeyScenarioUniqueIndex(db); err != nil {
		logger.Bg().Mod("gateway").Err(err).Error("ensure gateway_key_scenario unique index failed")
	}
	logger.Bg().Mod("system").Info("register table success")
}
