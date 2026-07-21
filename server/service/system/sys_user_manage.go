package system

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
	"github.com/hllkk/devops-admin/server/utils"
	"gorm.io/gorm"
)

// 用户管理 CRUD(对齐前端 /system/user/* 管理员侧资源)。
// auth 链路(Login/Register/GetUserInfo/GetUserDetail)仍在 sys_user.go,本文件只含管理方法。

// userOrder 用户列表统一排序:主键 id(雪花)升序。限定 sys_users.id 以避免 roleId join 时列名歧义。
const userOrder = "sys_users.id ASC"

// GetList 分页查用户列表(对齐前端 GET /system/user/list)。
// deptId/userName/nickName/phonenumber/status 过滤;roleId>0 时 join sys_user_role;分页走 LimitOffset。
func (s *UserService) GetList(ctx context.Context, q systemReq.UserSearch) (list []system.SysUser, total int64, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&system.SysUser{})
	if q.UserName != "" {
		db = db.Where("user_name LIKE ?", "%"+q.UserName+"%")
	}
	if q.NickName != "" {
		db = db.Where("nick_name LIKE ?", "%"+q.NickName+"%")
	}
	if q.Phonenumber != "" {
		db = db.Where("phonenumber LIKE ?", "%"+q.Phonenumber+"%")
	}
	if q.Status != "" {
		db = db.Where("status = ?", q.Status)
	}
	if q.DeptId > 0 {
		db = db.Where("dept_id = ?", q.DeptId)
	}
	if q.RoleId > 0 {
		db = db.Joins("JOIN sys_user_role ON sys_user_role.sys_user_id = sys_users.id").
			Where("sys_user_role.sys_role_id = ?", q.RoleId)
	}
	limit, offset := q.LimitOffset()
	db = db.Preload("Dept").Preload("Roles") // 主部门供回填 deptName;Roles 供回填 superAdmin(任一角色 SuperAdmin=true)
	if limit > 0 {
		err = db.Count(&total).Order(userOrder).Limit(limit).Offset(offset).Find(&list).Error
	} else {
		err = db.Count(&total).Order(userOrder).Find(&list).Error
	}
	if err == nil {
		// deptName/superAdmin 均为 gorm:"-" 内存字段,Preload 后显式回填(前端列表部门列+超管保护依赖)
		for i := range list {
			list[i].DeptName = list[i].Dept.DeptName
			list[i].SuperAdmin = list[i].GetSuperAdmin()
		}
	}
	return
}

// GetDeptUserList 部门下用户列表(对齐前端 GET /system/user/list/dept/{deptId},部门负责人选择用,不分页)。
func (s *UserService) GetDeptUserList(ctx context.Context, deptId int64) (list []system.SysUser, err error) {
	err = global.OPS_DB.WithContext(ctx).Where("dept_id = ? AND status = ?", deptId, "0").Order(userOrder).Find(&list).Error
	return
}

// Create 新增用户(事务:用户名唯一校验 → 建用户[bcrypt 密码 + UUID + PasswordUpdatedAt] → 保存 sys_user_role/sys_user_post)。
func (s *UserService) Create(ctx context.Context, req systemReq.UserOperateParams, createBy int64) error {
	if req.UserName == "" {
		return errors.New("用户名不能为空")
	}
	if req.NickName == "" {
		return errors.New("昵称不能为空")
	}
	if req.Password == "" {
		return errors.New("密码不能为空")
	}
	if err := utils.ValidatePasswordComplexity(req.Password, (&SecurityConfigService{}).Current(ctx)); err != nil {
		return err
	}
	var cnt int64
	if err := global.OPS_DB.WithContext(ctx).Model(&system.SysUser{}).Where("user_name = ?", req.UserName).Count(&cnt).Error; err != nil {
		return err
	}
	if cnt > 0 {
		return errors.New("用户名已存在")
	}
	u := system.SysUser{
		DeptId:      req.DeptId.Int64(),
		UserName:    req.UserName,
		NickName:    req.NickName,
		Email:       req.Email,
		Phonenumber: req.Phonenumber,
		Sex:         req.Sex,
		Status:      req.Status,
		Remark:      req.Remark,
		UUID:        uuid.New(),
	}
	u.Password = utils.BcryptHash(req.Password)
	now := time.Now()
	u.PasswordUpdatedAt = &now
	u.CreateBy = createBy
	u.UpdateBy = createBy
	roleIds := toInt64Slice(req.RoleIds)
	if len(roleIds) > 0 {
		u.RoleId = roleIds[0] // 主角色(登录链路 claims 用),取所选角色第一个回填,避免落 default 888
	}
	postIds := toInt64Slice(req.PostIds)
	return global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&u).Error; err != nil {
			return err
		}
		if err := saveUserRoles(tx, u.UserId, roleIds); err != nil {
			return err
		}
		return saveUserPosts(tx, u.UserId, postIds)
	})
}

