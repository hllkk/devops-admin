package response

// AssetOverview 服务器概览统计(模块顶层概览页,卡片数据)。
// ByType/ByEnv 统计未软删全量(含停用);MonitorStatus/AgentStatus 只统计启用中
// (停用资产不采集,状态恒 unknown 无统计意义)。
type AssetOverview struct {
	Total         int64            `json:"total" example:"150"`       // 资产总数(未软删)
	ActiveTotal   int64            `json:"activeTotal" example:"142"` // 启用中资产数
	ByType        map[string]int64 `json:"byType"`                    // 各类型数量(physical/vm/docker_host/db_instance/net_device)
	MonitorStatus map[string]int64 `json:"monitorStatus"`             // 监控状态数量(启用中: online/offline/unknown)
	AgentStatus   map[string]int64 `json:"agentStatus"`               // Agent状态数量(启用中: none/installing/running/lost)
	ByEnv         map[string]int64 `json:"byEnv"`                     // 环境分布(未软删, env 为空的归 unknown)
}
