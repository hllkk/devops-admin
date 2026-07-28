package system

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
	"gorm.io/gorm"
)

// DepartmentService 部门业务服务(对齐前端 /system/dept/* 资源)
type DepartmentService struct{}

// deptOrder 部门统一排序:order_num 升序,同序按 dept_id 升序。
const deptOrder = "order_num ASC, dept_id ASC"

// GetDeptList 查部门列表(对齐前端 GET /system/dept/list)。
// 部门为树形,列表不分页,全量返回平表由前端组装树;deptName 模糊,status 精确。
func (s *DepartmentService) GetDeptList(ctx context.Context, q systemReq.DeptSearch) (list []system.SysDepartment, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&system.SysDepartment{})
	if q.DeptName != "" {
		db = db.Where("dept_name LIKE ?", "%"+q.DeptName+"%")
	}
	if q.Status != "" {
		db = db.Where("status = ?", q.Status)
	}
	err = db.Order(deptOrder).Find(&list).Error
	return
}

// GetExcludeDeptList 查排除指定部门及其子部门的列表(对齐前端 GET /system/dept/list/exclude/{deptId})。
// 编辑部门选父级时用,避免把自己/子部门设为父级造成环。deptId 无效(0 或不存在)时返回全部。
func (s *DepartmentService) GetExcludeDeptList(ctx context.Context, deptId int64) (list []system.SysDepartment, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&system.SysDepartment{})
	if deptId > 0 {
		var d system.SysDepartment
		// First 必须用独立 tx,不能复用 db: GORM 在 clone==0 的 db 上链式调用共享同一 Statement,
		// 复用会让 First 残留的 RaiseErrorOnNotFound/查询后 db.Error(NotFound)/WHERE/LIMIT 污染后续 Find
		// ——Query 回调 if db.Error==nil 会短路, 或残留 RaiseErrorOnNotFound 叠加矛盾条件误报 NotFound,
		// 导致选父级接口在 deptId 存在/不存在两种情况下都报"获取失败"。
		if e := global.OPS_DB.WithContext(ctx).Where("dept_id = ?", deptId).First(&d).Error; e != nil {
			if errors.Is(e, gorm.ErrRecordNotFound) {
				return list, db.Order(deptOrder).Find(&list).Error
			}
			return nil, e
		}
		// fullChain = 该部门完整祖级链(含自己);子孙的 ancestors 等于它或以它+", "开头
		fullChain := d.Ancestors + "," + strconv.FormatInt(deptId, 10)
		db = db.Where("dept_id <> ?", deptId).
			Where("ancestors <> ? AND ancestors NOT LIKE ?", fullChain, fullChain+",%")
	}
	err = db.Order(deptOrder).Find(&list).Error
	return
}

// CreateDept 新增部门;ancestors = 父.ancestors + "," + 父.deptId(顶层 parentId<=0 → "0"),
// 同父级下 deptName 唯一校验(对齐 RuoYi checkDeptNameUnique),createBy 填审计字段。
func (s *DepartmentService) CreateDept(ctx context.Context, req systemReq.DeptOperateParams, createBy int64) error {
	if req.DeptName == "" {
		return errors.New("部门名称不能为空")
	}
	parentId := req.ParentId.Int64()
	var ancestors string
	if parentId > 0 {
		var parent system.SysDepartment
		if err := global.OPS_DB.WithContext(ctx).Where("dept_id = ?", parentId).First(&parent).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("父部门不存在")
			}
			return err
		}
		ancestors = parent.Ancestors + "," + strconv.FormatInt(parentId, 10)
	} else {
		parentId = 0
		ancestors = "0"
	}
	if err := s.checkDeptNameUnique(ctx, req.DeptName, parentId, 0); err != nil {
		return err
	}
	d := system.SysDepartment{
		ParentId:     parentId,
		Ancestors:    ancestors,
		DeptName:     req.DeptName,
		DeptCategory: req.DeptCategory,
		OrderNum:     req.OrderNum,
		Leader:       req.Leader,
		Phone:        req.Phone,
		Email:        req.Email,
		Status:       req.Status,
	}
	// CreateBy/UpdateBy 为内嵌 OPS_AUDIT_MODEL 的提升字段,struct literal 中不可直接命名,改用赋值写入。
	d.CreateBy = createBy
	d.UpdateBy = createBy
	return global.OPS_DB.WithContext(ctx).Create(&d).Error
}

