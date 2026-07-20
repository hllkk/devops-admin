package system

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
	"github.com/hllkk/devops-admin/server/utils/sse"
	"gorm.io/gorm"
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
// 公告(type=2)全员广播不入 record;通知(type=1)按 targetType 定向展开目标用户、
// 事务落 sys_notice + sys_notice_record(预生成未读行),并对在线用户 SSE 推送。
func (s *NoticeService) CreateNotice(ctx context.Context, req systemReq.NoticeOperateParams, createBy int64) error {
	if req.NoticeTitle == "" {
		return errors.New("公告标题不能为空")
	}
	// 公告强制全员广播;通知 targetType 缺省按 all
	isAnnouncement := req.NoticeType == system.NoticeTypeAnnouncement
	targetType := req.TargetType
	if isAnnouncement || targetType == "" {
		targetType = "all"
	}

	// 展开定向目标(公告/全员 → nil,表示广播不入 record)
	var targetUserIDs []int64
	if !isAnnouncement && targetType != "all" {
		targetUserIDs = s.expandTargetUserIDs(ctx, targetType, req.TargetUserIds, req.TargetDeptIds)
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

	// 事务:落主表 + 定向通知的接收记录(预生成未读行,去重)
	err := global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Create(&n).Error; e != nil {
			return e
		}
		if len(targetUserIDs) == 0 {
			return nil
		}
		records := make([]system.SysNoticeRecord, 0, len(targetUserIDs))
		seen := make(map[int64]struct{}, len(targetUserIDs))
		for _, uid := range targetUserIDs {
			if uid == 0 {
				continue
			}
			if _, ok := seen[uid]; ok {
				continue
			}
			seen[uid] = struct{}{}
			records = append(records, system.SysNoticeRecord{NoticeId: n.NoticeId, UserId: uid})
		}
		if len(records) == 0 {
			return nil
		}
		return tx.CreateInBatches(&records, 500).Error
	})
	if err != nil {
		return err
	}

	// 在线加速推送(离线靠 record 兜底,上线拉取)
	s.publishNotice(n, targetUserIDs)
	return nil
}

// publishNotice 推送通知事件:目标为空(公告/全员)→ 广播;否则投递给目标在线用户。
// 离线静默丢弃(hub 只负责在线加速)。Event.Data 为 JSON,前端 store 统一解析。
// hub 用 uint,用户 ID 为 int64,此处转换。
func (s *NoticeService) publishNotice(n system.SysNotice, targetUserIDs []int64) {
	payload, _ := json.Marshal(map[string]any{
		"noticeId":      n.NoticeId,
		"noticeTitle":   n.NoticeTitle,
		"noticeContent": n.NoticeContent,
		"noticeType":    n.NoticeType,
	})
	event := sse.Event{Name: "", Data: string(payload)}
	if len(targetUserIDs) == 0 {
		sse.Default().Broadcast(event)
		return
	}
	uids := make([]uint, 0, len(targetUserIDs))
	for _, id := range targetUserIDs {
		uids = append(uids, uint(id))
	}
	sse.Default().PublishToUsers(uids, event)
}

// expandTargetUserIDs 展开定向投递目标用户(去重)。
// users:直接用指定用户;depts:部门含子部门下的全部用户(主部门 dept_id ∪ 多部门 sys_user_departments)。
// 用户名查询失败不阻断(尽力而为)。
func (s *NoticeService) expandTargetUserIDs(ctx context.Context, targetType string, userIDs, deptIDs []int64) []int64 {
	set := make(map[int64]struct{})
	if targetType == "users" {
		for _, uid := range userIDs {
			if uid != 0 {
				set[uid] = struct{}{}
			}
		}
	}
	if targetType == "depts" && len(deptIDs) > 0 {
		expanded := DataScopeServiceApp.ExpandDeptIDs(ctx, deptIDs)
		if len(expanded) > 0 {
			// data_scope:skip 旁路数据权限回调:定向投递要查全量目标用户,不受当前操作者范围限制
			db := global.OPS_DB.WithContext(ctx).Set("data_scope:skip", true)
			var primary, multi []int64
			db.Table("sys_users").Where("dept_id IN ?", expanded).Pluck("id", &primary)
			db.Table("sys_user_departments").Where("sys_department_id IN ?", expanded).Pluck("sys_user_id", &multi)
			for _, uid := range primary {
				set[uid] = struct{}{}
			}
			for _, uid := range multi {
				set[uid] = struct{}{}
			}
		}
	}
	out := make([]int64, 0, len(set))
	for uid := range set {
		out = append(out, uid)
	}
	return out
}

// NoticeRecordVO 通知接收视图(join sys_notice 带正文,供前端未读/历史列表)。
type NoticeRecordVO struct {
	NoticeId      int64      `json:"noticeId,string"`
	NoticeTitle   string     `json:"noticeTitle"`
	NoticeType    string     `json:"noticeType"`
	NoticeContent string     `json:"noticeContent"`
	ReadAt        *time.Time `json:"readAt,omitempty"`
	CreateTime    time.Time  `json:"createTime"`
}

// GetNoticeRecordList 当前用户的通知接收列表(分页,可仅未读)。join sys_notice 带正文。
func (s *NoticeService) GetNoticeRecordList(ctx context.Context, userId int64, q systemReq.NoticeUnreadSearch) (list []NoticeRecordVO, total int64, err error) {
	// data_scope:skip 旁路:本人通知查询按 user_id 精确过滤,不受数据权限范围干扰
	db := global.OPS_DB.WithContext(ctx).Set("data_scope:skip", true).
		Table("sys_notice_record AS r").
		Select("r.notice_id, r.read_at, r.create_time, n.notice_title, n.notice_type, n.notice_content").
		Joins("JOIN sys_notice n ON n.notice_id = r.notice_id").
		Where("r.user_id = ?", userId).
		Where("r.deleted_at IS NULL").
		Where("n.deleted_at IS NULL")
	if q.OnlyUnread {
		db = db.Where("r.read_at IS NULL")
	}
	limit, offset := q.LimitOffset()
	if limit > 0 {
		err = db.Count(&total).Order("r.create_time DESC").Limit(limit).Offset(offset).Scan(&list).Error
	} else {
		err = db.Count(&total).Order("r.create_time DESC").Scan(&list).Error
	}
	return
}

// MarkNoticeRead 标记当前用户的通知已读(noticeIds 为空=全部已读)。仅更新本用户未读记录。
func (s *NoticeService) MarkNoticeRead(ctx context.Context, userId int64, noticeIds []int64) error {
	// data_scope:skip 旁路:仅更新本人记录,不受数据权限范围干扰
	q := global.OPS_DB.WithContext(ctx).Set("data_scope:skip", true).Model(&system.SysNoticeRecord{}).
		Where("user_id = ?", userId).
		Where("read_at IS NULL")
	if len(noticeIds) > 0 {
		q = q.Where("notice_id IN ?", noticeIds)
	}
	return q.Update("read_at", time.Now()).Error
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
