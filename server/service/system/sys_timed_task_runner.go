package system

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/task"
	"github.com/hllkk/devops-admin/server/utils/datascope"
	"github.com/hllkk/devops-admin/server/utils/logger"
	"github.com/hllkk/devops-admin/server/utils/sse"
)

// 超时用 var 而非 const: 单测需收窄
var (
	defaultMethodTimeout = 5 * time.Minute
	defaultHTTPTimeout   = 30 * time.Second
)

const (
	maxHTTPRespBytes = 1 << 20 // HTTP 响应体读取上限 1MB
	maxLogTextLen    = 4000    // error/output 落库截断长度
	alertEventName   = "timedTask:alert"
)

// errTaskTimeout 超时哨兵: Runner 据此把状态记为 timeout 而非 fail
var errTaskTimeout = errors.New("任务执行超时")

func truncateText(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(截断)"
}

// taskRunMu 任务级互斥锁池: key=taskID(uint)。RunTask 用 TryLock 防同任务重叠执行
// (手动触发狂点、慢任务 + 高频 spec 叠加)。单实例内有效; 删除任务残留一个空 mutex(可忽略)。
var taskRunMu sync.Map

func taskLock(id uint) *sync.Mutex {
	v, _ := taskRunMu.LoadOrStore(id, &sync.Mutex{})
	return v.(*sync.Mutex)
}

// RunTask 统一执行入口(自动调度与手动触发共用):
// panic 兜底、起止/耗时/状态/错误落 sys_timed_task_logs、失败经 SSE 告警。
// 阻塞执行; 调度器回调与手动触发均应在独立 goroutine 中调用。
func (s *TimedTaskService) RunTask(t system.SysTimedTask, trigger string) {
	// 单实例互斥: 同一任务上一次执行未结束时跳过本次触发(防手动狂点、慢任务+高频spec叠加)。
	mu := taskLock(t.ID)
	if !mu.TryLock() {
		logger.Bg().Mod("timedTask").Info(fmt.Sprintf("任务 %s 上一次执行尚未结束, 跳过本次触发(%s)", t.Name, trigger))
		return
	}
	defer mu.Unlock()

	started := time.Now()
	logWritten := false
	// panic 兜底(对齐本函数 doc): 链路任一环意外 panic 时记日志, 并在正常落库未发生时补一条
	// fail 日志防丢失。robfig/cron v3 默认不 recover, 不兜底会崩整个进程。
	defer func() {
		if r := recover(); r != nil {
			logger.Bg().Mod("timedTask").Error(fmt.Sprintf("任务执行 panic: %s: %v", t.Name, r))
			if logWritten {
				return
			}
			finished := time.Now()
			row := system.SysTimedTaskLog{
				TaskId:      t.ID,
				TaskName:    t.Name,
				TriggerType: trigger,
				StartedAt:   started,
				FinishedAt:  finished,
				DurationMs:  finished.Sub(started).Milliseconds(),
				Status:      system.TimedTaskStatusFail,
				ErrorMsg:    truncateText(fmt.Sprintf("panic: %v", r), maxLogTextLen),
			}
			ctx := datascope.WithSystem(context.Background())
			if err := global.OPS_DB.WithContext(ctx).Create(&row).Error; err != nil {
				logger.Bg().Mod("timedTask").Err(err).Error("定时任务 panic 兜底日志落库失败: " + t.Name)
			}
		}
	}()

	var output string
	var runErr error
	switch t.ExecutorType {
	case system.TimedTaskExecutorMethod:
		output, runErr = s.runMethod(t)
	case system.TimedTaskExecutorHTTP:
		output, runErr = s.runHTTP(t)
	default:
		runErr = fmt.Errorf("未知执行器类型: %s", t.ExecutorType)
	}
	finished := time.Now()

	status := system.TimedTaskStatusSuccess
	errMsg := ""
	if runErr != nil {
		if errors.Is(runErr, errTaskTimeout) {
			status = system.TimedTaskStatusTimeout
		} else {
			status = system.TimedTaskStatusFail
		}
		errMsg = truncateText(runErr.Error(), maxLogTextLen)
	}

	logRow := system.SysTimedTaskLog{
		TaskId:      t.ID,
		TaskName:    t.Name,
		TriggerType: trigger,
		StartedAt:   started,
		FinishedAt:  finished,
		DurationMs:  finished.Sub(started).Milliseconds(),
		Status:      status,
		ErrorMsg:    errMsg,
		Output:      truncateText(output, maxLogTextLen),
	}
	ctx := datascope.WithSystem(context.Background())
	if err := global.OPS_DB.WithContext(ctx).Create(&logRow).Error; err != nil {
		logger.Bg().Mod("timedTask").Err(err).Error("定时任务执行日志落库失败: " + t.Name)
	}
	logWritten = true
	if runErr != nil {
		logger.Bg().Mod("timedTask").Err(runErr).Error("定时任务执行失败: " + t.Name)
		s.alertFailure(t, errMsg)
	}
}

