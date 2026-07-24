package system

import (
	"os"

	"github.com/gin-gonic/gin"
)

type InitRouter struct{}

func (s *InitRouter) InitInitRouter(Router *gin.RouterGroup) {
	initRouter := Router.Group("init")
	{
		initRouter.POST("checkdb", dbApi.CheckDB)       // 检测是否需要初始化数据库
		initRouter.POST("autoInitDB", dbApi.AutoInitDB) // Docker 环境自动初始化（用 config + env）

		// 手动初始化向导仅在非 Docker 环境注册。Docker 生产已由 autoInitDB 用 config +
		// INIT_ADMIN_PASSWORD 自动初始化；手填向导(initdb/conn-test/ping-redis)会接受调用方
		// 指定的任意 DB/Redis 地址，在"server 已起、尚未 autoInitDB"的窗口期构成
		// SSRF / 内网端口探测 / 库劫持面，故生产(DOCKER_ENV=true)屏蔽。
		if os.Getenv("DOCKER_ENV") != "true" {
			initRouter.POST("initdb", dbApi.InitDB)        // 初始化数据库（向导手填）
			initRouter.POST("conn-test", dbApi.PingDB)     // 数据库连接测试
			initRouter.POST("ping-redis", dbApi.PingRedis) // Redis 连接测试
		}
	}
}