// UpdateDept 修改部门;parentId 必填且不可为自身/子孙(防环),同父级下 deptName 唯一(排除自身),
// parentId 变更时同步本部门及所有子孙的 ancestors(前缀替换,对齐 RuoYi updateDept)。
func (s *DepartmentService) UpdateDept(ctx context.Context, req systemReq.DeptOperateParams, updateBy int64) error {
	deptId := req.DeptId.Int64()
	if deptId == 0 {
		return errors.New("部门ID不能为空")
	}
	if req.DeptName == "" {
		return errors.New("部门名称不能为空")
	}
	parentId := req.ParentId.Int64()
	if parentId == 0 {
		return errors.New("父部门不能为空")
	}
	if parentId == deptId {
		return errors.New("父部门不能为自身")
	}
	var old system.SysDepartment
	if err := global.OPS_DB.WithContext(ctx).Where("dept_id = ?", deptId).First(&old).Error; err != nil {
		return err
	}
	// 防环:新父级不能是自身或自身的子孙(前端 exclude 已过滤,后端兜底)
	if s.isDescendant(ctx, deptId, parentId) {
		return errors.New("父部门不能为自身或子部门")
	}
	var parent system.SysDepartment
	if err := global.OPS_DB.WithContext(ctx).Where("dept_id = ?", parentId).First(&parent).Error; err != nil {
		return errors.New("父部门不存在")
	}
	if err := s.checkDeptNameUnique(ctx, req.DeptName, parentId, deptId); err != nil {
		return err
	}
	newAncestors := parent.Ancestors + "," + strconv.FormatInt(parentId, 10)
	oldFullChain := old.Ancestors + "," + strconv.FormatInt(deptId, 10)
	newFullChain := newAncestors + "," + strconv.FormatInt(deptId, 10)
	// 主部门更新与子孙 ancestors 同步须原子:中途失败会留下不一致的祖级链(exclude 树错位、
	// 后续移动漏改子孙),故整体走事务。子孙同步用一条 SQL 做前缀替换(旧完整链→新完整链),
	// 免去逐条 UPDATE 的 N+1(大部门树下移动高层部门可达成百上千次 UPDATE)。
	if err := global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&system.SysDepartment{}).Where("dept_id = ?", deptId).
			Updates(map[string]interface{}{
				"parent_id":     parentId,
				"ancestors":     newAncestors,
				"dept_name":     req.DeptName,
				"dept_category": req.DeptCategory,
				"order_num":     req.OrderNum,
				"leader":        req.Leader,
				"phone":         req.Phone,
				"email":         req.Email,
				"status":        req.Status,
				"update_by":     updateBy,
			}).Error; err != nil {
			return err
		}
		// 父级变更时同步子孙 ancestors:子孙 = ancestors 等于旧完整链(直接子)或以其+", "开头(更深子);
		// 尾段 = SUBSTRING(ancestors, len(oldFullChain)+1)(SQL 1-based),前缀拼新完整链。
		if oldFullChain != newFullChain {
			return tx.Model(&system.SysDepartment{}).
				Where("ancestors = ? OR ancestors LIKE ?", oldFullChain, oldFullChain+",%").
				Update("ancestors", gorm.Expr("CONCAT(?, SUBSTRING(ancestors, ?))", newFullChain, len(oldFullChain)+1)).Error
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// DeleteDept 批量删除部门;存在子部门 / 用户(主部门或多部门关联) / 岗位 / 角色数据权限引用时禁止删除(对齐 RuoYi)。
func (s *DepartmentService) DeleteDept(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return errors.New("未选择删除项")
	}
	if err := s.assertNoChild(ctx, ids); err != nil {
		return err
	}
	if err := s.assertNoRef(ctx, "用户", &system.SysUser{}, "dept_id", ids); err != nil {
		return err
	}
	if err := s.assertNoRef(ctx, "用户", &system.SysUserDepartment{}, "sys_department_id", ids); err != nil {
		return err
	}
	if err := s.assertNoRef(ctx, "岗位", &system.SysPost{}, "dept_id", ids); err != nil {
		return err
	}
	if err := s.assertNoRef(ctx, "角色数据权限", &system.SysRoleDepartment{}, "sys_department_id", ids); err != nil {
		return err
	}
	return global.OPS_DB.WithContext(ctx).Where("dept_id IN ?", ids).Delete(&system.SysDepartment{}).Error
}

