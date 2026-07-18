package initialize

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/docs"
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/middleware"
	"github.com/hllkk/devops-admin/server/router"
	"github.com/hllkk/devops-admin/server/service"
	"github.com/hllkk/devops-admin/server/utils/logger"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
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
	// RequestMeta 必须最先：保证 panic 日志与 X-Request-Id 响应头都带 request_id
	Router.Use(middleware.RequestMeta())
	// 使用自定义的 Recovery 中间件，记录 panic 并入库
	Router.Use(middleware.GinRecovery(true))
	// 全局访问日志 + 唯一 body/resp 捕获点（供 OperationRecord 复用）
	Router.Use(middleware.AccessLog())
	// 启动操作日志异步写入协程（幂等），供 OperationRecord 落表
	service.ServiceGroupApp.SystemServiceGroup.SysOperLogService.StartWriter()
	if gin.Mode() == gin.DebugMode {
		Router.Use(gin.Logger())
	}

	systemRouter := router.RouterGroupApp.System
	// mediaRouter := router.RouterGroupApp.Media
	// 如果想要不使用nginx代理前端网页，可以修改 web/.env.production 下的
	// VUE_APP_BASE_API = /
	// VUE_APP_BASE_PATH = http://localhost
	// 然后执行打包命令 npm run build。在打开下面3行注释
	// Router.StaticFile("/favicon.ico", "./dist/favicon.ico")
	// Router.Static("/assets", "./dist/assets")   // dist里面的静态资源
	// Router.StaticFile("/", "./dist/index.html") // 前端网页入口页面

	Router.StaticFS(global.OPS_CONFIG.Local.StorePath, justFilesFilesystem{http.Dir(global.OPS_CONFIG.Local.StorePath)})
	// Router.Use(middleware.LoadTls())  // 如果需要使用https 请打开此中间件 然后前往 core/server.go 将启动模式 更变为 Router.RunTLS("端口","你的cre/pem文件","你的key文件")
	// 跨域，如需跨域可以打开下面的注释
	// Router.Use(middleware.Cors()) // 直接放行全部跨域请求
	// Router.Use(middleware.CorsByRules()) // 按照配置的规则放行跨域请求
	docs.SwaggerInfo.BasePath = global.OPS_CONFIG.System.RouterPrefix
	Router.GET(global.OPS_CONFIG.System.RouterPrefix+"/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	logger.Bg().Mod("system").Info("register swagger handler")
	// 方便统一添加路由组前缀 多服务器上线使用

	PublicGroup := Router.Group(global.OPS_CONFIG.System.RouterPrefix)
	PrivateGroup := Router.Group(global.OPS_CONFIG.System.RouterPrefix)

	// OperationRecord 置于 JWTAuth 之后:operName 已可用,且可记录授权拒绝(403)与业务结果
	PrivateGroup.Use(middleware.JWTAuth()).
		Use(middleware.OperationRecord()).
		Use(middleware.MustChangePwdGuard()).
		// Use(middleware.CasbinHandler()).
		Use(middleware.DataScope())

	{
		// 健康监测
		PublicGroup.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, "ok")
		})
	}
	{
		systemRouter.InitInitRouter(PublicGroup) // 自动初始化相关
		systemRouter.InitBaseRouter(PublicGroup) // 注册基础功能路由(login POST + captcha GET,不做鉴权)
	}
	{

		systemRouter.InitAuthRouter(PrivateGroup)  // 鉴权路由(getUserInfo/logout/refreshToken)
		systemRouter.InitRouteRouter(PrivateGroup) // 路由下发(/route/getConstantRoutes)
		systemRouter.InitDictRouter(PrivateGroup)  // 字典管理(/system/dict/type/*、/system/dict/data/*)
	}

	{
		// systemRouter.InitApiRouter(PrivateGroup, PublicGroup)               // 注册功能api路由
		// systemRouter.InitJwtRouter(PrivateGroup)                            // jwt相关路由
		// systemRouter.InitUserRouter(PrivateGroup)                           // 注册用户路由
		// systemRouter.InitMenuRouter(PrivateGroup)                           // 注册menu路由
		// systemRouter.InitSystemRouter(PrivateGroup)                         // system相关路由
		// systemRouter.InitSysVersionRouter(PrivateGroup)                     // 发版相关路由
		// systemRouter.InitCasbinRouter(PrivateGroup)                         // 权限相关路由
		// systemRouter.InitRoleRouter(PrivateGroup)                      // 注册角色路由
		// systemRouter.InitSysDepartmentRouter(PrivateGroup)                  // 注册部门路由
		// systemRouter.InitSysPositionRouter(PrivateGroup)                    // 注册岗位路由
		// systemRouter.InitSysDataAccessLogRouter(PrivateGroup)               // 数据权限审计日志
		// systemRouter.InitSysDictionaryRouter(PrivateGroup)                  // 字典管理
		// systemRouter.InitSysOperationRecordRouter(PrivateGroup)             // 操作记录
		// systemRouter.InitSysDictionaryDetailRouter(PrivateGroup)            // 字典详情管理
		// systemRouter.InitRoleBtnRouterRouter(PrivateGroup)             // 按钮权限管理
		// systemRouter.InitSysExportTemplateRouter(PrivateGroup, PublicGroup) // 导出模板
		// systemRouter.InitSysParamsRouter(PrivateGroup, PublicGroup)         // 参数管理
		// systemRouter.InitSysErrorRouter(PrivateGroup, PublicGroup)          // 错误日志
		// systemRouter.InitLoginLogRouter(PrivateGroup)                       // 登录日志
		// systemRouter.InitSecurityConfigRouter(PrivateGroup)                 // 安全配置
		// systemRouter.InitApiTokenRouter(PrivateGroup)                       // apiToken签发
		// systemRouter.InitTimedTaskRouter(PrivateGroup)                      // 定时任务
		// exampleRouter.InitCustomerRouter(PrivateGroup)                      // 客户路由
		// mediaRouter.InitFileUploadAndDownloadRouter(PrivateGroup)           // 文件上传下载功能路由
		// mediaRouter.InitAttachmentCategoryRouterRouter(PrivateGroup)        // 媒体分类
		// mediaRouter.InitMediaUploadRouter(PrivateGroup)                     // 大文件上传
	}

	//插件路由安装
	// InstallPlugin(PrivateGroup, PublicGroup, Router)

	// 注册业务路由
	initBizRouter(PrivateGroup, PublicGroup)

	global.OPS_ROUTERS = Router.Routes()

	logger.Bg().Mod("system").Info("router register success")
	return Router
}
