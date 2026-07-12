package system

// SysRole 系统角色，对齐前端 Api.System.Role。
// 超管角色由初始化流程以显式 RoleId=1 种子（雪花回调只在主键为 0 时填，不覆盖）。
type SysRole struct {
	RoleId            int64  `gorm:"column:role_id;primaryKey;autoIncrement:false" json:"roleId,string"`
	RoleName          string `gorm:"column:role_name;size:64" json:"roleName"`
	RoleKey           string `gorm:"column:role_key;size:100" json:"roleKey"`
	RoleSort          int    `gorm:"column:role_sort;default:0" json:"roleSort"`
	MenuCheckStrictly bool   `gorm:"column:menu_check_strictly;default:false" json:"menuCheckStrictly"`
	Status            string `gorm:"column:status;size:1;default:'0'" json:"status"`
	SuperAdmin        bool   `gorm:"column:super_admin;default:false" json:"superAdmin"`
	Remark            string `gorm:"column:remark;size:500;default:''" json:"remark"`
	Flag              bool   `gorm:"-" json:"flag"` // 瞬态：当前用户是否拥有此角色，不入库
	AuditModel
}

// TableName 自定义表名 sys_role
func (SysRole) TableName() string { return "sys_role" }