// runMethod 执行已注册本体方法。
// 超时语义: 只能标记状态, goroutine 无法强杀; 任务函数应响应 ctx 取消。
func (s *TimedTaskService) runMethod(t system.SysTimedTask) (string, error) {
	fn, ok := task.Get(t.MethodName)
	if !ok {
		return "", fmt.Errorf("方法 %s 未注册(需在 initialize/timer.go 经 task.Register 注册)", t.MethodName)
	}
	ctx, cancel := context.WithTimeout(datascope.WithSystem(context.Background()), defaultMethodTimeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("panic: %v", r)
			}
		}()
		done <- fn(ctx, json.RawMessage(t.Params))
	}()
	select {
	case err := <-done:
		if err != nil && errors.Is(err, context.DeadlineExceeded) {
			return "", errTaskTimeout
		}
		return "", err
	case <-ctx.Done():
		return "", errTaskTimeout
	}
}

// runHTTP 执行 HTTP 回调(SSRF 防护见 sys_timed_task_http.go)
func (s *TimedTaskService) runHTTP(t system.SysTimedTask) (string, error) {
	u, err := url.Parse(t.HttpUrl)
	if err != nil {
		return "", fmt.Errorf("URL 非法: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("仅允许 http/https, 实际为 %q", u.Scheme)
	}
	method := strings.ToUpper(strings.TrimSpace(t.HttpMethod))
	if method == "" {
		method = http.MethodGet
	}
	var body io.Reader
	if t.HttpBody != "" {
		body = strings.NewReader(t.HttpBody)
	}
	req, err := http.NewRequest(method, t.HttpUrl, body)
	if err != nil {
		return "", fmt.Errorf("构造请求失败: %w", err)
	}
	if len(t.HttpHeader) > 0 {
		var hdr map[string]string
		if err := json.Unmarshal(t.HttpHeader, &hdr); err != nil {
			return "", fmt.Errorf("http_header 必须是 JSON 对象: %w", err)
		}
		for k, v := range hdr {
			req.Header.Set(k, v)
		}
	}

	client := newTimedTaskHTTPClient(t.HttpAllowPrivate, defaultHTTPTimeout)
	resp, err := client.Do(req)
	if err != nil {
		var uerr *url.Error
		if errors.As(err, &uerr) && uerr.Timeout() {
			return "", errTaskTimeout
		}
		return "", err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(io.LimitReader(resp.Body, maxHTTPRespBytes))
	out := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(data))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return out, fmt.Errorf("非 2xx 响应: %d", resp.StatusCode)
	}
	return out, nil
}

// alertFailure 失败告警: 查超级管理员(SuperAdmin=true 角色)用户, 经本体 SSE Hub 定向推送(离线静默丢弃, 不阻塞)。
// 历史硬编码 alertRoleID=888 是 GVA 遗留:devops-admin 角色主键走雪花 int64, 不存在 888 角色, 原查询恒空。
func (s *TimedTaskService) alertFailure(t system.SysTimedTask, errMsg string) {
	var ids []uint
	superRoleIDs := global.OPS_DB.Model(&system.SysRole{}).Select("role_id").Where("super_admin = ?", true)
	if err := global.OPS_DB.Model(&system.SysUserRole{}).
		Where("sys_role_id IN (?)", superRoleIDs).
		Pluck("sys_user_id", &ids).Error; err != nil {
		logger.Bg().Mod("timedTask").Err(err).Error("查询告警接收人(超管)失败")
		return
	}
	if len(ids) == 0 {
		return
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"type":  alertEventName,
		"taskId": t.ID,
		"name":   t.Name,
		"error":  errMsg,
		"time":   time.Now().Format(time.RFC3339),
	})
	// Name 留空走 SSE 默认 message 通道:前端 useEventSource(url,[]) 只监听 message,
	// 故通知/告警统一走 message,用 payload.type 区分(见 web/src/utils/sse.ts)。
	sse.Default().PublishToUsers(ids, sse.Event{Name: "", Data: string(payload)})
}
