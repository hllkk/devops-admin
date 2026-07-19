package system

import "github.com/hllkk/devops-admin/server/global"

// SysDepartment 部门(组织架构树,对外业务实体,字段对齐前端 Api.System.Dept)
//
// 设计要点:
//   - 嵌入 OPS_AUDIT_MODEL 获取 createTime/updateTime/createBy/updateBy(对齐前端 CommonRecord)
//   - 主键 DeptId 走雪花 int64(json deptId,string); 列名 dept_id(data_scope subtreeUnion 依赖)
//   - 额外维护 Ancestors 祖级链,便于数据权限"本部门及子级"快速取子树
//   - leader 存用户ID(number),phone/email 为部门独立联系字段(不从 leader 关联带出,对齐前端)
//   - Children/NamePath 内存组装,不建列
type SysDepartment struct {
	global.OPS_AUDIT_MODEL
	DeptId       int64           `json:"deptId,string" gorm:"primarykey;comment:部门ID"`       // 部门ID(列名 dept_id)
	ParentId     int64           `json:"parentId,string" gorm:"default:0;comment:父部门ID"`     // 父部门ID(0为顶级)
	Ancestors    string          `json:"ancestors" gorm:"comment:祖级链,逗号分隔如 0,1,5"`           // 祖级链
	DeptName     string          `json:"deptName" gorm:"index;comment:部门名称"`                 // 部门名称
	DeptCategory string          `json:"deptCategory" gorm:"comment:部门类别编码"`                 // 部门类别编码
	OrderNum     int             `json:"orderNum" gorm:"default:0;comment:显示顺序"`             // 显示顺序
	Leader       int64           `json:"leader" gorm:"comment:负责人用户ID"`                      // 负责人(用户ID,对齐前端 number)
	Phone        string          `json:"phone" gorm:"comment:联系电话"`                          // 联系电话(部门独立)
	Email        string          `json:"email" gorm:"comment:邮箱"`                            // 邮箱(部门独立)
	Status       string          `json:"status" gorm:"default:0;size:1;comment:部门状态 0正常1停用"` // 部门状态(对齐前端 '0'/'1')
	Children     []SysDepartment `json:"children" gorm:"-"`                                  // 子部门(内存组装,不建列)
	NamePath     string          `json:"namePath" gorm:"-"`                                  // 公司/部门全路径名(内存组装,不建列)
}

func (SysDepartment) TableName() string {
	return "sys_departments"
}

// DeptTreeNode 部门树节点(对齐前端 Api.Common.CommonTreeRecord,岗位页左侧部门树 + 新增抽屉部门选择用)。
// id/parentId/weight 用数字序列化,对齐前端 expandedKeys=[100] 等数字 key。
type DeptTreeNode struct {
	Id       int64          `json:"id"`       // 部门ID(数字,对齐前端 NTree/NTreeSelect 的数字 key)
	ParentId int64          `json:"parentId"` // 父部门ID
	Label    string         `json:"label"`    // 部门名称
	Weight   int            `json:"weight"`   // 显示顺序(对应 SysDepartment.OrderNum)
	Children []DeptTreeNode `json:"children"` // 子部门
}
