package system

import "github.com/hllkk/devops-admin/server/global"

// SysRole 角色(平表,对外业务实体,字段对齐前端 Api.System.Role)
//
// 设计要点:
//   - 嵌入 OPS_AUDIT_MODEL 获取 createTime/updateTime/createBy/updateBy(对齐前端 CommonRecord)
//   - 主键 RoleId 走雪花 int64(json roleId,string); 列名 role_id(data_scope 依赖)
//   - 前端 Role 为平表(无 parentId/children),已去除角色树
//   - DataScope 为数据权限档位(json "dataScope,string" 字符串传输,前端 dataScopeRecord 对齐):1全部2本部门及子级3本部门4仅本人5自定义
//   - DeptCheckStrictly 部门树是否关联显示(数据权限自定义档部门树回显用)
//   - Flag 为内存字段(用户是否拥有该角色标识),不建列
type SysRole struct {
	global.OPS_AUDIT_MODEL
	RoleId            int64  `json:"roleId,string" gorm:"primarykey;comment:角色ID"`                                                   // 角色ID(列名 role_id)
	RoleName          string `json:"roleName" gorm:"index;comment:角色名称"`                                                             // 角色名称
	RoleKey           string `json:"roleKey" gorm:"uniqueIndex;comment:角色权限字符串"`                                                     // 角色权限字符串
	RoleSort          int    `json:"roleSort" gorm:"default:0;comment:显示顺序"`                                                         // 显示顺序
	Status            string `json:"status" gorm:"default:0;size:1;comment:角色状态 0正常1停用"`                                             // 角色状态(对齐前端 '0'/'1')
	SuperAdmin        bool   `json:"superAdmin" gorm:"default:false;comment:是否管理员"`                                                  // 是否管理员
	MenuCheckStrictly bool   `json:"menuCheckStrictly" gorm:"default:true;comment:菜单树选择项是否关联显示"`                                     // 菜单树选择项是否关联显示
	DeptCheckStrictly bool   `json:"deptCheckStrictly" gorm:"default:true;comment:部门树选择项是否关联显示"`                                     // 部门树选择项是否关联显示(数据权限自定义档部门树用)
	DataScope         int    `json:"dataScope,string" gorm:"default:1;comment:数据范围 1全部2本部门及子级3本部门4仅本人5自定义"`                          // 数据范围档位(json string 传输,引擎 datascope.go 消费)
	DefaultRouter     string `json:"defaultRouter" gorm:"default:admin;comment:默认路由(角色登录后默认打开的路由名,如 admin/disk/server/system_user)"` // 默认路由(登录入口;主角色 user.RoleId 的 DefaultRouter 决定)
	Remark            string `json:"remark" gorm:"comment:备注"`                                                                       // 备注
	Flag              bool   `json:"flag" gorm:"-"`                                                                                  // 用户是否存在此角色标识(内存组装,默认不存在)
}

func (SysRole) TableName() string {
	return "sys_roles"
}