// Update 修改用户(事务:更新用户[password 空=不改] + 全量替换 sys_user_role/sys_user_post)。
func (s *UserService) Update(ctx context.Context, req systemReq.UserOperateParams, updateBy int64) error {
	userId := req.UserId.Int64()
	if userId == 0 {
		return errors.New("用户ID不能为空")
	}
	if req.NickName == "" {
		return errors.New("昵称不能为空")
	}
	updates := map[string]interface{}{
		"dept_id":     req.DeptId.Int64(),
		"nick_name":   req.NickName,
		"email":       req.Email,
		"phonenumber": req.Phonenumber,
		"sex":         req.Sex,
		"status":      req.Status,
		"remark":      req.Remark,
		"update_by":   updateBy,
	}
	changePwd := req.Password != ""
	if changePwd {
		if err := utils.ValidatePasswordComplexity(req.Password, (&SecurityConfigService{}).Current(ctx)); err != nil {
			return err
		}
		updates["password"] = utils.BcryptHash(req.Password)
	}
	roleIds := toInt64Slice(req.RoleIds)
	if len(roleIds) > 0 {
		updates["role_id"] = roleIds[0] // 主角色随所选角色同步,避免落 default 888
	}
	postIds := toInt64Slice(req.PostIds)
	return global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&system.SysUser{}).Where("id = ?", userId).Updates(updates).Error; err != nil {
			return err
		}
		if changePwd {
			// PasswordUpdatedAt 为指针字段,map 不便,单独 Update
			if err := tx.Model(&system.SysUser{}).Where("id = ?", userId).Update("password_updated_at", time.Now()).Error; err != nil {
				return err
			}
		}
		if err := saveUserRoles(tx, userId, roleIds); err != nil {
			return err
		}
		return saveUserPosts(tx, userId, postIds)
	})
}

// UpdateStatus 修改用户状态(对齐前端 PUT /system/user/changeStatus)。
func (s *UserService) UpdateStatus(ctx context.Context, req systemReq.UserOperateParams, updateBy int64) error {
	userId := req.UserId.Int64()
	if userId == 0 {
		return errors.New("用户ID不能为空")
	}
	return global.OPS_DB.WithContext(ctx).Model(&system.SysUser{}).Where("id = ?", userId).
		Updates(map[string]interface{}{"status": req.Status, "update_by": updateBy}).Error
}

// Delete 批量删除用户(事务:清理 sys_user_role/sys_user_post/sys_user_departments → 删用户)。
func (s *UserService) Delete(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return errors.New("未选择删除项")
	}
	return global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("sys_user_id IN ?", ids).Delete(&system.SysUserRole{}).Error; err != nil {
			return err
		}
		if err := tx.Where("sys_user_id IN ?", ids).Delete(&system.SysUserPost{}).Error; err != nil {
			return err
		}
		if err := tx.Where("sys_user_id IN ?", ids).Delete(&system.SysUserDepartment{}).Error; err != nil {
			return err
		}
		return tx.Where("id IN ?", ids).Delete(&system.SysUser{}).Error
	})
}

// GetDetail 用户详情(对齐前端 GET /system/user/{userId}):postIds/roleIds(字符串)/roles(供 drawer 回显+下拉)。
func (s *UserService) GetDetail(ctx context.Context, userId int64) (result system.UserInfo, err error) {
	var u system.SysUser
	if err = global.OPS_DB.WithContext(ctx).Preload("Roles").Where("id = ?", userId).First(&u).Error; err != nil {
		return
	}
	result.Roles = u.Roles
	var roleIds []int64
	global.OPS_DB.WithContext(ctx).Model(&system.SysUserRole{}).Where("sys_user_id = ?", userId).Pluck("sys_role_id", &roleIds)
	result.RoleIds = int64ToStrSlice(roleIds)
	var postIds []int64
	global.OPS_DB.WithContext(ctx).Model(&system.SysUserPost{}).Where("sys_user_id = ?", userId).Pluck("sys_post_id", &postIds)
	result.PostIds = int64ToStrSlice(postIds)
	return
}

