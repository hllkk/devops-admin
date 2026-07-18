package system

import (
	"context"
	"errors"
	"time"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
	"github.com/hllkk/devops-admin/server/utils"
	"gorm.io/gorm"
)

type UserService struct{}

// Login 校验用户名密码 返回带 Roles 的用户(登录链路 claims 需要 SuperAdmin)
func (userService *UserService) Login(ctx context.Context, u *system.SysUser) (user system.SysUser, err error) {
	err = global.OPS_DB.WithContext(ctx).
		Preload("Roles").
		Where("user_name = ?", u.UserName).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errors.New("用户名不存在或密码错误")
		}
		return user, err
	}
	if !utils.BcryptCheck(u.Password, user.Password) {
		return user, errors.New("用户名不存在或密码错误")
	}
	return user, nil
}

// Register 注册(管理员建号) 用户名唯一 + bcrypt 加密
func (userService *UserService) Register(ctx context.Context, u system.SysUser) (system.SysUser, error) {
	var count int64
	if err := global.OPS_DB.WithContext(ctx).Model(&system.SysUser{}).Where("user_name = ?", u.UserName).Count(&count).Error; err != nil {
		return u, err
	}
	if count > 0 {
		return u, errors.New("用户名已存在")
	}
	u.Password = utils.BcryptHash(u.Password)
	now := time.Now()
	u.PasswordUpdatedAt = &now
	if err := global.OPS_DB.WithContext(ctx).Create(&u).Error; err != nil {
		return u, err
	}
	return u, nil
}

// GetUserInfo 按 userId 查用户(含 Roles/Dept) 供 /auth/getUserInfo 组装
func (userService *UserService) GetUserInfo(ctx context.Context, userId int64) (user system.SysUser, err error) {
	err = global.OPS_DB.WithContext(ctx).
		Preload("Roles").Preload("Dept").
		Where("id = ?", userId).
		First(&user).Error
	return
}

// GetUserDetail 组装 getUserInfo 响应所需:用户实体(含 roles/dept) + roleKey 列表 + perms 列表。
// 超管(任一角色 SuperAdmin)permissions=["*:*:*"];其余按 角色→sys_role_menu→sys_menu.perms 聚合去重。
func (userService *UserService) GetUserDetail(ctx context.Context, userId int64) (user system.SysUser, roles []string, permissions []string, err error) {
	user, err = userService.GetUserInfo(ctx, userId)
	if err != nil {
		return
	}
	roles = make([]string, 0, len(user.Roles))
	roleIds := make([]int64, 0, len(user.Roles))
	isSuper := false
	for _, r := range user.Roles {
		if r.RoleKey != "" {
			roles = append(roles, r.RoleKey)
		}
		roleIds = append(roleIds, r.RoleId)
		if r.SuperAdmin {
			isSuper = true
		}
	}
	if isSuper {
		permissions = []string{"*:*:*"}
		return
	}
	if len(roleIds) > 0 {
		err = global.OPS_DB.WithContext(ctx).Model(&system.SysMenu{}).
			Where("menu_id IN (?)",
				global.OPS_DB.Model(&system.SysRoleMenu{}).
					Where("sys_role_id IN ?", roleIds).
					Select("sys_menu_id")).
			Where("perms <> ''").
			Distinct("perms").
			Pluck("perms", &permissions).Error
		if err != nil {
			return
		}
	}
	if permissions == nil {
		permissions = []string{}
	}
	return
}

// GetUserInfoList 分页查用户列表
func (userService *UserService) GetUserInfoList(ctx context.Context, q systemReq.GetUserList) (list []system.SysUser, total int64, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&system.SysUser{})
	if q.UserName != "" {
		db = db.Where("user_name LIKE ?", "%"+q.UserName+"%")
	}
	limit, offset := q.LimitOffset()
	if limit > 0 {
		err = db.Count(&total).Limit(limit).Offset(offset).Find(&list).Error
	} else {
		err = db.Count(&total).Find(&list).Error
	}
	return
}
