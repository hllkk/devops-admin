package system

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/common"
	"github.com/hllkk/devops-admin/server/model/system"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
	"github.com/hllkk/devops-admin/server/utils/logger"
	"github.com/hllkk/devops-admin/server/utils/sse"
)

type SysErrorService struct{}

// errorFieldMaxRunes 错误日志前端可控长文本字段(rune)上限。
// createSysError 为匿名公开上报入口(无鉴权可写),按 rune 截断防存储膨胀/DoS。
const errorFieldMaxRunes = 4000

// CreateSysError 创建错误日志记录
// Author [yourname](https://github.com/yourname)
func (sysErrorService *SysErrorService) CreateSysError(ctx context.Context, sysError *system.SysError) (err error) {
	if global.OPS_DB == nil {
		return nil
	}
	// 匿名上报不可信:对前端可控字段截断(Form/Info/Solution 按 rune 限长,
	// RequestID/TraceID 对齐 varchar(64) 防超长致插入失败),防撑大 text 列 / 刷库。
	clampStrPtr(sysError.Form, errorFieldMaxRunes)
	clampStrPtr(sysError.Info, errorFieldMaxRunes)
	clampStrPtr(sysError.Solution, errorFieldMaxRunes)
	clampStrPtr(&sysError.RequestID, 64)
	clampStrPtr(&sysError.TraceID, 64)
	err = global.OPS_DB.WithContext(ctx).Create(sysError).Error
	return err
}

// clampStrPtr 按 rune 上限截断字符串(超长则保留前 max 个 rune + 截断标记);nil 指针跳过。
func clampStrPtr(p *string, max int) {
	if p == nil || max <= 0 {
		return
	}
	if r := []rune(*p); len(r) > max {
		*p = string(r[:max]) + "...[截断]"
	}
}

// DeleteSysError 删除错误日志记录
// Author [yourname](https://github.com/yourname)
func (sysErrorService *SysErrorService) DeleteSysError(ctx context.Context, ID string) (err error) {
	err = global.OPS_DB.WithContext(ctx).Delete(&system.SysError{}, "id = ?", ID).Error
	return err
}

// DeleteSysErrorByIds 批量删除错误日志记录
// Author [yourname](https://github.com/yourname)
func (sysErrorService *SysErrorService) DeleteSysErrorByIds(ctx context.Context, IDs []string) (err error) {
	err = global.OPS_DB.WithContext(ctx).Delete(&[]system.SysError{}, "id in ?", IDs).Error
	return err
}

// UpdateSysError 更新错误日志记录
// Author [yourname](https://github.com/yourname)
func (sysErrorService *SysErrorService) UpdateSysError(ctx context.Context, sysError system.SysError) (err error) {
	err = global.OPS_DB.WithContext(ctx).Model(&system.SysError{}).Where("id = ?", sysError.ID).Updates(&sysError).Error
	return err
}

// GetSysError 根据ID获取错误日志记录
// Author [yourname](https://github.com/yourname)
func (sysErrorService *SysErrorService) GetSysError(ctx context.Context, ID string) (sysError system.SysError, err error) {
	err = global.OPS_DB.WithContext(ctx).Where("id = ?", ID).First(&sysError).Error
	return
}

// GetSysErrorInfoList 分页获取错误日志记录
// Author [yourname](https://github.com/yourname)
func (sysErrorService *SysErrorService) GetSysErrorInfoList(ctx context.Context, info systemReq.SysErrorSearch) (list []system.SysError, total int64, err error) {
	limit, offset := info.LimitOffset()
	// 创建db
	db := global.OPS_DB.WithContext(ctx).Model(&system.SysError{}).Order("create_time desc")
	var sysErrors []system.SysError
	// 如果有条件搜索 下方会自动创建搜索语句
	if len(info.CreatedAtRange) == 2 {
		db = db.Where("create_time BETWEEN ? AND ?", info.CreatedAtRange[0], info.CreatedAtRange[1])
	}

	if info.Form != nil && *info.Form != "" {
		db = db.Where("form = ?", *info.Form)
	}
	if info.Info != nil && *info.Info != "" {
		db = db.Where("info LIKE ?", "%"+*info.Info+"%")
	}
	err = db.Count(&total).Error
	if err != nil {
		return
	}

	if limit != 0 {
		db = db.Limit(limit).Offset(offset)
	}

	err = db.Find(&sysErrors).Error
	return sysErrors, total, err
}