// ResetPwd 重置用户密码(对齐前端 PUT /system/user/resetPwd;明文 bcrypt 存储,无加密中间件)。
func (s *UserService) ResetPwd(ctx context.Context, req systemReq.ResetUserPwdParams, updateBy int64) error {
	userId := req.UserId.Int64()
	if userId == 0 {
		return errors.New("用户ID不能为空")
	}
	if req.Password == "" {
		return errors.New("密码不能为空")
	}
	if err := utils.ValidatePasswordComplexity(req.Password, (&SecurityConfigService{}).Current(ctx)); err != nil {
		return err
	}
	return global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&system.SysUser{}).Where("id = ?", userId).
			Updates(map[string]interface{}{"password": utils.BcryptHash(req.Password), "update_by": updateBy}).Error; err != nil {
			return err
		}
		return tx.Model(&system.SysUser{}).Where("id = ?", userId).Update("password_updated_at", time.Now()).Error
	})
}

// ChangeMyPassword 当前用户自助改密(密码过期强制改密场景的唯一入口)。
// 流程:校验旧密码 → 新旧不同 → 复杂度校验 → bcrypt 新密码 → 刷 PasswordUpdatedAt。
// Preload Roles 以保证 GetSuperAdmin 正确(超管改密重发 token 需 SuperAdmin=true)。
// 返回带 Roles 的 user,供 handler 重发 MustChangePwd=false token。
func (s *UserService) ChangeMyPassword(ctx context.Context, userId int64, oldPwd, newPwd string) (system.SysUser, error) {
	var u system.SysUser
	if err := global.OPS_DB.WithContext(ctx).Preload("Roles").Where("id = ?", userId).First(&u).Error; err != nil {
		return u, errors.New("用户不存在")
	}
	if !utils.BcryptCheck(oldPwd, u.Password) {
		return u, errors.New("旧密码错误")
	}
	if oldPwd == newPwd {
		return u, errors.New("新密码不能与旧密码相同")
	}
	if err := utils.ValidatePasswordComplexity(newPwd, (&SecurityConfigService{}).Current(ctx)); err != nil {
		return u, err
	}
	now := time.Now()
	return u, global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&system.SysUser{}).Where("id = ?", userId).
			Update("password", utils.BcryptHash(newPwd)).Error; err != nil {
			return err
		}
		return tx.Model(&system.SysUser{}).Where("id = ?", userId).
			Update("password_updated_at", now).Error
	})
}

// saveUserRoles 全量替换用户角色(删后批量插 sys_user_role)。
func saveUserRoles(tx *gorm.DB, userId int64, roleIds []int64) error {
	if err := tx.Where("sys_user_id = ?", userId).Delete(&system.SysUserRole{}).Error; err != nil {
		return err
	}
	rows := make([]system.SysUserRole, 0, len(roleIds))
	for _, rid := range roleIds {
		if rid > 0 {
			rows = append(rows, system.SysUserRole{SysUserId: userId, SysRoleId: rid})
		}
	}
	if len(rows) == 0 {
		return nil
	}
	return tx.Create(&rows).Error
}

// saveUserPosts 全量替换用户岗位(删后批量插 sys_user_post)。
func saveUserPosts(tx *gorm.DB, userId int64, postIds []int64) error {
	if err := tx.Where("sys_user_id = ?", userId).Delete(&system.SysUserPost{}).Error; err != nil {
		return err
	}
	rows := make([]system.SysUserPost, 0, len(postIds))
	for _, pid := range postIds {
		if pid > 0 {
			rows = append(rows, system.SysUserPost{SysUserId: userId, SysPostId: pid})
		}
	}
	if len(rows) == 0 {
		return nil
	}
	return tx.Create(&rows).Error
}

// int64ToStrSlice 将 []int64 转为 []string(对齐前端 string[])。
func int64ToStrSlice(ids []int64) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, strconv.FormatInt(id, 10))
	}
	return out
}
