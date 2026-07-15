package system

import (
	"context"
	"fmt"

	adapter "github.com/casbin/gorm-adapter/v3"
	"github.com/hllkk/devops-admin/server/model/system"
	sysSvc "github.com/hllkk/devops-admin/server/service/system"
	"gorm.io/gorm"
)

const initOrderCasbin = sysSvc.InitOrderSystem + 7

type initCasbin struct{}

func init() { sysSvc.RegisterInit(initOrderCasbin, &initCasbin{}) }

func (i *initCasbin) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, sysSvc.ErrMissingDBContext
	}
	return ctx, db.AutoMigrate(&adapter.CasbinRule{})
}

func (i *initCasbin) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	return db.Migrator().HasTable(&adapter.CasbinRule{})
}

func (i *initCasbin) InitializerName() string { return adapter.CasbinRule{}.TableName() }

// InitializeData 遍历所有角色，按其 C 菜单 apis 重算 casbin 策略。
// 方式 B：策略源是 sys_menu(C).apis，由 CasbinService.UpdateCasbin 推导，不硬编码 CasbinRule。
// 顺序 +7：必须在 role(+2)/menu(+3)/role_menu(+6) 之后执行。
func (i *initCasbin) InitializeData(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, sysSvc.ErrMissingDBContext
	}
	var roleIds []int64
	if err := db.Model(&system.SysRole{}).Pluck("role_id", &roleIds).Error; err != nil {
		return ctx, fmt.Errorf("casbin seed 查询角色失败: %w", err)
	}
	svc := sysSvc.CasbinService{}
	for _, rid := range roleIds {
		if err := svc.UpdateCasbin(rid); err != nil {
			return ctx, fmt.Errorf("casbin role %d 策略生成失败: %w", rid, err)
		}
	}
	return ctx, nil
}

// DataInserted 总返回 false：UpdateCasbin 幂等（清旧+重写），每次 initdb 都重算以保持一致。
func (i *initCasbin) DataInserted(ctx context.Context) bool { return false }
