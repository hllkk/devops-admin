package system

import (
	"github.com/gin-gonic/gin"
)

type InitRouter struct{}

func (s *InitRouter) InitInitRouter(Router *gin.RouterGroup) {
	initRouter := Router.Group("init")
	{
		initRouter.POST("initdb", dbApi.InitDB)         // 初始化数据库（向导手填）
		initRouter.POST("checkdb", dbApi.CheckDB)       // 检测是否需要初始化数据库
		initRouter.POST("autoInitDB", dbApi.AutoInitDB) // Docker 环境自动初始化（用 config + env）
		initRouter.POST("conn-test", dbApi.PingDB)      // 数据库连接测试
		initRouter.POST("ping-redis", dbApi.PingRedis)  // Redis 连接测试
	}
}
