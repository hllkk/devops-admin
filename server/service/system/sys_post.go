package system

import (
	"context"
	"errors"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
)

// PostService 岗位业务服务(对齐前端 /system/post/* 资源)
type PostService struct{}

// postOrder 岗位列表统一排序:post_sort 升序,同序按 post_id 降序(对齐 RuoYi/前端展示惯例)。
const postOrder = "post_sort ASC, post_id DESC"

// GetPostList 分页查岗位列表(对齐前端 GET /system/post/list)。
// postCode/postName 模糊;status/deptId/belongDeptId 精确(belongDeptId 为左侧部门树点击过滤);分页走 PageInfo.LimitOffset。
func (s *PostService) GetPostList(ctx context.Context, q systemReq.PostSearch) (list []system.SysPost, total int64, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&system.SysPost{})
	if q.PostCode != "" {
		db = db.Where("post_code LIKE ?", "%"+q.PostCode+"%")
	}
	if q.PostName != "" {
		db = db.Where("post_name LIKE ?", "%"+q.PostName+"%")
	}
	if q.Status != "" {
		db = db.Where("status = ?", q.Status)
	}
	// belongDeptId(部门树点击)与 deptId 同为 dept_id 精确过滤,二选一,前者优先
	if q.BelongDeptId > 0 {
		db = db.Where("dept_id = ?", q.BelongDeptId)
	} else if q.DeptId > 0 {
		db = db.Where("dept_id = ?", q.DeptId)
	}
	limit, offset := q.LimitOffset()
	if limit > 0 {
		err = db.Count(&total).Order(postOrder).Limit(limit).Offset(offset).Find(&list).Error
	} else {
		err = db.Count(&total).Order(postOrder).Find(&list).Error
	}
	return
}

// CreatePost 新增岗位;postCode/postName 必填 + postCode 唯一性校验(对齐 RuoYi),createBy 填审计字段。
func (s *PostService) CreatePost(ctx context.Context, req systemReq.PostOperateParams, createBy int64) error {
	if req.PostCode == "" {
		return errors.New("岗位编码不能为空")
	}
	if req.PostName == "" {
		return errors.New("岗位名称不能为空")
	}
	var cnt int64
	if err := global.OPS_DB.WithContext(ctx).Model(&system.SysPost{}).
		Where("post_code = ?", req.PostCode).Count(&cnt).Error; err != nil {
		return err
	}
	if cnt > 0 {
		return errors.New("岗位编码已存在")
	}
	p := system.SysPost{
		DeptId:       req.DeptId,
		PostCode:     req.PostCode,
		PostCategory: req.PostCategory,
		PostName:     req.PostName,
		PostSort:     req.PostSort,
		Status:       req.Status,
		Remark:       req.Remark,
	}
	// CreateBy/UpdateBy 为内嵌 OPS_AUDIT_MODEL 的提升字段,struct literal 中不可直接命名,改用赋值写入。
	p.CreateBy = createBy
	p.UpdateBy = createBy
	return global.OPS_DB.WithContext(ctx).Create(&p).Error
}

// UpdatePost 修改岗位;postId 必填 + postCode 唯一性校验(排除自身),updateBy 填审计字段。
func (s *PostService) UpdatePost(ctx context.Context, req systemReq.PostOperateParams, updateBy int64) error {
	if req.PostId == 0 {
		return errors.New("岗位ID不能为空")
	}
	if req.PostCode == "" {
		return errors.New("岗位编码不能为空")
	}
	var cnt int64
	if err := global.OPS_DB.WithContext(ctx).Model(&system.SysPost{}).
		Where("post_code = ? AND post_id <> ?", req.PostCode, req.PostId).Count(&cnt).Error; err != nil {
		return err
	}
	if cnt > 0 {
		return errors.New("岗位编码已存在")
	}
	return global.OPS_DB.WithContext(ctx).Model(&system.SysPost{}).Where("post_id = ?", req.PostId).
		Updates(map[string]interface{}{
			"dept_id":       req.DeptId,
			"post_code":     req.PostCode,
			"post_category": req.PostCategory,
			"post_name":     req.PostName,
			"post_sort":     req.PostSort,
			"status":        req.Status,
			"remark":        req.Remark,
			"update_by":     updateBy,
		}).Error
}

// DeletePost 批量删除岗位;已被用户引用(sys_user_post)的岗位禁止删除(对齐 RuoYi "岗位已分配,不能删除")。
func (s *PostService) DeletePost(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return errors.New("未选择删除项")
	}
	var used int64
	if err := global.OPS_DB.WithContext(ctx).Model(&system.SysUserPost{}).
		Where("sys_post_id IN ?", ids).Count(&used).Error; err != nil {
		return err
	}
	if used > 0 {
		return errors.New("岗位已分配给用户,不能删除")
	}
	return global.OPS_DB.WithContext(ctx).Where("post_id IN ?", ids).Delete(&system.SysPost{}).Error
}

// GetPostOptionList 岗位下拉选择(对齐前端 GET /system/post/optionselect)。
// 默认返回 status='0' 启用岗位;deptId>0 时限定该部门。供用户管理抽屉分配岗位用。
func (s *PostService) GetPostOptionList(ctx context.Context, deptId int64) (list []system.SysPost, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&system.SysPost{}).Where("status = ?", "0")
	if deptId > 0 {
		db = db.Where("dept_id = ?", deptId)
	}
	err = db.Order(postOrder).Find(&list).Error
	return
}

// ExportPostList 按列表查询条件导出岗位(全量,不分页;过滤条件与 GetPostList 一致,加导出上限)。
func (s *PostService) ExportPostList(ctx context.Context, q systemReq.PostSearch) (list []system.SysPost, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&system.SysPost{})
	if q.PostCode != "" {
		db = db.Where("post_code LIKE ?", "%"+q.PostCode+"%")
	}
	if q.PostName != "" {
		db = db.Where("post_name LIKE ?", "%"+q.PostName+"%")
	}
	if q.Status != "" {
		db = db.Where("status = ?", q.Status)
	}
	if q.BelongDeptId > 0 {
		db = db.Where("dept_id = ?", q.BelongDeptId)
	} else if q.DeptId > 0 {
		db = db.Where("dept_id = ?", q.DeptId)
	}
	err = db.Order(postOrder).Limit(ExportMaxRows).Find(&list).Error
	return
}

