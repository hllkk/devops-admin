package system

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/common"
	"github.com/hllkk/devops-admin/server/model/system"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/datascope"
	"github.com/hllkk/devops-admin/server/utils/logger"
	"gorm.io/gorm"
)

// RoleService 角色业务服务(对齐前端 /system/role/* 资源)
type RoleService struct{}

// roleOrder 角色统一排序:role_sort 升序,同序按 role_id 升序。
const roleOrder = "role_sort ASC, role_id ASC"

// GetRoleOptionList 获取启用角色列表(选择框用,不分页;对齐前端 GET /system/role/optionselect)。
// 返回 status='0' 的全部角色,前端 RoleSelect 渲染为下拉选项。
func (s *RoleService) GetRoleOptionList(ctx context.Context) (list []system.SysRole, err error) {
	err = global.OPS_DB.WithContext(ctx).Where("status = ?", "0").Order(roleOrder).Find(&list).Error
	return
}

// GetRoleList 分页查角色列表(对齐前端 GET /system/role/list)。
// roleName/roleKey 模糊,status 精确;分页走 PageInfo.LimitOffset。
func (s *RoleService) GetRoleList(ctx context.Context, q systemReq.RoleSearch) (list []system.SysRole, total int64, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&system.SysRole{})
	if q.RoleName != "" {
		db = db.Where("role_name LIKE ?", "%"+q.RoleName+"%")
	}
	if q.RoleKey != "" {
		db = db.Where("role_key LIKE ?", "%"+q.RoleKey+"%")
	}
	if q.Status != "" {
		db = db.Where("status = ?", q.Status)
	}
	limit, offset := q.LimitOffset()
	if limit > 0 {
		err = db.Count(&total).Order(roleOrder).Limit(limit).Offset(offset).Find(&list).Error
	} else {
		err = db.Count(&total).Order(roleOrder).Find(&list).Error
	}
	return
}

// CreateRole 新增角色 + 分配菜单(事务:roleKey 唯一校验 → 建角色 → 批量插 sys_role_menu)。
func (s *RoleService) CreateRole(ctx context.Context, req systemReq.RoleOperateParams, createBy int64) error {
	if req.RoleName == "" {
		return errors.New("角色名称不能为空")
	}
	if req.RoleKey == "" {
		return errors.New("角色权限字符不能为空")
	}
	var cnt int64
	if err := global.OPS_DB.WithContext(ctx).Model(&system.SysRole{}).Where("role_key = ?", req.RoleKey).Count(&cnt).Error; err != nil {
		return err
	}
	if cnt > 0 {
		return errors.New("角色权限字符已存在")
	}
	r := system.SysRole{
		RoleName:          req.RoleName,
		RoleKey:           req.RoleKey,
		RoleSort:          req.RoleSort,
		Status:            req.Status,
		MenuCheckStrictly: req.MenuCheckStrictly,
		DefaultRouter:     req.DefaultRouter,
		Remark:            req.Remark,
	}
	// CreateBy/UpdateBy 为内嵌 OPS_AUDIT_MODEL 的提升字段,struct literal 中不可直接命名,改用赋值写入。
	r.CreateBy = createBy
	r.UpdateBy = createBy
	menuIds := toInt64Slice(req.MenuIds)
	if err := global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&r).Error; err != nil {
			return err
		}
		return saveRoleMenus(tx, r.RoleId, menuIds)
	}); err != nil {
		return err
	}
	// 菜单授权事务成功后,同步该角色 casbin 接口策略(全量替换;失败仅告警,下次授权自愈)
	syncRoleCasbinPolicy(r.RoleId, menuIds)
	return nil
}

// UpdateRole 修改角色 + 全量替换菜单分配(事务:roleKey 唯一排除自身 → 更新角色 → 删后插 sys_role_menu)。
// 注:不更新 SuperAdmin/DataScope(前端 RoleOperateParams 未含,走保留值)。
func (s *RoleService) UpdateRole(ctx context.Context, req systemReq.RoleOperateParams, updateBy int64) error {
	roleId := req.RoleId.Int64()
	if roleId == 0 {
		return errors.New("角色ID不能为空")
	}
	if req.RoleName == "" {
		return errors.New("角色名称不能为空")
	}
	if req.RoleKey == "" {
		return errors.New("角色权限字符不能为空")
	}
	var cnt int64
	if err := global.OPS_DB.WithContext(ctx).Model(&system.SysRole{}).
		Where("role_key = ? AND role_id <> ?", req.RoleKey, roleId).Count(&cnt).Error; err != nil {
		return err
	}
	if cnt > 0 {
		return errors.New("角色权限字符已存在")
	}
	menuIds := toInt64Slice(req.MenuIds)
	if err := global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&system.SysRole{}).Where("role_id = ?", roleId).
			Updates(map[string]interface{}{
				"role_name":           req.RoleName,
				"role_key":            req.RoleKey,
				"role_sort":           req.RoleSort,
				"menu_check_strictly": req.MenuCheckStrictly,
				"status":              req.Status,
				"default_router":      req.DefaultRouter,
				"remark":              req.Remark,
				"update_by":           updateBy,
			}).Error; err != nil {
			return err
		}
		return saveRoleMenus(tx, roleId, menuIds)
	}); err != nil {
		return err
	}
	// 菜单授权事务成功后,同步该角色 casbin 接口策略(全量替换;失败仅告警,下次授权自愈)
	syncRoleCasbinPolicy(roleId, menuIds)
	return nil
}

