package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/hllkk/devops-admin/server/global"
	servermod "github.com/hllkk/devops-admin/server/model/server"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// prometheus.go Prometheus 集成层（pull 架构的数据面）：
//
//   平台资产表 ──HTTP SD──→ Prometheus ──pull──→ 150 台 node_exporter(:9100)
//        ↑                                        │
//        └──SyncMonitorStatus(定时刮 up 回写状态)  └──本文件代查 API(快照/趋势)
//
// 平台前端不直连 Prometheus：后端代查(instant/range query)投影成原有接口
// 形态(GetSnapshot/GetTrend)，前端零改动。告警(slice4)走 Alertmanager webhook 回流。

// PrometheusService Prometheus 查询与服务发现。
type PrometheusService struct{}

// promHTTPClient 查询客户端(短超时——查询快)。
var promHTTPClient = &http.Client{Timeout: 15 * time.Second}

// promRow Prometheus 查询结果行(instant 的 Value / range 的 Values 共用承载)。
type promRow struct {
	Metric map[string]string `json:"metric"`
	Value  [2]any            `json:"value"`  // instant: [ts, "val"]
	Values [][2]any          `json:"values"` // range: [[ts,"val"],...]
}

// promQueryResult Prometheus API 响应结构(部分投影)。
type promQueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string    `json:"resultType"`
		Result     []promRow `json:"result"`
	} `json:"data"`
}

// AgentSnapshot 快照出网视图(保持自研时代字段形态,前端零改动;数据源换 Prometheus)。
type AgentSnapshot struct {
	CpuPercent     float64   `json:"cpuPercent"`
	Loadavg1       float64   `json:"loadavg1"`
	MemTotalKB     uint64    `json:"memTotalKb"`
	MemAvailableKB uint64    `json:"memAvailableKb"`
	DiskTotalBytes uint64    `json:"diskTotalBytes"`
	DiskUsedBytes  uint64    `json:"diskUsedBytes"`
	NetInBps       float64   `json:"netInBps"`
	NetOutBps      float64   `json:"netOutBps"`
	UptimeSec      uint64    `json:"uptimeSec"`
	ReportedAt     time.Time `json:"reportedAt"`
	Stale          bool      `json:"stale"` // 查不到(Prometheus 未配置/未刮到)时 true
}

// MetricPoint 趋势出网点(保持自研时代字段形态)。
type MetricPoint struct {
	Ts         time.Time `json:"ts"`
	Cpu        float64   `json:"cpu"`
	CpuMax     float64   `json:"cpuMax"`
	MemPct     float64   `json:"memPct"`
	MemPctMax  float64   `json:"memPctMax"`
	DiskPct    float64   `json:"diskPct"`
	DiskPctMax float64   `json:"diskPctMax"`
	NetIn      float64   `json:"netIn"`
	NetInMax   float64   `json:"netInMax"`
	NetOut     float64   `json:"netOut"`
	NetOutMax  float64   `json:"netOutMax"`
	Loadavg1   float64   `json:"loadavg1"`
}

// MetricsTrend 趋势响应。
type MetricsTrend struct {
	Range  string        `json:"range"`
	Points []MetricPoint `json:"points"`
}

// instanceLabel 资产的 Prometheus instance 标签(IP:port)。
func (s *PrometheusService) instanceOf(asset servermod.Asset) string {
	port := global.OPS_CONFIG.ServerModule.Agent.ExporterPort
	if port <= 0 {
		port = exporterPortDefault
	}
	return fmt.Sprintf("%s:%d", asset.ManageIp, port)
}

// query instant query。返回结果集(可能空)。
func (s *PrometheusService) query(ctx context.Context, promQL string) ([]promRow, error) {
	var res promQueryResult
	if err := s.doAPI(ctx, "/api/v1/query", url.Values{"query": {promQL}}, &res); err != nil {
		return nil, err
	}
	return res.Data.Result, nil
}

// queryRange range query。返回结果集。
func (s *PrometheusService) queryRange(ctx context.Context, promQL string, start, end time.Time, step time.Duration) ([]promRow, error) {
	var res promQueryResult
	if err := s.doAPI(ctx, "/api/v1/query_range", url.Values{
		"query": {promQL},
		"start": {fmt.Sprintf("%d", start.Unix())},
		"end":   {fmt.Sprintf("%d", end.Unix())},
		"step":  {fmt.Sprintf("%ds", int(step.Seconds()))},
	}, &res); err != nil {
		return nil, err
	}
	return res.Data.Result, nil
}

// doAPI 调 Prometheus HTTP API。
func (s *PrometheusService) doAPI(ctx context.Context, path string, params url.Values, out any) error {
	base := strings.TrimRight(s.baseURL(), "/")
	if base == "" {
		return fmt.Errorf("Prometheus 未配置(server.prometheus.base-url)")
	}
	u := base + path + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := promHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求 Prometheus 失败: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Prometheus %d: %s", resp.StatusCode, string(raw))
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("Prometheus 响应解析失败: %w", err)
	}
	return nil
}

