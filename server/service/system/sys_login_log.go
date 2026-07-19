package system

import (
	"context"
	"errors"
	"time"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
)

type LoginLogService struct{}

// CreateLoginLog 写一条登录日志(供登录链路成功/失败记录)
func (s *LoginLogService) CreateLoginLog(ctx context.Context, log system.SysLoginLog) error {
	if log.LoginTime.IsZero() {
		log.LoginTime = time.Now()
	}
	return global.OPS_DB.WithContext(ctx).Create(&log).Error
}

// GetLoginLogList 分页查登录日志列表(对齐前端 GET /log/loginlog/list)。
// 按 userName/ipaddr 模糊、status 精确过滤;分页统一走 PageInfo.LimitOffset(内置 MaxPageSize=100 截断)。
func (s *LoginLogService) GetLoginLogList(ctx context.Context, q systemReq.LoginLogSearch) (list []system.SysLoginLog, total int64, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&system.SysLoginLog{})
	if q.UserName != "" {
		db = db.Where("user_name LIKE ?", "%"+q.UserName+"%")
	}
	if q.Ipaddr != "" {
		db = db.Where("ipaddr LIKE ?", "%"+q.Ipaddr+"%")
	}
	if q.Status != "" {
		db = db.Where("status = ?", q.Status)
	}
	limit, offset := q.LimitOffset()
	if limit > 0 {
		err = db.Count(&total).Order("info_id DESC").Limit(limit).Offset(offset).Find(&list).Error
	} else {
		err = db.Count(&total).Order("info_id DESC").Find(&list).Error
	}
	return
}

// DeleteLoginLog 批量删除登录日志(按 info_id,物理删除:日志为 append-only 审计数据,无恢复需求)。
// 模型嵌入 OPS_AUDIT_MODEL 带 DeletedAt 软删除,故用 Unscoped() 物理删,避免软删垃圾行累积。
func (s *LoginLogService) DeleteLoginLog(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return errors.New("未选择删除项")
	}
	return global.OPS_DB.WithContext(ctx).Unscoped().Where("info_id IN ?", ids).Delete(&system.SysLoginLog{}).Error
}

// CleanLoginLog 清空全部登录日志(物理删除)。
func (s *LoginLogService) CleanLoginLog(ctx context.Context) error {
	return global.OPS_DB.WithContext(ctx).Unscoped().Where("1 = 1").Delete(&system.SysLoginLog{}).Error
}

// UnlockLoginLog 解锁账号:清除失败计数与锁(复用 security_lock.ClearLoginFail)。
func (s *LoginLogService) UnlockLoginLog(ctx context.Context, username string) error {
	if username == "" {
		return errors.New("用户名不能为空")
	}
	ClearLoginFail(ctx, username)
	return nil
}
