package system

import (
	"errors"
	"strconv"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/utils"
)

// CasbinService 权限策略服务。对照 GVA service/system/sys_casbin.go 的
// UpdateCasbin / ClearCasbin / FreshCasbin 模式，适配当前项目：
//   - 角色体系用 sys_role（RoleId）而非 GVA 的 sys_authority（AuthorityId）
//   - API 资源来自 sys_menu(C 型).apis（方式 B）而非 GVA 的 sys_api 独立表
//   - enforcer 为 SyncedCachedEnforcer（带缓存），RemoveFilteredPolicy 后需 InvalidateCache
type CasbinService struct{}

// UpdateCasbin 重算指定角色的 casbin 策略：清旧 → 按其 C 型菜单的 apis 去重重写 → 失效缓存。
// 被 init 的 sys_casbin initializer 与（未来）角色分配菜单的 service 调用。
func (s *CasbinService) UpdateCasbin(roleId int64) error {
	if global.OPS_DB == nil {
		return errors.New("数据库未初始化")
	}
	e := utils.GetCasbin()
	if e == nil {
		return errors.New("casbin enforcer 未初始化")
	}
	sub := strconv.FormatInt(roleId, 10)

	// 1. 清该角色旧策略（RemoveFilteredPolicy 不 cache-aware，第 4 步统一失效缓存）
	if _, err := e.RemoveFilteredPolicy(0, sub); err != nil {
		return err
	}

	// 2. 查该角色关联的 C 型菜单
	var menuIds []int64
	if err := global.OPS_DB.Model(&system.SysRoleMenu{}).
		Where("role_id = ?", roleId).Pluck("menu_id", &menuIds).Error; err != nil {
		return err
	}
	if len(menuIds) == 0 {
		_ = e.InvalidateCache()
		return nil
	}
	var menus []system.SysMenu
	if err := global.OPS_DB.Where("menu_id IN ? AND menu_type = ?", menuIds, "C").
		Find(&menus).Error; err != nil {
		return err
	}

	// 3. 遍历 apis 去重写策略（对照 GVA UpdateCasbin 的 deduplicateMap）
	rules := make([][]string, 0, len(menus)*2)
	seen := make(map[string]bool, len(menus)*2)
	for _, m := range menus {
		for _, api := range m.Apis {
			key := sub + api.Path + api.Method
			if seen[key] {
				continue
			}
			seen[key] = true
			rules = append(rules, []string{sub, api.Path, api.Method})
		}
	}
	if len(rules) > 0 {
		if _, err := e.AddPolicies(rules); err != nil {
			return err
		}
	}

	// 4. 失效缓存（SyncedCachedEnforcer 区别于 GVA 普通 Enforcer 的必要操作）
	_ = e.InvalidateCache()
	return nil
}
