package system

import (
	"context"
	"errors"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
)

// MenuService 菜单业务服务(对齐前端 /system/menu/* 资源)
type MenuService struct{}

// menuOrder 菜单统一排序:order_num 升序,同序按 menu_id 升序。
const menuOrder = "order_num ASC, menu_id ASC"

// GetMenuList 查菜单列表(对齐前端 GET /system/menu/list)。
// 菜单为树形,列表不分页,全量返回平表由前端组装树;menuName 模糊,status/menuType/parentId 精确。
func (s *MenuService) GetMenuList(ctx context.Context, q systemReq.MenuSearch) (list []system.SysMenu, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&system.SysMenu{})
	if q.MenuName != "" {
		db = db.Where("menu_name LIKE ?", "%"+q.MenuName+"%")
	}
	if q.Status != "" {
		db = db.Where("status = ?", q.Status)
	}
	if q.MenuType != "" {
		db = db.Where("menu_type = ?", q.MenuType)
	}
	if q.ParentId > 0 {
		db = db.Where("parent_id = ?", q.ParentId)
	}
	err = db.Order(menuOrder).Find(&list).Error
	return
}

// CreateMenu 新增菜单;menuName 必填,createBy 填审计字段。
func (s *MenuService) CreateMenu(ctx context.Context, req systemReq.MenuOperateParams, createBy int64) error {
	if req.MenuName == "" {
		return errors.New("菜单名称不能为空")
	}
	m := system.SysMenu{
		ParentId:   req.ParentId.Int64(),
		MenuType:   req.MenuType,
		MenuName:   req.MenuName,
		OrderNum:   req.OrderNum,
		Path:       req.Path,
		Component:  req.Component,
		QueryParam: req.QueryParam,
		IsFrame:    req.IsFrame,
		IsCache:    req.IsCache,
		Visible:    req.Visible,
		Status:     req.Status,
		Perms:      req.Perms,
		Icon:       req.Icon,
		Remark:     req.Remark,
	}
	// CreateBy/UpdateBy 为内嵌 OPS_AUDIT_MODEL 的提升字段,struct literal 中不可直接命名,改用赋值写入。
	m.CreateBy = createBy
	m.UpdateBy = createBy
	return global.OPS_DB.WithContext(ctx).Create(&m).Error
}

// UpdateMenu 修改菜单;menuId 必填,updateBy 填审计字段。
func (s *MenuService) UpdateMenu(ctx context.Context, req systemReq.MenuOperateParams, updateBy int64) error {
	menuId := req.MenuId.Int64()
	if menuId == 0 {
		return errors.New("菜单ID不能为空")
	}
	if req.MenuName == "" {
		return errors.New("菜单名称不能为空")
	}
	return global.OPS_DB.WithContext(ctx).Model(&system.SysMenu{}).Where("menu_id = ?", menuId).
		Updates(map[string]interface{}{
			"parent_id":   req.ParentId.Int64(),
			"menu_type":   req.MenuType,
			"menu_name":   req.MenuName,
			"order_num":   req.OrderNum,
			"path":        req.Path,
			"component":   req.Component,
			"query_param": req.QueryParam,
			"is_frame":    req.IsFrame,
			"is_cache":    req.IsCache,
			"visible":     req.Visible,
			"status":      req.Status,
			"perms":       req.Perms,
			"icon":        req.Icon,
			"remark":      req.Remark,
			"update_by":   updateBy,
		}).Error
}

// DeleteMenu 删除单个菜单;存在子菜单或被角色引用(sys_role_menu)时禁止删除(对齐 RuoYi)。
func (s *MenuService) DeleteMenu(ctx context.Context, menuId int64) error {
	if menuId == 0 {
		return errors.New("菜单ID不能为空")
	}
	var childCnt int64
	if err := global.OPS_DB.WithContext(ctx).Model(&system.SysMenu{}).
		Where("parent_id = ?", menuId).Count(&childCnt).Error; err != nil {
		return err
	}
	if childCnt > 0 {
		return errors.New("存在子菜单,不允许删除")
	}
	var roleCnt int64
	if err := global.OPS_DB.WithContext(ctx).Model(&system.SysRoleMenu{}).
		Where("sys_menu_id = ?", menuId).Count(&roleCnt).Error; err != nil {
		return err
	}
	if roleCnt > 0 {
		return errors.New("菜单已分配给角色,不允许删除")
	}
	return global.OPS_DB.WithContext(ctx).Where("menu_id = ?", menuId).Delete(&system.SysMenu{}).Error
}

