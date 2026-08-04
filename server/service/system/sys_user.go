package system

import (
	"context"
	"errors"
	"time"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/utils"
	"gorm.io/gorm"
)

// dummyHash 用户名不存在时跑一次等价 bcrypt,抹平与"密码错误"分支的响应时序差异,消除用户名枚举侧信道。
// 包加载时计算一次(DefaultCost≈100ms),不进入请求路径,不影响运行期性能。
var dummyHash = utils.BcryptHash("dummy-timing-equalization")

type UserService struct{}

// Login 校验用户名密码 返回带 Roles 的用户(登录链路 claims 需要 SuperAdmin)
func (userService *UserService) Login(ctx context.Context, u *system.SysUser) (user system.SysUser, err error) {
	err = global.OPS_DB.WithContext(ctx).
		Preload("Roles", "status = ?", "0"). // 仅加载启用角色:停用角色不进 claims/perms,使"停用角色"真正收回权限
		Where("user_name = ?", u.UserName).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 用户名不存在也跑一次等价 bcrypt,抹平与"密码错误"分支的响应时序,消除用户名枚举侧信道
			_ = utils.BcryptCheck(u.Password, dummyHash)
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
		Preload("Roles", "status = ?", "0").Preload("Dept"). // 与 Login 对齐:仅启用角色参与 getUserInfo 组装
		Where("id = ?", userId).
		First(&user).Error
	return
}

// validAppModules 业务模块标识固定顺序(与前端 ALL_MODULES 对齐);新增模块在此追加一行。
var validAppModules = []string{"admin", "server", "gateway"}

// filterAppModules 仅保留合法模块并按固定顺序输出(剔除历史脏值/非法 module)。
func filterAppModules(modules []string) []string {
	seen := make(map[string]bool, len(modules))
	for _, m := range modules {
		seen[m] = true
	}
	apps := make([]string, 0, len(validAppModules))
	for _, m := range validAppModules {
		if seen[m] {
			apps = append(apps, m)
		}
	}
	return apps
}

// GetUserDetail 组装 getUserInfo 响应所需:用户实体(含 roles/dept) + roleKey 列表 + perms 列表 + 可见应用(apps)。
// 超管(任一角色 SuperAdmin)permissions=["*:*:*"]、apps=全部模块;其余按 角色→sys_role_menu→sys_menu 聚合:
// perms 去重;apps 按 module 去重(该模块下有任意一个授权菜单即视为有该模块权限)。
func (userService *UserService) GetUserDetail(ctx context.Context, userId int64) (user system.SysUser, roles []string, permissions []string, apps []string, defaultRouter string, err error) {
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
		// 主角色(user.RoleId)的默认路由作为登录入口
		if r.RoleId == user.RoleId && r.DefaultRouter != "" {
			defaultRouter = r.DefaultRouter
		}
	}
	if isSuper {
		permissions = []string{"*:*:*"}
		apps = append([]string{}, validAppModules...) // 超管可见全部模块
		return
	}
	if len(roleIds) > 0 {
		// 仅聚合启用菜单(status='0')的按钮权限:停用菜单的 perms 不应再下发给前端按钮显隐/鉴权
		err = global.OPS_DB.WithContext(ctx).Model(&system.SysMenu{}).
			Where("menu_id IN (?)",
				global.OPS_DB.Model(&system.SysRoleMenu{}).
					Where("sys_role_id IN ?", roleIds).
					Select("sys_menu_id")).
			Where("status = ?", "0").
			Where("perms <> ''").
			Distinct("perms").
			Pluck("perms", &permissions).Error
		if err != nil {
			return
		}
		// 可见应用(APP):复用同一角色→菜单授权关系,按 module 去重;该模块下有任意授权菜单即算有权限
		var modules []string
		err = global.OPS_DB.WithContext(ctx).Model(&system.SysMenu{}).
			Where("menu_id IN (?)",
				global.OPS_DB.Model(&system.SysRoleMenu{}).
					Where("sys_role_id IN ?", roleIds).
					Select("sys_menu_id")).
			Where("status = ?", "0").
			Where("module <> ''").
			Distinct("module").
			Pluck("module", &modules).Error
		if err != nil {
			return
		}
		apps = filterAppModules(modules)
	}
	if permissions == nil {
		permissions = []string{}
	}
	if apps == nil {
		apps = []string{}
	}
	return
}
