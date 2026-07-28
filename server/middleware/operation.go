package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
	"github.com/hllkk/devops-admin/server/service"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// 操作日志元数据覆盖键:业务 handler 可在进入处理前 c.Set 这两个键,
// 精确标注操作所属模块(title)与操作类型(businessType),无需引入注解框架。
// 未设置时由中间件按路由模板/HTTP 方法推导默认值。
const (
	ctxOperTitleKey = "ops_oper_title"         // 系统模块(如 "用户管理")
	ctxOperBTypeKey = "ops_oper_business_type" // 操作类型 '0'~'9'
)

// 业务操作类型枚举(对齐前端 Api.Log.BusinessType 与字典 sys_oper_type)
const (
	bizOther   = "0" // 其它
	bizInsert  = "1" // 新增
	bizUpdate  = "2" // 修改
	bizRemove  = "3" // 删除
	bizGrant   = "4" // 授权
	bizExport  = "5" // 导出
	bizImport  = "6" // 导入
	bizForce   = "7" // 强退
	bizGenCode = "8" // 生成代码
	bizClean   = "9" // 清空
)

// 操作状态(对齐前端 Common.EnableStatus / 字典 sys_common_status:0 正常 1 异常)
const (
	operStatusNormal = "0"
	operStatusError  = "1"
)

// 操作类别(0 其它 1 后台用户 2 手机端用户)
const (
	operatorOther   = "0"
	operatorBackend = "1"
)

// OperationRecord 操作日志中间件:对齐 RuoYi / 前端 Api.Log.OperLog 契约,
// 采集请求参数、响应结果、耗时、状态与操作人,经异步队列落表(sys_oper_log),不阻塞业务请求。
//
// 依赖前置的 AccessLog 已在全局阶段读取并缓存请求体(ctxReqBodyKey, 已脱敏截断)
// 与响应缓冲(ctxRespBufferKey, 上限 1MB):本中间件只读取,不重复读 body、不二次包装 writer。
func OperationRecord() gin.HandlerFunc {
	operLogSvc := &service.ServiceGroupApp.SystemServiceGroup.SysOperLogService
	return func(c *gin.Context) {
		// 仅记录写操作:GET/HEAD 为查询,不落操作日志(避免高频查询撑大 sys_oper_log,
		// 与 resolveBusinessType 将 GET 归为「其它」的语义一致)。请求仍正常放行。
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
			c.Next()
			return
		}
		start := time.Now()

		record := system.SysOperLog{
			OperIp:        c.ClientIP(),
			RequestMethod: c.Request.Method,
			Method:        c.HandlerName(), // 方法名称:Go handler 全限定名(RuoYi 存控制器方法,等价语义)
			OperUrl:       c.Request.URL.Path,
			OperParam:     buildOperParam(c),
			Title:         resolveTitle(c),
			BusinessType:  resolveBusinessType(c),
			OperatorType:  operatorOther,
			OperTime:      start,
		}
		if claims := claimsFromContext(c); claims != nil {
			record.OperName = claims.Username
			record.OperatorType = operatorBackend
		}
		// 链路关联:与 access log 的 request_id/trace_id 对齐,便于跨日志排障
		if f := logger.FromCtx(c.Request.Context()); f != nil {
			record.RequestID = f.GetRequestID()
			record.TraceID = f.GetTraceID()
		}
		// 显式用操作起始时刻 start 落审计时间(优先于 gorm 默认 time.Now(); 非零值 gorm 不覆盖)。
		record.CreatedAt = start
		record.UpdatedAt = start

		c.Next()

		// 状态与耗时:5xx 或存在私有错误视为异常
		statusCode := c.Writer.Status()
		privateErrs := c.Errors.ByType(gin.ErrorTypePrivate).String()
		record.CostTime = int(time.Since(start).Milliseconds())
		if statusCode >= http.StatusInternalServerError || privateErrs != "" {
			record.Status = operStatusError
			record.ErrorMsg = strings.TrimRight(privateErrs, "\n")
		} else {
			record.Status = operStatusNormal
		}

		// 响应结果:复用 AccessLog 已捕获的缓冲(上限 1MB), 再次脱敏截断后落库
		record.JsonResult = buildJsonResult(c)

		// 异步入队:满则丢弃, 绝不阻塞业务请求
		operLogSvc.Enqueue(record)
	}
}