// buildMenuTreeSelect 将菜单平表按 parent_id 组装成树并映射为 MenuTreeSelectNode(仅树选择所需字段)。
// 输入需已按展示顺序排序(menuOrder);parent_id=0 为顶级。O(n),用 map 索引子节点避免嵌套查询。
func (s *MenuService) buildMenuTreeSelect(all []system.SysMenu) []system.MenuTreeSelectNode {
	childrenOf := make(map[int64][]system.SysMenu, len(all))
	for _, m := range all {
		childrenOf[m.ParentId] = append(childrenOf[m.ParentId], m)
	}
	var attach func([]system.SysMenu) []system.MenuTreeSelectNode
	attach = func(nodes []system.SysMenu) []system.MenuTreeSelectNode {
		out := make([]system.MenuTreeSelectNode, 0, len(nodes))
		for _, n := range nodes {
			node := system.MenuTreeSelectNode{
				Id:       n.MenuId,
				Label:    n.MenuName,
				MenuType: n.MenuType,
				Icon:     n.Icon,
				Visible:  n.Visible,
				Status:   n.Status,
			}
			if kids := childrenOf[n.MenuId]; len(kids) > 0 {
				node.Children = attach(kids)
			}
			out = append(out, node)
		}
		return out
	}
	return attach(childrenOf[0])
}

// GetMenuTreeSelect 全量菜单树(对齐前端 GET /system/menu/treeselect,选父级/树选择用)。
// 返回已组装的 MenuTreeSelectNode 树(精简字段),前端 NTree 直接渲染。
func (s *MenuService) GetMenuTreeSelect(ctx context.Context) (list []system.MenuTreeSelectNode, err error) {
	var menus []system.SysMenu
	if err = global.OPS_DB.WithContext(ctx).Order(menuOrder).Find(&menus).Error; err != nil {
		return
	}
	list = s.buildMenuTreeSelect(menus)
	return
}

// GetRoleMenuTreeSelect 角色菜单权限树(对齐前端 GET /system/menu/roleMenuTreeselect/{roleId})。
// menus=全部菜单的 MenuTreeSelectNode 树(精简字段,后端组装);checkedKeys=该角色已分配菜单的叶子节点 ID(NTree cascade 回显用,对齐 RuoYi)。
func (s *MenuService) GetRoleMenuTreeSelect(ctx context.Context, roleId int64) (result system.RoleMenuTreeSelect, err error) {
	var menus []system.SysMenu
	if err = global.OPS_DB.WithContext(ctx).Order(menuOrder).Find(&menus).Error; err != nil {
		return
	}
	result.Menus = s.buildMenuTreeSelect(menus)
	var roleMenuIds []int64
	if err = global.OPS_DB.WithContext(ctx).Model(&system.SysRoleMenu{}).
		Where("sys_role_id = ?", roleId).Pluck("sys_menu_id", &roleMenuIds).Error; err != nil {
		return
	}
	result.CheckedKeys = s.leafCheckedKeys(ctx, roleMenuIds)
	return
}

// CascadeDeleteMenu 级联删除:收集选中菜单及其全部子孙(按 parent_id 递归),删菜单 + 清理 sys_role_menu 关联。
func (s *MenuService) CascadeDeleteMenu(ctx context.Context, menuIds []int64) error {
	if len(menuIds) == 0 {
		return errors.New("未选择删除项")
	}
	all := s.collectWithDescendants(ctx, menuIds)
	if err := global.OPS_DB.WithContext(ctx).Where("sys_menu_id IN ?", all).Delete(&system.SysRoleMenu{}).Error; err != nil {
		return err
	}
	return global.OPS_DB.WithContext(ctx).Where("menu_id IN ?", all).Delete(&system.SysMenu{}).Error
}

// leafCheckedKeys 取角色已分配菜单的叶子 ID(角色菜单中,没有子菜单也属于角色菜单的节点)。
// 对齐 RuoYi:供 NTree cascade 回显(父级自动半选/全选,只需回显最深层)。
func (s *MenuService) leafCheckedKeys(ctx context.Context, roleMenuIds []int64) []int64 {
	if len(roleMenuIds) == 0 {
		return []int64{}
	}
	// 角色菜单中"作为父"的(有子菜单也属于角色菜单):parent_id IN roleIds AND menu_id IN roleIds
	var parentsWithChild []int64
	global.OPS_DB.WithContext(ctx).Model(&system.SysMenu{}).
		Where("parent_id IN ? AND menu_id IN ?", roleMenuIds, roleMenuIds).
		Distinct("parent_id").Pluck("parent_id", &parentsWithChild)
	parentSet := make(map[int64]bool, len(parentsWithChild))
	for _, p := range parentsWithChild {
		parentSet[p] = true
	}
	leaves := make([]int64, 0, len(roleMenuIds))
	for _, id := range roleMenuIds {
		if !parentSet[id] {
			leaves = append(leaves, id)
		}
	}
	return leaves
}

// collectWithDescendants 按 parent_id 递归收集 ids 及其全部子孙 ID(菜单无 ancestors 字段,只能递归)。
func (s *MenuService) collectWithDescendants(ctx context.Context, ids []int64) []int64 {
	all := make(map[int64]bool, len(ids))
	for _, id := range ids {
		all[id] = true
	}
	current := ids
	for len(current) > 0 {
		var children []int64
		global.OPS_DB.WithContext(ctx).Model(&system.SysMenu{}).
			Where("parent_id IN ?", current).Pluck("menu_id", &children)
		next := make([]int64, 0, len(children))
		for _, c := range children {
			if !all[c] {
				all[c] = true
				next = append(next, c)
			}
		}
		current = next
	}
	result := make([]int64, 0, len(all))
	for id := range all {
		result = append(result, id)
	}
	return result
}
