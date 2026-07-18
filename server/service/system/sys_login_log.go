package system

import (
	"context"
	"time"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
)

type LoginLogService struct{}

// CreateLoginLog 写一条登录日志(供登录链路成功/失败记录)
func (s *LoginLogService) CreateLoginLog(ctx context.Context, log system.SysLoginLog) error {
	if log.LoginTime.IsZero() {
		log.LoginTime = time.Now()
	}
	return global.OPS_DB.WithContext(ctx).Create(&log).Error
}
