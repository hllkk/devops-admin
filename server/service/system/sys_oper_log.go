package system

import (
	"sync"
	"time"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// SysOperLogService 操作日志服务:承载操作日志的异步落表。
// 查询/批量删除/清空等读路径由后续 monitor/operlog API 接入,此处先补齐写路径。
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
