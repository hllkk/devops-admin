package utils

import (
	"sync"

	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/hllkk/devops-admin/server/global"
	"go.uber.org/zap"
)

var (
	syncedCachedEnforcer *casbin.SyncedCachedEnforcer
	once                 sync.Once
)

// casbinModelText RBAC model 文本: sub=角色ID、obj=接口路径(去 RouterPrefix)、act=HTTP方法。
// matcher 用 keyMatch2 路径通配(/system/user/* 匹配子路径) + act 通配(p.act=="*" 放行所有方法,菜单级策略用)。
// 抽为包级变量便于测试构造内存 enforcer 验证 matcher 行为,避免与实现脱钩。
var casbinModelText = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = r.sub == p.sub && keyMatch2(r.obj,p.obj) && (r.act == p.act || p.act == "*")
`

// GetCasbin 获取casbin实例
func GetCasbin() *casbin.SyncedCachedEnforcer {
	once.Do(func() {
		a, err := gormadapter.NewAdapterByDB(global.OPS_DB)
		if err != nil {
			zap.L().Error("适配数据库失败请检查 casbin 配置和数据表", zap.Error(err))
			return
		}
		m, err := model.NewModelFromString(casbinModelText)
		if err != nil {
			zap.L().Error("字符串加载模型失败", zap.Error(err))
			return
		}
		enforcer, err := casbin.NewSyncedCachedEnforcer(m, a)
		if err != nil {
			zap.L().Error("casbin enforcer 初始化失败", zap.Error(err))
			return
		}
		enforcer.SetExpireTime(60 * 60)
		if err = enforcer.LoadPolicy(); err != nil {
			zap.L().Error("casbin 策略加载失败", zap.Error(err))
			return
		}
		syncedCachedEnforcer = enforcer
	})
	return syncedCachedEnforcer
}
