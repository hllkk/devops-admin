package system

import (
	"context"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/utils/datascope"
)

type DataScopeService struct{}

var DataScopeServiceApp = new(DataScopeService)

// BuildIdentity 依据用户与角色, 构建一次请求的数据权限身份。
// 所有查询显式 data_scope:skip 旁路, 避免回调递归; 子树用内存 BFS 计算, 免方言差异。
func (s *DataScopeService) BuildIdentity(ctx context.Context, userID, roleID int64) (*datascope.Identity, error) {
	id := &datascope.Identity{UserID: userID, RoleID: roleID}

	// 1) 角色数据范围
	var auth system.SysRole
	err := global.OPS_DB.WithContext(ctx).Set("data_scope:skip", true).
		Select("role_id", "data_scope").
		Where("role_id = ?", roleID).First(&auth).Error
	if err == nil && auth.DataScope != 0 {
		id.Scope = auth.DataScope
	} else {
		id.Scope = datascope.ScopeAll // 查不到或未配置 → 默认全部(与列默认一致, 安全不破坏)
	}

	// 2) 用户主部门
	var user system.SysUser
	if e := global.OPS_DB.WithContext(ctx).Set("data_scope:skip", true).
		Select("id", "dept_id").Where("id = ?", userID).First(&user).Error; e == nil {
		id.DeptID = user.DeptId
	}

	// 3) 用户全部归属部门(多部门)
	var deptIDs []int64
	global.OPS_DB.WithContext(ctx).Set("data_scope:skip", true).
		Model(&system.SysUserDepartment{}).
		Where("sys_user_id = ?", userID).
		Pluck("sys_department_id", &deptIDs)

	roots := make(map[int64]struct{}, len(deptIDs)+1)
	for _, d := range deptIDs {
		roots[d] = struct{}{}
	}
	if id.DeptID != 0 {
		roots[id.DeptID] = struct{}{}
	}
	for d := range roots {
		id.DeptIDs = append(id.DeptIDs, d) // 直接归属部门集("本部门"档用)
	}

	// 4) 可见部门 = 所有归属部门子树的并集("本部门及以下"档用)
	id.VisibleDeptIDs = s.subtreeUnion(ctx, roots)

	// 5) 自定义部门集(角色配置, 直接集合不展开子树)
	if id.Scope == datascope.ScopeCustom {
		global.OPS_DB.WithContext(ctx).Set("data_scope:skip", true).
			Model(&system.SysRoleDepartment{}).
			Where("sys_role_id = ?", roleID).
			Pluck("sys_department_id", &id.CustomDeptIDs)
	}
	return id, nil
}

// subtreeUnion 内存 BFS 计算多个部门子树的并集(含自身)。部门量级小, 一次全量加载即可。
func (s *DataScopeService) subtreeUnion(ctx context.Context, roots map[int64]struct{}) []int64 {
	if len(roots) == 0 {
		return nil
	}
	var all []system.SysDepartment
	global.OPS_DB.WithContext(ctx).Set("data_scope:skip", true).
		Select("dept_id", "parent_id").Find(&all)

	children := make(map[int64][]int64, len(all))
	for _, d := range all {
		children[d.ParentId] = append(children[d.ParentId], d.DeptId)
	}

	visited := make(map[int64]struct{}, len(roots))
	queue := make([]int64, 0, len(roots))
	for r := range roots {
		queue = append(queue, r)
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if _, ok := visited[cur]; ok {
			continue
		}
		visited[cur] = struct{}{}
		for _, c := range children[cur] {
			if _, ok := visited[c]; !ok {
				queue = append(queue, c)
			}
		}
	}

	res := make([]int64, 0, len(visited))
	for id := range visited {
		res = append(res, id)
	}
	return res
}
