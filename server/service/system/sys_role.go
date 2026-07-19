package system

import (
	"context"
	"errors"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
	"gorm.io/gorm"
)

// RoleService 角色业务服务(对齐前端 /system/role/* 资源)
type RoleService struct{}

// roleOrder 角色统一排序:role_sort 升序,同序按 role_id 升序。
const roleOrder = "role_sort ASC, role_id ASC"

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
		Remark:            req.Remark,
	}
	// CreateBy/UpdateBy 为内嵌 OPS_AUDIT_MODEL 的提升字段,struct literal 中不可直接命名,改用赋值写入。
	r.CreateBy = createBy
	r.UpdateBy = createBy
	menuIds := toInt64Slice(req.MenuIds)
	return global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&r).Error; err != nil {
			return err
		}
		return saveRoleMenus(tx, r.RoleId, menuIds)
	})
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
	return global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&system.SysRole{}).Where("role_id = ?", roleId).
			Updates(map[string]interface{}{
				"role_name":           req.RoleName,
				"role_key":            req.RoleKey,
				"role_sort":           req.RoleSort,
				"menu_check_strictly": req.MenuCheckStrictly,
				"status":              req.Status,
				"remark":              req.Remark,
				"update_by":           updateBy,
			}).Error; err != nil {
			return err
		}
		return saveRoleMenus(tx, roleId, menuIds)
	})
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
	return global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("sys_role_id IN ?", ids).Delete(&system.SysRoleMenu{}).Error; err != nil {
			return err
		}
		if err := tx.Where("sys_role_id IN ?", ids).Delete(&system.SysRoleDepartment{}).Error; err != nil {
			return err
		}
		return tx.Where("role_id IN ?", ids).Delete(&system.SysRole{}).Error
	})
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

// saveRoleMenus 全量替换角色菜单关联:先删 sys_role_menu where role_id,再批量插。
func saveRoleMenus(tx *gorm.DB, roleId int64, menuIds []int64) error {
	if err := tx.Where("sys_role_id = ?", roleId).Delete(&system.SysRoleMenu{}).Error; err != nil {
		return err
	}
	rows := make([]system.SysRoleMenu, 0, len(menuIds))
	for _, mid := range menuIds {
		if mid > 0 {
			rows = append(rows, system.SysRoleMenu{SysRoleId: roleId, SysMenuId: mid})
		}
	}
	if len(rows) == 0 {
		return nil
	}
	return tx.Create(&rows).Error
}

// toInt64Slice 将 []Int64String 转为 []int64。
func toInt64Slice(ids []systemReq.Int64String) []int64 {
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.Int64())
	}
	return out
}
