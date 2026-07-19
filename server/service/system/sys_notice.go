package system

import (
	"context"
	"errors"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
)

// NoticeService 通知公告业务服务(对齐前端 /system/notice/* 资源)
type NoticeService struct{}

// GetNoticeList 分页查通知公告列表(对齐前端 GET /system/notice/list)。
// 按 noticeTitle 模糊、noticeType 精确过滤;
// createByName 由 createBy 关联 sys_users.user_name 批量查组装(CreateByName 标 gorm:"-" 不建列)。
func (s *NoticeService) GetNoticeList(ctx context.Context, q systemReq.NoticeSearch) (list []system.SysNotice, total int64, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&system.SysNotice{})
	if q.NoticeTitle != "" {
		db = db.Where("notice_title LIKE ?", "%"+q.NoticeTitle+"%")
	}
	if q.NoticeType != "" {
		db = db.Where("notice_type = ?", q.NoticeType)
	}
	limit, offset := q.LimitOffset()
	if limit > 0 {
		err = db.Count(&total).Order("notice_id DESC").Limit(limit).Offset(offset).Find(&list).Error
	} else {
		err = db.Count(&total).Order("notice_id DESC").Find(&list).Error
	}
	if err != nil || len(list) == 0 {
		return
	}
	fillCreateByName(ctx, list)
	return
}

// fillCreateByName 收集 createBy(用户id) 去重,批量查 sys_users.user_name 回填 CreateByName。
// 名称查询失败不阻断列表(仅影响展示,尽力而为)。
func fillCreateByName(ctx context.Context, list []system.SysNotice) {
	uidSet := make(map[int64]struct{}, len(list))
	for i := range list {
		if list[i].CreateBy != 0 {
			uidSet[list[i].CreateBy] = struct{}{}
		}
	}
	if len(uidSet) == 0 {
		return
	}
	uids := make([]int64, 0, len(uidSet))
	for uid := range uidSet {
		uids = append(uids, uid)
	}
	type idName struct {
		ID       int64
		UserName string
	}
	var users []idName
	// SysUser 主键 DB 列复用 id(column:id),故 WHERE id IN ?
	if qerr := global.OPS_DB.WithContext(ctx).Table("sys_users").
		Select("id, user_name").Where("id IN ?", uids).Scan(&users).Error; qerr != nil {
		return
	}
	nameMap := make(map[int64]string, len(users))
	for _, u := range users {
		nameMap[u.ID] = u.UserName
	}
	for i := range list {
		if name, ok := nameMap[list[i].CreateBy]; ok {
			list[i].CreateByName = name
		}
	}
}

// CreateNotice 新增通知公告;createBy/updateBy 填审计字段。
func (s *NoticeService) CreateNotice(ctx context.Context, req systemReq.NoticeOperateParams, createBy int64) error {
	if req.NoticeTitle == "" {
		return errors.New("公告标题不能为空")
	}
	n := system.SysNotice{
		NoticeTitle:   req.NoticeTitle,
		NoticeType:    req.NoticeType,
		NoticeContent: req.NoticeContent,
		Status:        req.Status,
	}
	// CreateBy/UpdateBy 为内嵌 OPS_AUDIT_MODEL 的提升字段,struct literal 中不可直接命名,改用赋值。
	n.CreateBy = createBy
	n.UpdateBy = createBy
	return global.OPS_DB.WithContext(ctx).Create(&n).Error
}

// UpdateNotice 修改通知公告;noticeId 必填,updateBy 填审计字段。
// 用 map 更新以显式覆盖全部可编辑字段(含空串),避免 GORM struct 更新遗漏零值字段。
func (s *NoticeService) UpdateNotice(ctx context.Context, req systemReq.NoticeOperateParams, updateBy int64) error {
	if req.NoticeId == 0 {
		return errors.New("公告ID不能为空")
	}
	updates := map[string]any{
		"notice_title":   req.NoticeTitle,
		"notice_type":    req.NoticeType,
		"notice_content": req.NoticeContent,
		"status":         req.Status,
		"update_by":      updateBy,
	}
	return global.OPS_DB.WithContext(ctx).Model(&system.SysNotice{}).
		Where("notice_id = ?", req.NoticeId).Updates(updates).Error
}

// DeleteNotice 批量删除通知公告(按 notice_id,业务实体走软删除,与 dict/role 一致)。
func (s *NoticeService) DeleteNotice(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return errors.New("未选择删除项")
	}
	return global.OPS_DB.WithContext(ctx).Where("notice_id IN ?", ids).Delete(&system.SysNotice{}).Error
}
