package system

import (
	"time"

	"go.uber.org/zap"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
)

// LoginLogService 登录日志服务：记录登录结果、分页查询、批量删除/清空。
type LoginLogService struct{}

// RecordLogin 记录一次登录（append-only）。写入失败仅告警，不阻断调用方流程。
// status: "0" 成功 / "1" 失败；browser 入参可传 User-Agent 原文。
func (s *LoginLogService) RecordLogin(userName, ipaddr, browser, status, msg string) {
	if global.OPS_DB == nil {
		return
	}
	now := time.Now()
	log := system.SysLoginLog{
		UserName:  userName,
		Ipaddr:    ipaddr,
		Browser:   browser,
		Status:    status,
		Msg:       msg,
		LoginTime: &now,
	}
	if err := global.OPS_DB.Create(&log).Error; err != nil {
		global.OPS_LOG.Warn("写入登录日志失败", zap.Error(err))
	}
}

// GetLoginLogList 分页查询登录日志，返回列表与总数（对齐前端 LoginLogList{pageNum,pageSize,total,rows}）。
func (s *LoginLogService) GetLoginLogList(search systemReq.LoginLogSearch) (list []system.SysLoginLog, total int64, err error) {
	db := global.OPS_DB.Model(&system.SysLoginLog{})
	if search.UserName != "" {
		db = db.Where("user_name LIKE ?", "%"+search.UserName+"%")
	}
	if search.Ipaddr != "" {
		db = db.Where("ipaddr LIKE ?", "%"+search.Ipaddr+"%")
	}
	if search.Status != "" {
		db = db.Where("status = ?", search.Status)
	}
	if err = db.Count(&total).Error; err != nil {
		return
	}
	err = db.Scopes(search.Paginate()).Order("login_time DESC").Find(&list).Error
	return
}

// DeleteLoginLog 批量删除登录日志（按 infoId）。
func (s *LoginLogService) DeleteLoginLog(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	return global.OPS_DB.Delete(&system.SysLoginLog{}, "info_id IN ?", ids).Error
}

// CleanLoginLog 清空全部登录日志。
func (s *LoginLogService) CleanLoginLog() error {
	return global.OPS_DB.Where("1 = 1").Delete(&system.SysLoginLog{}).Error
}
