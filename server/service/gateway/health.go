package gateway

import (
	"context"
	"fmt"
	"time"

	"gorm.io/datatypes"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/gateway"
	gatewayResp "github.com/hllkk/devops-admin/server/model/gateway/response"
	"github.com/hllkk/devops-admin/server/utils/litellm"
)

// HealthService 健康检查(P3)。对 AIHelms 的三处刻意偏离(均避其坑)：
//   - 补「基础组件」卡(LiteLLM/PG/Redis ping)——AIHelms 探测强依赖 LiteLLM，
//     最关键依赖自身反而无监控；
//   - 模型部署健康=真实探测落库(经 LiteLLM 数据面 ping)——AIHelms 纯配置判断
//     (有启用部署即"健康")与真实可用性无关，命名误导；
//   - 不做 Docker 环境自检(AIHelms 在请求进程里跑 subprocess)——server 跑在容器内
//     默认摸不到 docker.sock，业务容器监控无从谈起；数据回流新鲜度替代其"环境"语义。
type HealthService struct{}

// 数据回流新鲜度阈值(分钟)：游标 10 分钟内推进=healthy(回流任务 5 分钟一跑)，
// 60 分钟内=warning，超时/无记录=danger/unknown。
const (
	freshnessHealthyMinutes = 10
	freshnessWarningMinutes = 60
)

// GetHealthSummary 四卡汇总+三块明细。MCP/部署读巡检落库，组件与新鲜度即时探测。
func (s *HealthService) GetHealthSummary(ctx context.Context) (gatewayResp.HealthSummary, error) {
	var sum gatewayResp.HealthSummary
	sum.CheckedAt = time.Now().Local().Format("2006-01-02 15:04:05")

	// ── MCP 上游卡+明细(现有巡检落库,分母=is_active) ──
	type statusCnt struct {
		Status string
		Count  int
	}
	var mcpCnts []statusCnt
	if err := global.OPS_DB.WithContext(ctx).Model(&gateway.MCPServer{}).
		Select("health_status AS status, COUNT(*) AS count").
		Where("is_active = ?", true).Group("health_status").Scan(&mcpCnts).Error; err != nil {
		return sum, err
	}
	for _, c := range mcpCnts {
		sum.Mcp = fillHealthCard(sum.Mcp, c.Status, c.Count)
	}
	var mcps []gateway.MCPServer
	if err := global.OPS_DB.WithContext(ctx).Select("mcp_server_id, name, server_name, health_status, last_health_check, health_check_error").
		Where("is_active = ?", true).Order("health_status ASC, name ASC").Find(&mcps).Error; err != nil {
		return sum, err
	}
	sum.McpItems = make([]gatewayResp.HealthMcpItem, 0, len(mcps))
	for i := range mcps {
		sum.McpItems = append(sum.McpItems, gatewayResp.HealthMcpItem{
			McpServerId: mcps[i].McpServerId, Name: mcps[i].Name, ServerName: mcps[i].ServerName,
			HealthStatus: mcps[i].HealthStatus, LastHealthCheck: fmtHealthTime(mcps[i].LastHealthCheck),
			HealthCheckError: mcps[i].HealthCheckError,
		})
	}

	// ── 模型部署卡+明细(路由组级巡检落库,分母=is_active 部署) ──
	var depCnts []statusCnt
	if err := global.OPS_DB.WithContext(ctx).Model(&gateway.ModelDeployment{}).
		Select("health_status AS status, COUNT(*) AS count").
		Where("is_active = ?", true).Group("health_status").Scan(&depCnts).Error; err != nil {
		return sum, err
	}
	for _, c := range depCnts {
		sum.Deployment = fillHealthCard(sum.Deployment, c.Status, c.Count)
	}
	type depRow struct {
		DeploymentId     int64
		ModelName        string
		ModelKey         string
		DeployName       string
		HealthStatus     string
		LastHealthCheck  *time.Time
		HealthCheckError string
	}
	var depRows []depRow
	if err := global.OPS_DB.WithContext(ctx).Table("gateway_model_deployment d").
		Select(`d.deployment_id, COALESCE(NULLIF(m.name,''), m.model_key) AS model_name, m.model_key,
			d.deploy_name, d.health_status, d.last_health_check, d.health_check_error`).
		Joins("LEFT JOIN gateway_model m ON m.model_id = d.model_id AND m.deleted_at IS NULL").
		Where("d.is_active = ? AND d.deleted_at IS NULL", true).
		Order("d.health_status ASC, model_name ASC").Scan(&depRows).Error; err != nil {
		return sum, err
	}
	sum.DeploymentItems = make([]gatewayResp.HealthDeploymentItem, 0, len(depRows))
	for _, r := range depRows {
		sum.DeploymentItems = append(sum.DeploymentItems, gatewayResp.HealthDeploymentItem{
			DeploymentId: r.DeploymentId, ModelName: r.ModelName, ModelKey: r.ModelKey, DeployName: r.DeployName,
			HealthStatus: r.HealthStatus, LastHealthCheck: fmtHealthTime(r.LastHealthCheck),
			HealthCheckError: r.HealthCheckError,
		})
	}

	// ── 基础组件(即时探测) ──
	sum.Components = s.probeComponents(ctx)

	// ── 数据回流新鲜度(游标 updated_at) ──
	sum.Freshness = s.probeFreshness(ctx)
	return sum, nil
}