// GetSysErrorSolution 异步处理错误
// Author [yourname](https://github.com/yourname)
func (sysErrorService *SysErrorService) GetSysErrorSolution(ctx context.Context, ID string, userID int64) (err error) {
	// 立即更新为处理中
	err = global.OPS_DB.WithContext(ctx).Model(&system.SysError{}).Where("id = ?", ID).Update("status", "处理中").Error
	if err != nil {
		return err
	}

	// 异步协程生成解决方案后更新状态。
	// goroutine 存活期远超请求(LLM 调用可达数分钟),而请求 ctx 在 handler 返回后
	// 即被取消,直接用 ctx 会让协程内查询/收尾更新全部 context.Canceled、任务永远
	// 卡在"处理中"。WithoutCancel 脱离请求取消生命周期,同时保留 ctx 里的链路字段
	// (request_id/trace_id 流入本协程的 SQL 日志与出站 LLM 调用)。
	bgCtx := context.WithoutCancel(ctx)
	go func(id string, uid int64) {
		// 查询当前错误信息用于生成方案
		var se system.SysError
		if err := global.OPS_DB.WithContext(bgCtx).Model(&system.SysError{}).Where("id = ?", id).First(&se).Error; err != nil {
			logger.Bg().Mod("biz").Err(err).Warn("AI处理: 查询错误日志失败")
			_ = global.OPS_DB.WithContext(bgCtx).Model(&system.SysError{}).Where("id = ?", id).Update("status", "处理失败").Error
			pushErrorlogSSE(uint(uid), id, "处理失败", "")
			return
		}

		var form, info string
		if se.Form != nil {
			form = *se.Form
		}
		if se.Info != nil {
			info = *se.Info
		}

		var solution string
		var llmErr error
		provider := global.OPS_CONFIG.Ai.Provider

		if provider == "ollama" {
			// 使用本地 Ollama 模型分析
			solution, llmErr = (&AiService{}).AnalyzeError(bgCtx, form, info)
		} else {
			// external 或空值(向后兼容): 走原有 autocode.ai-path 代理路径
			llmReq := common.JSONMap{
				"mode": "solution",
				"info": info,
				"form": form,
			}
			if data, err := (&AutoCodeService{}).LLMAuto(bgCtx, llmReq); err == nil {
				solution = fmt.Sprintf("%v", data.(map[string]interface{})["text"])
			} else {
				llmErr = err
			}
		}

		if llmErr == nil {
			if err := global.OPS_DB.WithContext(bgCtx).Model(&system.SysError{}).Where("id = ?", id).Updates(map[string]interface{}{"status": "处理完成", "solution": solution}).Error; err != nil {
				logger.Bg().Mod("biz").Err(err).Warn("AI处理: 写入解决方案失败")
			}
			pushErrorlogSSE(uint(uid), id, "处理完成", solution)
		} else {
			logger.Bg().Mod("biz").Err(llmErr).Warn("AI处理: 生成解决方案失败")
			if err := global.OPS_DB.WithContext(bgCtx).Model(&system.SysError{}).Where("id = ?", id).Update("status", "处理失败").Error; err != nil {
				logger.Bg().Mod("biz").Err(err).Warn("AI处理: 更新失败状态失败")
			}
			pushErrorlogSSE(uint(uid), id, "处理失败", "")
		}
	}(ID, userID)

	return nil
}

// pushErrorlogSSE 向触发 AI 处理的用户推送状态变更(用户不在线时静默丢弃)。
func pushErrorlogSSE(userID uint, id string, status string, solution string) {
	payload, _ := json.Marshal(map[string]string{
		"type":     "errorlog:update",
		"id":       id,
		"status":   status,
		"solution": solution,
	})
	sse.Default().Publish(userID, sse.Event{Name: "", Data: string(payload)})
}