// buildOperParam 构造请求参数文本。
// GET/HEAD: 取查询串(无请求体);其余: 复用 AccessLog 已脱敏截断的请求体(multipart 为 "[文件]")。
func buildOperParam(c *gin.Context) string {
	if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
		q := c.Request.URL.Query()
		if len(q) == 0 {
			return ""
		}
		// 收敛为单值 map 便于阅读;marshal 成 JSON 后走脱敏, 命中敏感字段会被打码
		m := make(map[string]string, len(q))
		for k, v := range q {
			if len(v) > 0 {
				m[k] = v[0]
			}
		}
		raw, _ := json.Marshal(m)
		return logger.Truncate(logger.SanitizeBody("application/json", string(raw)), logger.AccessLogMaxBytes())
	}
	return c.GetString(ctxReqBodyKey)
}

// buildJsonResult 读取 AccessLog 捕获的响应缓冲, 脱敏截断后返回;二进制/下载类响应只标记不落正文。
func buildJsonResult(c *gin.Context) string {
	v, ok := c.Get(ctxRespBufferKey)
	if !ok {
		return ""
	}
	buf, ok := v.(*bytes.Buffer)
	if !ok || buf.Len() == 0 {
		return ""
	}
	h := c.Writer.Header()
	if isBinaryResponse(h) {
		return "[二进制响应]"
	}
	sanitized := logger.SanitizeBody(h.Get("Content-Type"), buf.String())
	return logger.Truncate(sanitized, logger.AccessLogMaxBytes())
}

// isBinaryResponse 判定是否为下载/二进制响应(不把二进制正文塞进 text 列)。
func isBinaryResponse(h http.Header) bool {
	ct := h.Get("Content-Type")
	switch {
	case strings.Contains(ct, "application/octet-stream"),
		strings.Contains(ct, "application/force-download"),
		strings.Contains(ct, "application/vnd.ms-excel"),
		strings.Contains(ct, "application/download"),
		strings.Contains(ct, "application/zip"),
		strings.Contains(ct, "application/pdf"),
		strings.Contains(ct, "image/"),
		strings.Contains(ct, "video/"),
		strings.Contains(ct, "multipart/form-data"):
		return true
	}
	return strings.Contains(h.Get("Content-Disposition"), "attachment")
}

// resolveTitle 解析系统模块:优先取业务 handler 显式设置的覆盖值;
// 否则用路由模板(低基数, 避开 path param)去掉前缀后的前两段作为模块定位(如 "system/user")。
func resolveTitle(c *gin.Context) string {
	if t := c.GetString(ctxOperTitleKey); t != "" {
		return t
	}
	route := strings.TrimPrefix(c.FullPath(), global.OPS_CONFIG.System.RouterPrefix)
	route = strings.Trim(route, "/")
	if route == "" {
		return ""
	}
	segs := strings.SplitN(route, "/", 3)
	if len(segs) >= 2 {
		return segs[0] + "/" + segs[1]
	}
	return segs[0]
}

// resolveBusinessType 解析操作类型:优先取业务 handler 显式设置的覆盖值;
// 否则按路径特征(export/import)与 HTTP 方法推导(POST 新增 / PUT,PATCH 修改 / DELETE 删除 / 其余 其它)。
func resolveBusinessType(c *gin.Context) string {
	if b := c.GetString(ctxOperBTypeKey); b != "" {
		return b
	}
	path := strings.ToLower(c.Request.URL.Path)
	switch {
	case strings.Contains(path, "export"):
		return bizExport
	case strings.Contains(path, "import"):
		return bizImport
	case strings.Contains(path, "clean"):
		return bizClean
	}
	switch c.Request.Method {
	case http.MethodPost:
		return bizInsert
	case http.MethodPut, http.MethodPatch:
		return bizUpdate
	case http.MethodDelete:
		return bizRemove
	default:
		return bizOther
	}
}

// claimsFromContext 仅读取 JWTAuth 已放入 gin context 的 claims, 不主动解析 token:
// 公开路由无 token, 主动解析会每请求产生一条无意义错误日志。未认证返回 nil。
func claimsFromContext(c *gin.Context) *systemReq.CustomClaims {
	if v, ok := c.Get("claims"); ok {
		if cl, ok := v.(*systemReq.CustomClaims); ok {
			return cl
		}
	}
	return nil
}
