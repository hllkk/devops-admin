package initialize

import (
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/service/system"
)

// RebuildRoleCasbinPolicies 启动期按 sys_role_menu 现状全量重建各角色 casbin 接口策略(幂等, 可重复调用)。
// 背景:casbin 策略仅在角色授权保存时写入,存量库的角色授权产生于 casbin 强制校验挂载之前,
// casbin_rule 为空导致非超管登录后全局"权限不足";启动重建使其不依赖人工重新保存授权。
// 必须在 RegisterTables(建表)之后调用。
func RebuildRoleCasbinPolicies() {
	if global.OPS_DB == nil {
		return
	}
	system.RoleServiceApp.RebuildAllRoleCasbinPolicies()
}