// UpdateRoleStatus 修改角色状态(对齐前端 PUT /system/role/changeStatus)。
func (s *RoleService) UpdateRoleStatus(ctx context.Context, req systemReq.RoleOperateParams, updateBy int64) error {
	roleId := req.RoleId.Int64()
	if roleId == 0 {
		return errors.New("角色ID不能为空")
	}
	return global.OPS_DB.WithContext(ctx).Model(&system.SysRole{}).Where("role_id = ?", roleId).
		Updates(map[string]interface{}{"status": req.Status, "update_by": updateBy}).Error
}

// DeleteRole 批量删除角色;被用户引用(sys_user_role)时禁删(对齐 RuoYi);清理 sys_role_menu/sys_role_departments。
func (s *RoleService) DeleteRole(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return errors.New("未选择删除项")
	}
	var userCnt int64
	if err := global.OPS_DB.WithContext(ctx).Model(&system.SysUserRole{}).
		Where("sys_role_id IN ?", ids).Count(&userCnt).Error; err != nil {
		return err
	}
	if userCnt > 0 {
		return errors.New("角色已分配给用户,不允许删除")
	}
	if err := global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("sys_role_id IN ?", ids).Delete(&system.SysRoleMenu{}).Error; err != nil {
			return err
		}
		if err := tx.Where("sys_role_id IN ?", ids).Delete(&system.SysRoleDepartment{}).Error; err != nil {
			return err
		}
		return tx.Where("role_id IN ?", ids).Delete(&system.SysRole{}).Error
	}); err != nil {
		return err
	}
	// 角色删除后,清理其 casbin 接口策略(防孤儿策略残留)
	clearRolesCasbinPolicy(ids)
	return nil
}

// GetAllocatedUserList 角色已分配用户分页(join sys_user_role,对齐前端 GET /system/role/authUser/allocatedList)。
func (s *RoleService) GetAllocatedUserList(ctx context.Context, q systemReq.RoleUserSearch) (list []system.SysUser, total int64, err error) {
	if q.RoleId == 0 {
		return nil, 0, errors.New("角色ID不能为空")
	}
	db := global.OPS_DB.WithContext(ctx).Model(&system.SysUser{}).
		Joins("JOIN sys_user_role ON sys_user_role.sys_user_id = sys_users.id").
		Where("sys_user_role.sys_role_id = ?", q.RoleId)
	if q.UserName != "" {
		db = db.Where("sys_users.user_name LIKE ?", "%"+q.UserName+"%")
	}
	if q.Phonenumber != "" {
		db = db.Where("sys_users.phonenumber LIKE ?", "%"+q.Phonenumber+"%")
	}
	limit, offset := q.LimitOffset()
	if limit > 0 {
		err = db.Count(&total).Limit(limit).Offset(offset).Find(&list).Error
	} else {
		err = db.Count(&total).Find(&list).Error
	}
	return
}

// AuthUserSelectAll 批量给角色授权用户(去重已有后批量插 sys_user_role,对齐前端 PUT /system/role/authUser/selectAll)。
func (s *RoleService) AuthUserSelectAll(ctx context.Context, roleId int64, userIds []int64) error {
	if roleId == 0 {
		return errors.New("角色ID不能为空")
	}
	if len(userIds) == 0 {
		return errors.New("未选择用户")
	}
	var existIds []int64
	if err := global.OPS_DB.WithContext(ctx).Model(&system.SysUserRole{}).
		Where("sys_role_id = ? AND sys_user_id IN ?", roleId, userIds).Pluck("sys_user_id", &existIds).Error; err != nil {
		return err
	}
	existSet := make(map[int64]bool, len(existIds))
	for _, id := range existIds {
		existSet[id] = true
	}
	rows := make([]system.SysUserRole, 0, len(userIds))
	for _, uid := range userIds {
		if uid > 0 && !existSet[uid] {
			rows = append(rows, system.SysUserRole{SysUserId: uid, SysRoleId: roleId})
		}
	}
	if len(rows) == 0 {
		return nil
	}
	return global.OPS_DB.WithContext(ctx).Create(&rows).Error
}

