package global

import (
	"fmt"
	"sync"

	"github.com/hllkk/devops-admin/server/config"
	"github.com/hllkk/devops-admin/server/utils/timer"

	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/server"
	"github.com/qiniu/qmgo"
	"github.com/redis/go-redis/v9"
	"github.com/songzhibin97/gkit/cache/local_cache"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

var (
	OPS_DB                  *gorm.DB
	OPS_DBList              map[string]*gorm.DB
	OPS_REDIS               redis.UniversalClient
	OPS_REDISList           map[string]redis.UniversalClient
	OPS_MONGO               *qmgo.QmgoClient
	OPS_CONFIG              config.Server
	OPS_VP                  *viper.Viper
	OPS_LOG                 *zap.Logger
	OPS_Timer               timer.Timer = timer.NewTimerTask()
	OPS_Concurrency_Control             = &singleflight.Group{}
	OPS_ROUTERS             gin.RoutesInfo
	OPS_ACTIVE_DBNAME       *string
	OPS_MCP_SERVER          *server.MCPServer
	BlackCache              local_cache.Cache
	lock                    sync.RWMutex
)

// GetGlobalDBByDBName 通过名称获取db list中的db
func GetGlobalDBByDBName(dbname string) *gorm.DB {
	lock.RLock()
	defer lock.RUnlock()
	return OPS_DBList[dbname]
}

// MustGetGlobalDBByDBName 通过名称获取db 如果不存在则panic
func MustGetGlobalDBByDBName(dbname string) *gorm.DB {
	lock.RLock()
	defer lock.RUnlock()
	db, ok := OPS_DBList[dbname]
	if !ok || db == nil {
		panic("db no init")
	}
	return db
}

func GetRedis(name string) redis.UniversalClient {
	redis, ok := OPS_REDISList[name]
	if !ok || redis == nil {
		panic(fmt.Sprintf("redis `%s` no init", name))
	}
	return redis
}
