package system

import (
	"time"

	"github.com/google/uuid"
	"github.com/hllkk/devops-admin/server/global"
)

// Login 认证链路抽象（GVA 基座遗留）。ID 仍是 uint，待登录实现时统一改 int64/string
// （见 aiDoc/frontend-backend/boundary.md 主键 ID 契约的例外说明）。
type Login interface {
	GetUsername() string
	GetNickname() string
	GetUUID() uuid.UUID
	GetUserId() uint
	GetAuthorityId() uint
	GetUserInfo() any
}

// SysUser 系统用户，对齐前端 Api.System.User。
// 主键 UserId 由雪花回调 ops:snowflake_id 自动填充；审计字段走 global.OPS_AUDIT_MODEL。
type SysUser struct {
	UserId      int64      `gorm:"column:user_id;primaryKey;autoIncrement:false" json:"userId,string"`
	DeptId      int64      `gorm:"column:dept_id;default:0" json:"deptId,string"`
	DeptName    string     `gorm:"column:dept_name;size:64;default:''" json:"deptName"`
	UserName    string     `gorm:"column:user_name;size:64;uniqueIndex" json:"userName"`
	NickName    string     `gorm:"column:nick_name;size:64" json:"nickName"`
	UserType    string     `gorm:"column:user_type;size:16;default:'00'" json:"userType"`
	Email       string     `gorm:"column:email;size:128;default:''" json:"email"`
	Phonenumber string     `gorm:"column:phonenumber;size:20;default:''" json:"phonenumber"`
	Sex         string     `gorm:"column:sex;size:1;default:'2'" json:"sex"`
	Avatar      string     `gorm:"column:avatar;size:512;default:''" json:"avatar"`
	Password    string     `gorm:"column:password;size:128" json:"-"`
	Status      string     `gorm:"column:status;size:1;default:'0'" json:"status"`
	LoginIp     string     `gorm:"column:login_ip;size:128;default:''" json:"loginIp"`
	LoginDate   *time.Time `gorm:"column:login_date" json:"loginDate"`
	Remark      string     `gorm:"column:remark;size:500;default:''" json:"remark"`
	global.OPS_AUDIT_MODEL
}

// TableName 自定义表名 sys_user
func (SysUser) TableName() string { return "sys_user" }
