package global

// Version/BuildTime 为 var（const 无法被 ldflags -X 注入）：
// 发布构建经 ldflags 注入（deploy/docker-prod/Dockerfile.server），
// 源码构建保留下方默认值；在线升级的版本比对以本值为准
var (
	// Version 当前版本号
	Version = "v1.0.7"
	// BuildTime 构建时间（RFC3339，裸构建为 unknown）
	BuildTime = "unknown"
)

// 应用静态信息
const (
	// AppName 应用名称
	AppName = "Devops-Admin"
	// Description 应用描述
	Description = "面向 AI 应用开发与智能体集成的全栈开发基础平台"
)