// HealthCheckAllDeployments 部署健康巡检：按模型路由组探测(同 model_key 的部署共享
// LiteLLM LB 组，网关级 ping 无法定位组内单节点——单节点故障由 LiteLLM allowed_fails/
// cooldown 兜底)，结论写组内全部启用部署。单机模式(未配置 LiteLLM)跳过。
func (s *HealthService) HealthCheckAllDeployments(ctx context.Context) (int, error) {
	cli := litellm.Default()
	if cli == nil {
		return 0, nil
	}
	type groupRow struct {
		ModelId      int64
		ModelKey     string
		Category     string
		Capabilities []byte
	}
	var groups []groupRow
	if err := global.OPS_DB.WithContext(ctx).Table("gateway_model m").
		Select(`DISTINCT m.model_id, m.model_key, m.category, m.capabilities`).
		Joins("JOIN gateway_model_deployment d ON d.model_id = m.model_id AND d.is_active = TRUE AND d.deleted_at IS NULL").
		Where("m.is_active = ? AND m.deleted_at IS NULL", true).
		Scan(&groups).Error; err != nil {
		return 0, err
	}

	checked := 0
	for _, g := range groups {
		if ctx.Err() != nil {
			return checked, ctx.Err()
		}
		path, body := buildModelProbe(gateway.Model{
			ModelId: g.ModelId, ModelKey: g.ModelKey, Category: g.Category,
			Capabilities: datatypes.JSON(g.Capabilities),
		})
		healthStatus, healthErr := s.probeRoute(ctx, cli, path, body)
		now := time.Now()
		if err := global.OPS_DB.WithContext(ctx).Model(&gateway.ModelDeployment{}).
			Where("model_id = ? AND is_active = ?", g.ModelId, true).
			Updates(map[string]any{
				"health_status": healthStatus, "last_health_check": now, "health_check_error": healthErr,
			}).Error; err != nil {
			return checked, err
		}
		checked++
	}
	return checked, nil
}

// probeRoute 单路由组探测：传输层错误/HTTP>=400 均为 unhealthy，信息脱敏。
func (s *HealthService) probeRoute(ctx context.Context, cli *litellm.Client, path string, body map[string]any) (status, healthErr string) {
	respStatus, respBody, err := cli.RawPost(ctx, path, body)
	if err != nil {
		return gateway.DeploymentHealthUnhealthy, SanitizeTechnicalDetail(err.Error())
	}
	if respStatus < 400 {
		return gateway.DeploymentHealthHealthy, ""
	}
	_, msg := classifyUpstreamError(respStatus)
	return gateway.DeploymentHealthUnhealthy, SanitizeTechnicalDetail(fmt.Sprintf("%s: %s", msg, string(respBody)))
}