func (s *PrometheusService) baseURL() string {
	return global.OPS_CONFIG.ServerModule.Prometheus.BaseURL
}

// GetSnapshot 资产实时快照(instant query 各核心指标;instance 级过滤)。
func (s *PrometheusService) GetSnapshot(ctx context.Context, assetId int64) (AgentSnapshot, error) {
	snap := AgentSnapshot{Stale: true}
	var asset servermod.Asset
	if err := global.OPS_DB.WithContext(ctx).Where("asset_id = ?", assetId).First(&asset).Error; err != nil {
		return snap, err
	}
	inst := s.instanceOf(asset)
	up, err := s.query(ctx, fmt.Sprintf(`up{instance=%q}`, inst))
	if err != nil || len(up) == 0 {
		return snap, nil // 未刮到:Stale
	}
	if v := parseVal(up[0].Value); v != 1 {
		return snap, nil // up=0: 采集端失联
	}
	snap.Stale = false
	snap.ReportedAt = time.Now().UTC()

	scalar := func(promQL string) float64 {
		rows, err := s.query(ctx, promQL)
		if err != nil || len(rows) == 0 {
			return 0
		}
		return parseVal(rows[0].Value)
	}
	snap.CpuPercent = scalar(fmt.Sprintf(`100 - avg(rate(node_cpu_seconds_total{instance=%q,mode="idle"}[2m])) * 100`, inst))
	snap.Loadavg1 = scalar(fmt.Sprintf(`node_load1{instance=%q}`, inst))
	snap.MemTotalKB = uint64(scalar(fmt.Sprintf(`node_memory_MemTotal_bytes{instance=%q}`, inst)) / 1024)
	snap.MemAvailableKB = uint64(scalar(fmt.Sprintf(`node_memory_MemAvailable_bytes{instance=%q}`, inst)) / 1024)
	snap.DiskTotalBytes = uint64(scalar(fmt.Sprintf(`node_filesystem_size_bytes{instance=%q,mountpoint="/"}`, inst)))
	snap.DiskUsedBytes = uint64(scalar(fmt.Sprintf(`node_filesystem_size_bytes{instance=%q,mountpoint="/"} - node_filesystem_avail_bytes{instance=%q,mountpoint="/"}`, inst, inst)))
	snap.NetInBps = scalar(fmt.Sprintf(`sum(rate(node_network_receive_bytes_total{instance=%q,device!~"lo|veth.*"}[2m]))`, inst))
	snap.NetOutBps = scalar(fmt.Sprintf(`sum(rate(node_network_transmit_bytes_total{instance=%q,device!~"lo|veth.*"}[2m]))`, inst))
	snap.UptimeSec = uint64(scalar(fmt.Sprintf(`node_time_seconds{instance=%q} - node_boot_time_seconds{instance=%q}`, inst, inst)))
	return snap, nil
}

// GetTrend 趋势(query_range):1h step30s / 1d step1m / 7d step10m / 30d step1h。
// range 查询按 instant 维度查 6 条 PromQL 再按时间戳拼点(数据量 720 点内可控)。
func (s *PrometheusService) GetTrend(ctx context.Context, assetId int64, rng string) (MetricsTrend, error) {
	out := MetricsTrend{Range: rng, Points: []MetricPoint{}}
	var asset servermod.Asset
	if err := global.OPS_DB.WithContext(ctx).Where("asset_id = ?", assetId).First(&asset).Error; err != nil {
		return out, err
	}
	inst := s.instanceOf(asset)

	var step time.Duration
	var window time.Duration
	switch rng {
	case "1h":
		step, window = 30*time.Second, time.Hour
	case "1d":
		step, window = time.Minute, 24*time.Hour
	case "7d":
		step, window = 10*time.Minute, 7*24*time.Hour
	case "30d":
		step, window = time.Hour, 30*24*time.Hour
	default:
		return out, nil
	}
	now := time.Now().UTC()
	start := now.Add(-window)

	series := map[time.Time]*MetricPoint{}
	collect := func(promQL string, fill func(p *MetricPoint, v float64)) error {
		rows, err := s.queryRange(ctx, promQL, start, now, step)
		if err != nil {
			return err
		}
		for _, r := range rows {
			for _, v := range r.Values {
				ts := time.Unix(int64(toFloat(v[0])), 0).UTC()
				p, ok := series[ts]
				if !ok {
					p = &MetricPoint{Ts: ts}
					series[ts] = p
				}
				fill(p, parseVal(v))
			}
		}
		return nil
	}
	queries := []struct {
		q    string
		fill func(p *MetricPoint, v float64)
	}{
		{fmt.Sprintf(`100 - avg(rate(node_cpu_seconds_total{instance=%q,mode="idle"}[%s]) ) * 100`, inst, durStr(step)), func(p *MetricPoint, v float64) { p.Cpu = v; p.CpuMax = v }},
		{fmt.Sprintf(`100 * (1 - node_memory_MemAvailable_bytes{instance=%q} / node_memory_MemTotal_bytes{instance=%q})`, inst, inst), func(p *MetricPoint, v float64) { p.MemPct = v; p.MemPctMax = v }},
		{fmt.Sprintf(`100 * (1 - node_filesystem_avail_bytes{instance=%q,mountpoint="/"} / node_filesystem_size_bytes{instance=%q,mountpoint="/"})`, inst, inst), func(p *MetricPoint, v float64) { p.DiskPct = v; p.DiskPctMax = v }},
		{fmt.Sprintf(`sum(rate(node_network_receive_bytes_total{instance=%q,device!~"lo|veth.*"}[%s]))`, inst, durStr(step)), func(p *MetricPoint, v float64) { p.NetIn = v; p.NetInMax = v }},
		{fmt.Sprintf(`sum(rate(node_network_transmit_bytes_total{instance=%q,device!~"lo|veth.*"}[%s]))`, inst, durStr(step)), func(p *MetricPoint, v float64) { p.NetOut = v; p.NetOutMax = v }},
		{fmt.Sprintf(`node_load1{instance=%q}`, inst), func(p *MetricPoint, v float64) { p.Loadavg1 = v }},
	}
	for _, q := range queries {
		if err := collect(q.q, q.fill); err != nil {
			logger.WithCtx(ctx).Mod("server").Err(err).Field("assetId", assetId).Error("趋势查询失败(单指标,继续其余)")
		}
	}
	// 排序输出
	for ts := range series {
		out.Points = append(out.Points, *series[ts])
	}
	sort.Slice(out.Points, func(i, j int) bool { return out.Points[i].Ts.Before(out.Points[j].Ts) })
	return out, nil
}

