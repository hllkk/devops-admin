package system

import (
	"testing"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupTestDB 初始化内存 sqlite + 建权限相关表，并赋给 global.OPS_DB。
// sqlite `file::memory:?cache=shared` 使整个 go test 运行期间共享同一份内存库，
// 故 migrate 后用 Unscoped 清空所有表（SysUser 等带软删除，必须 Unscoped 才彻底清），
// 保证每个测试从干净状态开始。casbin_rule 由 GetCasbin() 懒创建，不在此清理。
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	tables := []interface{}{&system.SysUser{}, &system.SysRole{}, &system.SysMenu{}, &system.SysDept{},
		&system.SysPost{}, &system.SysUserRole{}, &system.SysRoleMenu{}}
	if err := db.AutoMigrate(tables...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	for _, m := range tables {
		if err := db.Unscoped().Where("1=1").Delete(m).Error; err != nil {
			t.Fatalf("clear %T: %v", m, err)
		}
	}
	global.OPS_DB = db
	return db
}
