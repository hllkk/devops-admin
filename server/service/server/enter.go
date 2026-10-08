package server

// ServiceGroup 服务器模块业务服务组(对齐 gateway 组的聚合方式)。
// slice1：统一资产 + 采集凭据；后续 slice 按分期规划追加(agent 安装/指标管道/拨测/IPMI/Docker/DB/SNMP)。
type ServiceGroup struct {
	AssetService
	CredentialService
}
