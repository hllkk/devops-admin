package system

import (
	"time"

	"github.com/google/uuid"
	"github.com/hllkk/devops-admin/server/global"
)

// Login 登录链路所需的最小用户视图(JWT claims 构建消费)
type Login interface {
	GetUsername() string
	GetNickname() string
	GetUUID() uuid.UUID
	GetUserId() int64
	GetRoleId() int64
	GetSuperAdmin() bool
	GetUserInfo() any
}

// SysUser 用户(对外业务实体,字段对齐前端 Api.System.User / RuoYi 契约)
//
// 设计要点:
//   - 嵌入 OPS_AUDIT_MODEL 获取 createTime/updateTime/createBy/updateBy(对齐前端 CommonRecord)
//   - 主键 UserId(DB 列复用 id,雪花 int64,json userId,string 对齐前端 IdType)
//   - 全部 ID 字段统一 int64:UserId/DeptId/RoleId 与雪花主键一致,无 uint 特例
//   - 字段命名严格对齐前端:userName/nickName/phonenumber(非 phone)/avatar(非 headerImg)/status('0'|'1')
//   - DeptId 为数据权限身份构建字段(service/system/data_scope.go 消费),json deptId 对齐前端
//   - UUID 供登录链路 claims 使用; RoleId 为主角色(登录链路 claims 用,前端不输出,故 json:"-")
//   - 多角色/多部门/多岗位走显式连接表 sys_user_role / sys_user_departments / sys_user_post
type SysUser struct {
	global.OPS_AUDIT_MODEL
	UserId      int64      `gorm:"primarykey;column:id;comment:用户ID" json:"userId,string"`                          // 用户ID(DB列复用id,雪花int64)
	DeptId      int64      `json:"deptId,string" gorm:"comment:主部门ID(数据归属/盖章)"`                                     // 主部门ID
	DeptName    string     `json:"deptName" gorm:"-"`                                                               // 部门名称(内存组装,列表展示)
	UserName    string     `json:"userName" gorm:"index;comment:用户登录名"`                                             // 用户登录名
	NickName    string     `json:"nickName" gorm:"default:系统用户;comment:用户昵称"`                                       // 用户昵称
	UserType    string     `json:"userType" gorm:"default:sys_user;size:32;comment:用户类型(sys_user系统用户)"`             // 用户类型
	Email       string     `json:"email" gorm:"comment:用户邮箱"`                                                       // 用户邮箱
	Phonenumber string     `json:"phonenumber" gorm:"index;comment:手机号"`                                            // 手机号(对齐前端 phonenumber)
	Sex         string     `json:"sex" gorm:"default:0;size:1;comment:性别 0男1女2未知"`                                  // 性别
	Avatar      string     `json:"avatar" gorm:"default:https://qmplusimg.henrongyi.top/gva_header.jpg;comment:头像"` // 头像
	Password    string     `json:"-" gorm:"comment:用户登录密码"`                                                         // 密码(不输出)
	PasswordUpdatedAt *time.Time `json:"passwordUpdatedAt,omitempty" gorm:"comment:密码最后修改时间"`                        // 密码最后修改时间(密码过期判定)
	Status      string     `json:"status" gorm:"default:0;size:1;comment:帐号状态 0正常1停用"`                              // 帐号状态(对齐前端 '0'/'1')
	LoginIp     string     `json:"loginIp" gorm:"comment:最后登录IP"`                                                   // 最后登录IP
	LoginDate   *time.Time `json:"loginDate" gorm:"comment:最后登录时间"`                                                 // 最后登录时间
	Remark      string     `json:"remark" gorm:"comment:备注"`                                                        // 备注
	UUID        uuid.UUID  `json:"uuid" gorm:"index;comment:用户UUID"`                                                // 用户UUID(登录链路)
	// 关联(多角色/多部门/多岗位走显式连接表)
	RoleId      int64           `json:"-" gorm:"default:888;comment:用户主角色ID(登录链路claims用,前端不输出)"` // 主角色ID
	Roles []SysRole `json:"roles" gorm:"many2many:sys_user_role;joinForeignKey:SysUserId;joinReferences:SysRoleId"`               // 多角色(join 列对齐 sys_user_id/sys_role_id)
	Dept        SysDepartment   `json:"dept" form:"-" gorm:"foreignKey:DeptId;comment:主部门"`                                    // 主部门;form:"-" 防御 gin 绑定递归
	Departments []SysDepartment `json:"departments" gorm:"many2many:sys_user_departments;joinForeignKey:SysUserId;joinReferences:SysDepartmentId"` // 多部门归属(数据可见范围)
	Posts       []SysPost       `json:"posts" gorm:"many2many:sys_user_post;joinForeignKey:SysUserId;joinReferences:SysPostId"` // 多岗位
}

func (SysUser) TableName() string {
	return "sys_users"
}

// UserInfo 用户详情响应(对齐前端 Api.System.UserInfo)。
// postIds/roleIds 用 []string 对齐前端 string[](NSelect/PostSelect 回显需与 Role.roleId 字符串匹配)。
type UserInfo struct {
	PostIds []string  `json:"postIds"` // 用户岗位 ID 列表(字符串)
	RoleIds []string  `json:"roleIds"` // 用户角色 ID 列表(字符串)
	Roles   []SysRole `json:"roles"`   // 用户角色列表(含 roleName/roleId 供下拉)
}

func (s *SysUser) GetUsername() string {
	return s.UserName
}

func (s *SysUser) GetNickname() string {
	return s.NickName
}

func (s *SysUser) GetUUID() uuid.UUID {
	return s.UUID
}

func (s *SysUser) GetUserId() int64 {
	return s.UserId
}

func (s *SysUser) GetRoleId() int64 {
	return s.RoleId
}

// GetSuperAdmin 是否拥有超管角色。
// 依赖登录链路 Preload Roles;任一关联角色 SuperAdmin=true 即视为超管,
// 供 JWT claims 携带,CasbinHandler 据此绕过策略校验。
func (s *SysUser) GetSuperAdmin() bool {
	for _, r := range s.Roles {
		if r.SuperAdmin {
			return true
		}
	}
	return false
}

func (s *SysUser) GetUserInfo() any {
	return *s
}
