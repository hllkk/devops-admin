package initialize

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/hllkk/devops-admin/server/docs"
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/middleware"
	"github.com/hllkk/devops-admin/server/router"
)

type justFilesFilesystem struct {
	fs http.FileSystem
}

func (fs justFilesFilesystem) Open(name string) (http.File, error) {
	f, err := fs.fs.Open(name)
	if err != nil {
		return nil, err
	}

	stat, err := f.Stat()
	if err == nil && stat.IsDir() {
		return nil, os.ErrPermission
	}

	return f, nil
}

// 初始化总路由

func Routers() *gin.Engine {
	Router := gin.New()

	// 信任代理：空则不信任任何代理（ClientIP=RemoteAddr，防 X-Forwarded-For 伪造绕过 IP 锁定/限流）；
	// 非空则仅信任指定 CIDR/IP，多层反代下正确解析真实客户端 IP
	if tp := global.OPS_CONFIG.System.TrustedProxies; len(tp) > 0 {
		if err := Router.SetTrustedProxies(tp); err != nil {
			global.OPS_LOG.Error("SetTrustedProxies 失败，回退为不信任任何代理: " + err.Error())
			_ = Router.SetTrustedProxies(nil)
		} else {
			global.OPS_LOG.Info("已配置可信代理，ClientIP 将从可信链路解析: " + strings.Join(tp, ","))
		}
	} else {
		_ = Router.SetTrustedProxies(nil)
		global.OPS_LOG.Info("未配置可信代理（trusted-proxies 为空），ClientIP=直连 RemoteAddr；反代部署请配置内网网段以正确解析真实客户端 IP")
	}

	// 使用自定义的 Recovery 中间件，记录 panic 并入库
	Router.Use(middleware.GinRecovery(true))
	if gin.Mode() == gin.DebugMode {
		Router.Use(gin.Logger())
	}
	Router.Use(middleware.CorsByRules()) // 跨域：按 cors 白名单规则放行；strict-whitelist 模式拒绝未匹配 Origin

	// 如果想要不使用nginx代理前端网页，可以修改 web/.env.production 下的
	// VUE_APP_BASE_API = /
	// VUE_APP_BASE_PATH = http://localhost
	// 然后执行打包命令 npm run build。在打开下面3行注释
	// Router.StaticFile("/favicon.ico", "./dist/favicon.ico")
	// Router.Static("/assets", "./dist/assets")   // dist里面的静态资源
	// Router.StaticFile("/", "./dist/index.html") // 前端网页入口页面

	Router.StaticFS(global.OPS_CONFIG.Local.StorePath, justFilesFilesystem{http.Dir(global.OPS_CONFIG.Local.StorePath)})
	// 跨域配置已由上方 CorsByRules() 处理，allow-all 模式等同原 middleware.Cors()
	docs.SwaggerInfo.BasePath = global.OPS_CONFIG.System.RouterPrefix
	Router.GET(global.OPS_CONFIG.System.RouterPrefix+"/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	global.OPS_LOG.Info("register swagger handler")

	// 路由组：按认证级别分为三级
	PublicGroup := Router.Group(global.OPS_CONFIG.System.RouterPrefix)
	PrivateGroup := Router.Group(global.OPS_CONFIG.System.RouterPrefix)

	PrivateGroup.Use(middleware.JWTAuth())

	// 健康监测
	PublicGroup.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, "ok")
	})

	// 模块自注册 — 每个模块实现 router.ModuleRouter 接口
	// 新增模块时：1) 实现三个方法  2) 加入此列表
	modules := []router.ModuleRouter{
		&router.RouterGroupApp.System,
	}
	for _, m := range modules {
		m.RegisterPublic(PublicGroup)
		m.RegisterPrivate(PrivateGroup)
	}

	// 注册业务路由（尚未实现 ModuleRouter 的旧模块）
	initBizRouter(PrivateGroup, PublicGroup)

	global.OPS_ROUTERS = Router.Routes()

	global.OPS_LOG.Info("router register success")
	return Router
}