// GetDeptOptionList 全量部门平表(对齐前端 GET /system/dept/optionselect,下拉选择用,前端组装树)。
func (s *DepartmentService) GetDeptOptionList(ctx context.Context) (list []system.SysDepartment, err error) {
	err = global.OPS_DB.WithContext(ctx).Order(deptOrder).Find(&list).Error
	return
}

// GetDeptTree 构建启用部门树(对齐前端 GET /system/post/deptTree,返回 CommonTreeRecord 结构)。
// 供岗位页左侧部门树 + 新增抽屉部门选择用;查启用部门按 parentId 递归组装。
func (s *DepartmentService) GetDeptTree(ctx context.Context) ([]system.DeptTreeNode, error) {
	var depts []system.SysDepartment
	if err := global.OPS_DB.WithContext(ctx).
		Where("status = ?", "0").Order(deptOrder).Find(&depts).Error; err != nil {
		return nil, err
	}
	return buildDeptTree(depts), nil
}

// checkDeptNameUnique 校验同父级下部门名称唯一(对齐 RuoYi checkDeptNameUnique);excludeId>0 时排除自身。
func (s *DepartmentService) checkDeptNameUnique(ctx context.Context, deptName string, parentId, excludeId int64) error {
	var cnt int64
	db := global.OPS_DB.WithContext(ctx).Model(&system.SysDepartment{}).
		Where("dept_name = ? AND parent_id = ?", deptName, parentId)
	if excludeId > 0 {
		db = db.Where("dept_id <> ?", excludeId)
	}
	if err := db.Count(&cnt).Error; err != nil {
		return err
	}
	if cnt > 0 {
		return errors.New("同级下已存在同名部门")
	}
	return nil
}

// isDescendant 判断 candidateId 是否为 deptId 的子孙(防环:新父级不可落在自身子树)。
// 判据:candidate 的祖级链 ancestors(形如 "0,1,5")中含 deptId 段。
func (s *DepartmentService) isDescendant(ctx context.Context, deptId, candidateId int64) bool {
	if candidateId == 0 {
		return false
	}
	var c system.SysDepartment
	if err := global.OPS_DB.WithContext(ctx).Where("dept_id = ?", candidateId).First(&c).Error; err != nil {
		return false
	}
	token := "," + strconv.FormatInt(deptId, 10) + ","
	return strings.Contains(","+c.Ancestors+",", token)
}

// assertNoChild 校验待删部门下无子部门。
func (s *DepartmentService) assertNoChild(ctx context.Context, ids []int64) error {
	var cnt int64
	if err := global.OPS_DB.WithContext(ctx).Model(&system.SysDepartment{}).
		Where("parent_id IN ?", ids).Count(&cnt).Error; err != nil {
		return err
	}
	if cnt > 0 {
		return errors.New("存在下级部门,不允许删除")
	}
	return nil
}

// assertNoRef 通用引用计数校验:指定 model 的指定列 IN ids 是否有记录,有则返回"存在{label},不允许删除"。
func (s *DepartmentService) assertNoRef(ctx context.Context, label string, model interface{}, column string, ids []int64) error {
	var cnt int64
	if err := global.OPS_DB.WithContext(ctx).Model(model).Where(column+" IN ?", ids).Count(&cnt).Error; err != nil {
		return err
	}
	if cnt > 0 {
		return errors.New("部门下存在" + label + ",不允许删除")
	}
	return nil
}

// buildDeptTree 将部门平表按 parentId 递归组装为树(顶级 parentId=0)。
// 依赖 depts 已按 order_num ASC 排序,保证同层展示顺序。
func buildDeptTree(depts []system.SysDepartment) []system.DeptTreeNode {
	byParent := make(map[int64][]system.SysDepartment, len(depts))
	for _, d := range depts {
		byParent[d.ParentId] = append(byParent[d.ParentId], d)
	}
	var build func(parentId int64) []system.DeptTreeNode
	build = func(parentId int64) []system.DeptTreeNode {
		children := byParent[parentId]
		nodes := make([]system.DeptTreeNode, 0, len(children))
		for _, d := range children {
			nodes = append(nodes, system.DeptTreeNode{
				Id:       d.DeptId,
				ParentId: d.ParentId,
				Label:    d.DeptName,
				Weight:   d.OrderNum,
				Children: build(d.DeptId),
			})
		}
		return nodes
	}
	return build(0)
}
