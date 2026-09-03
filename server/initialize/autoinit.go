package initialize

import (
	"os"

	"github.com/hllkk/devops-admin/server/service/system"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// DockerAutoInit Docker 环境启动期自动初始化：
// DOCKER_ENV=true 且尚未初始化且 config 完整且已设 INIT_ADMIN_PASSWORD 时，
// 用挂载 config.yaml（敏感项经 env 覆盖）自动建库/建表/建管理员，
// 让 compose up 后系统直接就绪，无需再经 /init 页面人工触发。
// 初始化幂等（有用户数据即跳过）；失败不退出进程——HTTP 服务照常启动，
// 可经 /init 页面一键初始化或 POST /init/autoInitDB 重试。
func DockerAutoInit() {
	if os.Getenv("DOCKER_ENV") != "true" {
		return
	}
	skipReason, err := (&system.InitDBService{}).AutoInitDBFromDockerConfig()
	switch {
	case err != nil:
		logger.Bg().Mod("init").Err(err).Error("启动期自动初始化失败，可经 /init 页面或 POST /init/autoInitDB 重试")
	case skipReason != "":
		logger.Bg().Mod("init").Info("跳过启动期自动初始化：" + skipReason)
	default:
		logger.Bg().Mod("init").Info("启动期自动初始化成功（admin / INIT_ADMIN_PASSWORD）")
	}
}