// probeComponents 基础组件即时探测：LiteLLM(未配置=unknown)/PostgreSQL/Redis(未配置=unknown)。
func (s *HealthService) probeComponents(ctx context.Context) []gatewayResp.HealthComponentItem {
	items := make([]gatewayResp.HealthComponentItem, 0, 3)

	// LiteLLM
	if cli := litellm.Default(); cli == nil {
		items = append(items, gatewayResp.HealthComponentItem{Name: "litellm", Status: gateway.MCPHealthUnknown, Message: "未配置(单机模式)"})
	} else {
		start := time.Now()
		if err := cli.Ping(ctx); err != nil {
			items = append(items, gatewayResp.HealthComponentItem{Name: "litellm", Status: gateway.MCPHealthUnhealthy,
				LatencyMs: msSince(start), Message: SanitizeTechnicalDetail(err.Error())})
		} else {
			items = append(items, gatewayResp.HealthComponentItem{Name: "litellm", Status: gateway.MCPHealthHealthy, LatencyMs: msSince(start)})
		}
	}

	// PostgreSQL
	start := time.Now()
	if err := global.OPS_DB.WithContext(ctx).Exec("SELECT 1").Error; err != nil {
		items = append(items, gatewayResp.HealthComponentItem{Name: "postgresql", Status: gateway.MCPHealthUnhealthy,
			LatencyMs: msSince(start), Message: err.Error()})
	} else {
		items = append(items, gatewayResp.HealthComponentItem{Name: "postgresql", Status: gateway.MCPHealthHealthy, LatencyMs: msSince(start)})
	}

	// Redis
	if global.OPS_REDIS == nil {
		items = append(items, gatewayResp.HealthComponentItem{Name: "redis", Status: gateway.MCPHealthUnknown, Message: "未配置"})
	} else {
		start := time.Now()
		if err := global.OPS_REDIS.Ping(ctx).Err(); err != nil {
			items = append(items, gatewayResp.HealthComponentItem{Name: "redis", Status: gateway.MCPHealthUnhealthy,
				LatencyMs: msSince(start), Message: err.Error()})
		} else {
			items = append(items, gatewayResp.HealthComponentItem{Name: "redis", Status: gateway.MCPHealthHealthy, LatencyMs: msSince(start)})
		}
	}
	return items
}

// probeFreshness 数据回流新鲜度：读 llm_logs/mcp_logs 两游标的 updated_at
//(每 5 分钟 tick 无论有无新数据都会推进，是回流管道活性的可靠信号；
// 区别于日志行时间——静默时段无日志不应误报)。
func (s *HealthService) probeFreshness(ctx context.Context) gatewayResp.HealthFreshness {
	f := gatewayResp.HealthFreshness{
		ThresholdMinutes: freshnessHealthyMinutes, StaleWarnMinutes: freshnessWarningMinutes,
		Status: gateway.MCPHealthUnknown,
	}
	var states []gateway.SyncState
	if err := global.OPS_DB.WithContext(ctx).
		Where("key IN ?", []string{syncStateKey, syncStateKeyMcp}).Find(&states).Error; err != nil {
		return f
	}
	var latest *time.Time
	for i := range states {
		at := states[i].UpdatedAt.Local()
		if states[i].Key == syncStateKey {
			f.LlmSyncAt = at.Format("2006-01-02 15:04:05")
		} else {
			f.McpSyncAt = at.Format("2006-01-02 15:04:05")
		}
		if latest == nil || states[i].UpdatedAt.After(*latest) {
			t := states[i].UpdatedAt
			latest = &t
		}
	}
	if latest == nil {
		return f // 无记录=unknown(回流任务从未跑过)
	}
	f.LastSyncAt = latest.Local().Format("2006-01-02 15:04:05")
	minutes := time.Since(*latest).Minutes()
	switch {
	case minutes <= freshnessHealthyMinutes:
		f.Status = gateway.MCPHealthHealthy
	case minutes <= freshnessWarningMinutes:
		f.Status = "warning"
	default:
		f.Status = "danger"
	}
	return f
}

// fillHealthCard 按状态累加卡片计数(unknown 兜底)。
func fillHealthCard(card gatewayResp.HealthCard, status string, n int) gatewayResp.HealthCard {
	card.Total += n
	switch status {
	case gateway.MCPHealthHealthy:
		card.Healthy += n
	case gateway.MCPHealthUnhealthy:
		card.Unhealthy += n
	default:
		card.Unknown += n
	}
	return card
}

// fmtHealthTime 巡检时间格式化(本地时区,nil=空串)。
func fmtHealthTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Local().Format("2006-01-02 15:04:05")
}

// msSince 毫秒耗时。
func msSince(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}
