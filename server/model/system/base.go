package system

// 状态码（char），与前端 EnableStatus 对齐：0 正常 / 1 停用。
//
// 审计基座 global.OPS_AUDIT_MODEL（内嵌 global.OPS_MODEL 时间戳）位于 server/global/common.go，
// 供对齐 RuoYi/前端 CommonRecord 的对外模型使用；system 包仅保留业务状态常量。
const (
	StatusEnable  = "0"
	StatusDisable = "1"
)