// SDTargets HTTP SD 端点输出(Prometheus http_sd_configs 格式)。
// 目标=已安装采集端且启用中的资产;asset_id 标签便于告警回流时归因。
func (s *PrometheusService) SDTargets(ctx context.Context) []map[string]any {
	var assets []servermod.Asset
	if err := global.OPS_DB.WithContext(ctx).
		Where("is_active = ? AND agent_status IN ?", true, []string{servermod.AgentStatusRunning, servermod.AgentStatusLost}).
		Find(&assets).Error; err != nil {
		return nil
	}
	out := make([]map[string]any, 0, len(assets))
	for _, a := range assets {
		if a.ManageIp == "" {
			continue
		}
		out = append(out, map[string]any{
			"targets": []string{s.instanceOf(a)},
			"labels": map[string]string{
				"job":      "node",
				"asset_id": fmt.Sprintf("%d", a.AssetId),
			},
		})
	}
	return out
}

// SyncMonitorStatus 定时刮 up 指标回写资产状态:
// up=1 → monitor online + 采集端 running;up=0 → monitor offline + 采集端 lost;
// 无 instance 行(未刮到) → monitor unknown。SyncMonitorStatus 为包级入口(timer 注册用)。
func (s *PrometheusService) SyncMonitorStatus(ctx context.Context) error {
	base := s.baseURL()
	if base == "" {
		return fmt.Errorf("Prometheus 未配置,跳过状态同步")
	}
	var assets []servermod.Asset
	if err := global.OPS_DB.WithContext(ctx).
		Where("is_active = ? AND agent_status <> ?", true, servermod.AgentStatusNone).
		Find(&assets).Error; err != nil {
		return err
	}
	if len(assets) == 0 {
		return nil
	}
	rows, err := s.query(ctx, `up{job="node"}`)
	if err != nil {
		return err
	}
	upMap := make(map[string]float64, len(rows))
	for _, r := range rows {
		if inst, ok := r.Metric["instance"]; ok {
			upMap[inst] = parseVal(r.Value)
		}
	}
	for i := range assets {
		inst := s.instanceOf(assets[i])
		v, scraped := upMap[inst]
		updates := map[string]any{}
		switch {
		case !scraped:
			updates["monitor_status"] = servermod.MonitorStatusUnknown
		case v == 1:
			updates["monitor_status"] = servermod.MonitorStatusOnline
			updates["agent_status"] = servermod.AgentStatusRunning
		default:
			updates["monitor_status"] = servermod.MonitorStatusOffline
			updates["agent_status"] = servermod.AgentStatusLost
		}
		if err := global.OPS_DB.WithContext(ctx).Model(&servermod.Asset{}).
			Where("asset_id = ?", assets[i].AssetId).Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}

// SyncMonitorStatus 包级入口(定时任务注册)。
func SyncMonitorStatus(ctx context.Context) error {
	return (&PrometheusService{}).SyncMonitorStatus(ctx)
}

// ----------------------------------------------------------------------------
// 工具
// ----------------------------------------------------------------------------

// parseVal Prometheus 值("1.234" 字符串)→float64。
func parseVal(v [2]any) float64 {
	return toFloat(v[1])
}

func toFloat(v any) float64 {
	switch x := v.(type) {
	case string:
		var f float64
		_, _ = fmt.Sscanf(x, "%g", &f)
		return f
	case float64:
		return x
	default:
		return 0
	}
}

// durStr step → PromQL range selector 字符串(30s/1m/10m/1h)。
func durStr(d time.Duration) string {
	switch {
	case d >= time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d >= time.Minute:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	default:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
}
