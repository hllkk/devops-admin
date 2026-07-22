package system

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// SysOperLogService 操作日志服务:承载操作日志的异步落表 + 读路径(列表/批量删除/清空)。
type SysOperLogService struct{}

// 异步写入: 有界缓冲 + 后台批量落表, 满则丢弃(审计尽力而为, 绝不阻塞业务请求)。
// 与 DataAccessLogService 保持同一套范式, 便于运维认知统一。
var (
	operLogCh         = make(chan system.SysOperLog, 1024)
	operLogWriterOnce sync.Once // reload 会重复调注册入口, 写协程只起一次
)

// Enqueue 非阻塞入队:操作日志中间件采集完成后调用。
// 缓冲已满时丢弃并告警, 绝不阻塞业务请求(审计尽力而为)。
func (s *SysOperLogService) Enqueue(rec system.SysOperLog) {
	select {
	case operLogCh <- rec:
	default:
		logger.Bg().Mod("operlog").Warn("操作日志缓冲已满, 记录被丢弃: " + rec.OperUrl)
	}
}

// StartWriter 启动后台批量写入协程(每 2 秒或攒满 100 条落一次表), 幂等。
func (s *SysOperLogService) StartWriter() {
	operLogWriterOnce.Do(s.startWriter)
}

func (s *SysOperLogService) startWriter() {
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		batch := make([]system.SysOperLog, 0, 128)
		flush := func() {
			if len(batch) == 0 {
				return
			}
			// sys_oper_log 以 sys_ 前缀, 数据权限回调自动跳过, 无需 WithSystem, 亦无递归噪声
			if err := global.OPS_DB.Create(&batch).Error; err != nil {
				logger.Bg().Mod("operlog").Err(err).Warn("操作日志批量写入失败")
			}
			batch = batch[:0]
		}
		for {
			select {
			case rec := <-operLogCh:
				// 按 IP 反查操作地点(异步消费,不阻塞业务);调用方未传时补齐
				if rec.OperLocation == "" && rec.OperIp != "" {
					rec.OperLocation = utils.ParseIPLocation(rec.OperIp)
				}
				batch = append(batch, rec)
				if len(batch) >= 100 {
					flush()
				}
			case <-ticker.C:
				flush()
			}
		}
	}()
}

// GetOperLogList 分页查操作日志列表(对齐前端 GET /log/operlog/list)。
// 按 title/operName/operIp 模糊、businessType/status 精确、operTime 时间范围(BETWEEN)过滤;
// 分页统一走 PageInfo.LimitOffset(内置 MaxPageSize=100 截断)。
func (s *SysOperLogService) GetOperLogList(ctx context.Context, q systemReq.OperLogSearch) (list []system.SysOperLog, total int64, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&system.SysOperLog{})
	if q.Title != "" {
		db = db.Where("title LIKE ?", "%"+q.Title+"%")
	}
	if q.BusinessType != "" {
		db = db.Where("business_type = ?", q.BusinessType)
	}
	if q.OperName != "" {
		db = db.Where("oper_name LIKE ?", "%"+q.OperName+"%")
	}
	if q.OperIp != "" {
		db = db.Where("oper_ip LIKE ?", "%"+q.OperIp+"%")
	}
	if q.Status != "" {
		db = db.Where("status = ?", q.Status)
	}
	// operTime 时间范围(BeginTime/EndTime 由 api 层从 query params[beginTime/endTime] 显式取后传入);
	// 仅成对提供时过滤,DB 直接解析 yyyy-MM-dd HH:mm:ss 字符串为 datetime。
	if q.BeginTime != "" && q.EndTime != "" {
		db = db.Where("oper_time BETWEEN ? AND ?", q.BeginTime, q.EndTime)
	}
	limit, offset := q.LimitOffset()
	if limit > 0 {
		err = db.Count(&total).Order("oper_id DESC").Limit(limit).Offset(offset).Find(&list).Error
	} else {
		err = db.Count(&total).Order("oper_id DESC").Find(&list).Error
	}
	return
}

// DeleteOperLog 批量删除操作日志(按 oper_id,物理删除:日志为 append-only 审计数据,无恢复需求)。
// 模型嵌入 OPS_AUDIT_MODEL 带 DeletedAt 软删除,故用 Unscoped() 物理删,避免软删垃圾行累积。
func (s *SysOperLogService) DeleteOperLog(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return errors.New("未选择删除项")
	}
	return global.OPS_DB.WithContext(ctx).Unscoped().Where("oper_id IN ?", ids).Delete(&system.SysOperLog{}).Error
}

// CleanOperLog 清空全部操作日志(物理删除)。
func (s *SysOperLogService) CleanOperLog(ctx context.Context) error {
	return global.OPS_DB.WithContext(ctx).Unscoped().Where("1 = 1").Delete(&system.SysOperLog{}).Error
}

// ExportOperLogList 按条件导出操作日志(全量,不分页;条件与 GetOperLogList 一致,含 operTime 时间范围,加导出上限)。
func (s *SysOperLogService) ExportOperLogList(ctx context.Context, q systemReq.OperLogSearch) (list []system.SysOperLog, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&system.SysOperLog{})
	if q.Title != "" {
		db = db.Where("title LIKE ?", "%"+q.Title+"%")
	}
	if q.BusinessType != "" {
		db = db.Where("business_type = ?", q.BusinessType)
	}
	if q.OperName != "" {
		db = db.Where("oper_name LIKE ?", "%"+q.OperName+"%")
	}
	if q.OperIp != "" {
		db = db.Where("oper_ip LIKE ?", "%"+q.OperIp+"%")
	}
	if q.Status != "" {
		db = db.Where("status = ?", q.Status)
	}
	if q.BeginTime != "" && q.EndTime != "" {
		db = db.Where("oper_time BETWEEN ? AND ?", q.BeginTime, q.EndTime)
	}
	err = db.Order("oper_id DESC").Limit(ExportMaxRows).Find(&list).Error
	return
}