// AuthUserCancelAll 批量取消角色用户授权(对齐前端 PUT /system/role/authUser/cancelAll)。
func (s *RoleService) AuthUserCancelAll(ctx context.Context, roleId int64, userIds []int64) error {
	if roleId == 0 {
		return errors.New("角色ID不能为空")
	}
	if len(userIds) == 0 {
		return errors.New("未选择用户")
	}
	return global.OPS_DB.WithContext(ctx).
		Where("sys_role_id = ? AND sys_user_id IN ?", roleId, userIds).Delete(&system.SysUserRole{}).Error
}

// saveRoleMenus 全量替换角色菜单关联:先删 sys_role_menu where role_id,再按 menuId 去重批量插。
// 去重防御:复合主键表必须按值去重(前端 getCheckedMenuIds 可能产出 number/string 同值重复)。
func saveRoleMenus(tx *gorm.DB, roleId int64, menuIds []int64) error {
	if err := tx.Where("sys_role_id = ?", roleId).Delete(&system.SysRoleMenu{}).Error; err != nil {
		return err
	}
	seen := make(map[int64]struct{}, len(menuIds))
	rows := make([]system.SysRoleMenu, 0, len(menuIds))
	for _, mid := range menuIds {
		if mid <= 0 {
			continue
		}
		if _, ok := seen[mid]; ok {
			continue
		}
		seen[mid] = struct{}{}
		rows = append(rows, system.SysRoleMenu{SysRoleId: roleId, SysMenuId: mid})
	}
	if len(rows) == 0 {
		return nil
	}
	return tx.Create(&rows).Error
}

// syncRoleCasbinPolicy 全量替换角色的 casbin 接口策略:按 menuIds 查菜单 api_prefix,
// 组装 (roleId, pattern, *) 策略,先清旧再写新。在 saveRoleMenus 事务成功后调用,
// 用 enforcer API 操作(自动落 casbin_rule 表 + 刷新缓存);失败仅记日志告警(全量替换语义,下次授权自愈)。
func syncRoleCasbinPolicy(roleId int64, menuIds []int64) {
	e := utils.GetCasbin()
	roleIdStr := strconv.FormatInt(roleId, 10)
	if e == nil {
		logger.Bg().Mod("rbac").Warn("casbin enforcer 未初始化, 跳过角色策略同步: roleId=" + roleIdStr)
		return
	}
	// 全量替换:先清该角色全部策略(含 menuIds 为空时仅清不写,用于角色被收回全部菜单)
	if _, err := e.RemoveFilteredPolicy(0, roleIdStr); err != nil {
		logger.Bg().Mod("rbac").Err(err).Error("清理角色旧 casbin 策略失败: roleId=" + roleIdStr)
		return
	}
	if len(menuIds) == 0 {
		return
	}
	// 查菜单 api_prefix,按逗号拆分得 pattern 列表
	var menus []system.SysMenu
	if err := global.OPS_DB.Select("api_prefix").Where("menu_id IN ?", menuIds).Find(&menus).Error; err != nil {
		logger.Bg().Mod("rbac").Err(err).Error("查询菜单 api_prefix 失败: roleId=" + roleIdStr)
		return
	}
	patterns := expandApiPrefixes(menus)
	if len(patterns) == 0 {
		return
	}
	rules := make([][]string, 0, len(patterns))
	for _, p := range patterns {
		rules = append(rules, []string{roleIdStr, p, "*"})
	}
	if _, err := e.AddPolicies(rules); err != nil {
		logger.Bg().Mod("rbac").Err(err).Error("写入角色 casbin 策略失败: roleId=" + roleIdStr)
	}
}

// expandApiPrefixes 从菜单列表展开 casbin obj pattern:按逗号拆分每个菜单的 ApiPrefix,
// 去空白、去重、跳过空值,返回有序唯一的 pattern 列表(供 syncRoleCasbinPolicy 组装策略)。
func expandApiPrefixes(menus []system.SysMenu) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(menus))
	for _, m := range menus {
		if strings.TrimSpace(m.ApiPrefix) == "" {
			continue
		}
		for _, p := range strings.Split(m.ApiPrefix, ",") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			out = append(out, p)
		}
	}
	return out
}

