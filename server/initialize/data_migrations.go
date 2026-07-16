package initialize

import (
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// dataMigration 单条幂等数据迁移：每次启动对「已初始化」的库执行，用于补齐 seed
// 增量（后续新增的菜单/字典/权限）或修复历史数据。必须幂等——可安全重复执行。
type dataMigration struct {
	name string
	fn   func(db *gorm.DB) error
}

// dataMigrations 数据迁移清单，按时间顺序追加。
//
// 为什么需要它：source/system 的 seed initializer 受 DataInserted 守卫保护（如
// sys_menu 查到 menu_id=1500 即整体跳过 InitializeData），已初始化的旧库不会再跑
// seed，因此后续新增的 seed 内容进不了旧库。本清单在每次启动时对已初始化库幂等补
// 数据，新老库都覆盖。
//
// 新增迁移步骤：
//  1. 在本文件（或新建 data_migration_xxx.go）写一个幂等函数，用 FirstOrCreate /
//     OnConflict / Where(...).Not(...) 注入缺失行；
//  2. 把 {name, fn} 追加到下方清单。
//
// 示例：
//
//	func ensureLogFileMenu(db *gorm.DB) error {
//	    return db.Where("menu_id = ?", 2300).
//	        Attrs(system.SysMenu{MenuName: "文件日志", ...}).
//	        FirstOrCreate(&system.SysMenu{}).Error
//	}
//
//	var dataMigrations = []dataMigration{
//	    {name: "ensure_log_file_menu", fn: ensureLogFileMenu},
//	}
var dataMigrations []dataMigration

// runDataMigrations 执行所有幂等数据迁移。
//
// 仅作用于「已初始化」的库：sys_users 为空 → 视为未初始化 → 全部跳过，等待前端向导
// 触发 InitDB（source/system seed）统一建数据。无迁移时静默返回。
func runDataMigrations(db *gorm.DB) {
	if db == nil || len(dataMigrations) == 0 {
		return
	}
	var userCount int64
	if err := db.Model(&system.SysUser{}).Count(&userCount).Error; err != nil {
		global.OPS_LOG.Warn("数据迁移跳过: 无法统计 sys_users", zap.Error(err))
		return
	}
	if userCount == 0 {
		global.OPS_LOG.Info("数据迁移跳过: 系统尚未初始化，等待前端触发 InitDB")
		return
	}
	for _, m := range dataMigrations {
		if err := m.fn(db); err != nil {
			global.OPS_LOG.Error("数据迁移失败", zap.String("migration", m.name), zap.Error(err))
		}
	}
	global.OPS_LOG.Info("数据迁移完成", zap.Int("count", len(dataMigrations)))
}
