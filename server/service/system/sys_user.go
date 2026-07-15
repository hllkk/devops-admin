package system

import (
	"errors"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/model/system/request"
	"github.com/hllkk/devops-admin/server/utils"
)

// UserService 用户服务。对照 GVA service/system/sys_user.go 的 Login / GetUserInfo，
// 适配当前项目：角色用 sys_role（含 super_admin），返回结构对齐 SoybeanAdmin 前端契约。
// dummyBcryptHash 启动时生成的固定 bcrypt 哈希，用于用户不存在时消耗等量校验时间，防御用户名枚举（时序攻击）。
var dummyBcryptHash = utils.BcryptHash("dummy-password-for-timing-equalization")

type UserService struct{}

// Login 校验用户名密码，签发 access + refresh 双 token（httpOnly cookie 由 API 层写入）。
func (s *UserService) Login(username, password string) (accessToken, refreshToken string, user system.SysUser, err error) {
	if err = global.OPS_DB.Where("user_name = ?", username).First(&user).Error; err != nil {
		// 用户不存在（或 DB 异常）：执行一次假 bcrypt，使响应耗时与“密码错误”接近，避免用户名枚举（时序攻击）
		_ = utils.BcryptCheck(password, dummyBcryptHash)
		return "", "", system.SysUser{}, errors.New("用户不存在或密码错误")
	}
	if !utils.BcryptCheck(password, user.Password) {
		return "", "", system.SysUser{}, errors.New("用户不存在或密码错误")
	}
	if user.Status != "0" {
		// 身份已验证才告知停用，不向未认证者泄露账号存在性
		return "", "", system.SysUser{}, errors.New("账号已停用")
	}
	_, isSuper, firstRoleId := s.getUserRoleIds(int64(user.UserId))
	bc := request.BaseClaims{
		ID:         uint(user.UserId),
		Username:   user.UserName,
		NickName:   user.NickName,
		RoleId:     firstRoleId,
		SuperAdmin: isSuper,
	}
	j := utils.NewJWT()
	if accessToken, err = j.CreateAccessToken(bc); err != nil {
		return "", "", system.SysUser{}, err
	}
	if refreshToken, err = j.CreateRefreshToken(bc); err != nil {
		return "", "", system.SysUser{}, err
	}
	return accessToken, refreshToken, user, nil
}

// getUserRoleIds 返回用户角色 id 列表、是否超管、首个角色 id（uint，供 BaseClaims.RoleId）。
func (s *UserService) getUserRoleIds(userId int64) (roleIds []int64, isSuper bool, firstRoleId uint) {
	global.OPS_DB.Model(&system.SysUserRole{}).Where("user_id = ?", userId).Pluck("role_id", &roleIds)
	if len(roleIds) > 0 {
		firstRoleId = uint(roleIds[0])
		var roles []system.SysRole
		global.OPS_DB.Where("role_id IN ?", roleIds).Find(&roles)
		for _, r := range roles {
			if r.SuperAdmin {
				isSuper = true
			}
		}
	}
	return
}

// GetUserInfo 聚合用户信息，对齐 SoybeanAdmin UserInfo{user, roles, permissions}。
// 对照 GVA GetUserInfo（Preload Authorities），适配：
//   - roles: 启用角色 Role[]（前端 UserInfo.user.roles）
//   - roleKeys: 启用角色 roleKey[]（前端 UserInfo.roles）
//   - permissions: 超管返回 ["*:*:*"]；否则取其角色关联的 C/F 菜单 perms 去重
func (s *UserService) GetUserInfo(userId int64) (user system.SysUser, roles []system.SysRole, roleKeys []string, perms []string, err error) {
	if err = global.OPS_DB.Where("user_id = ?", userId).First(&user).Error; err != nil {
		return system.SysUser{}, nil, nil, nil, err
	}
	roleIds, isSuper, _ := s.getUserRoleIds(userId)

	roles = []system.SysRole{}
	if len(roleIds) > 0 {
		global.OPS_DB.Where("role_id IN ? AND status = ?", roleIds, "0").Find(&roles)
	}
	roleKeys = make([]string, 0, len(roles))
	for _, r := range roles {
		roleKeys = append(roleKeys, r.RoleKey)
	}

	if isSuper {
		perms = []string{"*:*:*"}
		return
	}
	perms = []string{}
	if len(roleIds) == 0 {
		return
	}
	var menuIds []int64
	global.OPS_DB.Model(&system.SysRoleMenu{}).Where("role_id IN ?", roleIds).Pluck("menu_id", &menuIds)
	if len(menuIds) == 0 {
		return
	}
	var menus []system.SysMenu
	global.OPS_DB.Where("menu_id IN ? AND menu_type IN ? AND status = ?", menuIds, []string{"C", "F"}, "0").Find(&menus)
	seen := map[string]struct{}{}
	for _, m := range menus {
		if m.Perms == "" {
			continue
		}
		if _, ok := seen[m.Perms]; ok {
			continue
		}
		seen[m.Perms] = struct{}{}
		perms = append(perms, m.Perms)
	}
	return
}