// clearRolesCasbinPolicy 批量清理角色的 casbin 接口策略(删角色后调用,防孤儿策略)。
func clearRolesCasbinPolicy(roleIds []int64) {
	e := utils.GetCasbin()
	if e == nil || len(roleIds) == 0 {
		return
	}
	for _, roleId := range roleIds {
		if _, err := e.RemoveFilteredPolicy(0, strconv.FormatInt(roleId, 10)); err != nil {
			logger.Bg().Mod("rbac").Err(err).Error("清理已删角色 casbin 策略失败: roleId=" + strconv.FormatInt(roleId, 10))
		}
	}
}

// toInt64Slice 将 []common.Int64String 转为 []int64。
func toInt64Slice(ids []common.Int64String) []int64 {
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.Int64())
	}
	return out
}

// ExportRoleList 按列表查询条件导出角色(全量,不分页;过滤条件与 GetRoleList 一致,加导出上限)。
func (s *RoleService) ExportRoleList(ctx context.Context, q systemReq.RoleSearch) (list []system.SysRole, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&system.SysRole{})
	if q.RoleName != "" {
		db = db.Where("role_name LIKE ?", "%"+q.RoleName+"%")
	}
	if q.RoleKey != "" {
		db = db.Where("role_key LIKE ?", "%"+q.RoleKey+"%")
	}
	if q.Status != "" {
		db = db.Where("status = ?", q.Status)
	}
	err = db.Order(roleOrder).Limit(ExportMaxRows).Find(&list).Error
	return
}

// UpdateRoleDataScope 分配角色数据权限(对齐前端 PUT /system/role/dataScope)。
// 事务:更新 sys_roles.data_scope/dept_check_strictly;档位=5(自定义)全量替换 sys_role_departments,非 5 清空。
// 超管角色(SuperAdmin)禁止改数据权限,防止被降级。
func (s *RoleService) UpdateRoleDataScope(ctx context.Context, req systemReq.RoleOperateParams, updateBy int64) error {
	roleId := req.RoleId.Int64()
	if roleId == 0 {
		return errors.New("角色ID不能为空")
	}
	if req.DataScope < datascope.ScopeAll || req.DataScope > datascope.ScopeCustom {
		return errors.New("数据范围档位非法")
	}
	var role system.SysRole
	if err := global.OPS_DB.WithContext(ctx).Select("role_id", "super_admin").
		Where("role_id = ?", roleId).First(&role).Error; err != nil {
		return errors.New("角色不存在")
	}
	if role.SuperAdmin {
		return errors.New("超级管理员角色不允许修改数据权限")
	}
	deptIds := toInt64Slice(req.DeptIds)
	return global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&system.SysRole{}).Where("role_id = ?", roleId).
			Updates(map[string]interface{}{
				"data_scope":          req.DataScope,
				"dept_check_strictly": req.DeptCheckStrictly,
				"update_by":           updateBy,
			}).Error; err != nil {
			return err
		}
		return saveRoleDepartments(tx, roleId, deptIds, req.DataScope)
	})
}

// GetRoleDeptTreeSelect 角色数据权限部门树(对齐前端 GET /system/role/deptTree/{roleId})。
// depts=启用部门树(复用 DepartmentService.GetDeptTree);checkedKeys=该角色自定义部门集(sys_role_departments 全量,忠实往返)。
func (s *RoleService) GetRoleDeptTreeSelect(ctx context.Context, roleId int64) (result system.RoleDeptTreeSelect, err error) {
	if roleId == 0 {
		return result, errors.New("角色ID不能为空")
	}
	result.Depts, err = (&DepartmentService{}).GetDeptTree(ctx)
	if err != nil {
		return
	}
	var checkedKeys []int64
	if err = global.OPS_DB.WithContext(ctx).Model(&system.SysRoleDepartment{}).
		Where("sys_role_id = ?", roleId).Pluck("sys_department_id", &checkedKeys).Error; err != nil {
		return
	}
	result.CheckedKeys = common.Int64StringSlice(checkedKeys)
	return
}

// saveRoleDepartments 按档位维护角色-部门关联:自定义档(5)全量替换,其余档清空(不依赖部门集)。
func saveRoleDepartments(tx *gorm.DB, roleId int64, deptIds []int64, scope int) error {
	if err := tx.Where("sys_role_id = ?", roleId).Delete(&system.SysRoleDepartment{}).Error; err != nil {
		return err
	}
	if scope != datascope.ScopeCustom {
		return nil
	}
	rows := make([]system.SysRoleDepartment, 0, len(deptIds))
	for _, did := range deptIds {
		if did > 0 {
			rows = append(rows, system.SysRoleDepartment{SysRoleId: roleId, SysDepartmentId: did})
		}
	}
	if len(rows) == 0 {
		return nil
	}
	return tx.Create(&rows).Error
}
