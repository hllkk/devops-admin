package initialize

import (
	"fmt"
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
	// SetTrustedProxies:ClientIP() 解析 X-Forwarded-For 时仅信任此处配置的反代 CIDR。
	// 空/nil → 仅信任直连 peer(ClientIP 返回 RemoteAddr,忽略 XFF,最严格);反代部署须显式配置 CIDR,
	// 否则 ClientIP 全是反代 IP,限流/登录日志/安全锁定 key 失真。须在任何中间件前设置。
	if err := Router.SetTrustedProxies(global.OPS_CONFIG.System.TrustedProxies); err != nil {
		panic(fmt.Errorf("SetTrustedProxies 失败: %w", err))
	}
	// RequestMeta 必须最先：保证 panic 日志与 X-Request-Id 响应头都带 request_id
	Router.Use(middleware.RequestMeta())
	// 使用自定义的 Recovery 中间件，记录 panic 并入库
	Router.Use(middleware.GinRecovery(true))
	// SSE 专用组:必须在 AccessLog 之前注册,绕过 captureWriter(缓冲会破坏流式 Flusher 并致内存膨胀)。
	// 仅挂 JWTAuth(httpOnly cookie 鉴权),不套 OperationRecord/AccessLog/Timeout。路径对齐前端 /resource/sse。
	{
		sseGroup := Router.Group(global.OPS_CONFIG.System.RouterPrefix)
		sseGroup.Use(middleware.JWTAuth())
		router.RouterGroupApp.System.InitNoticeSSERouter(sseGroup)
		router.RouterGroupApp.System.InitTimedTaskSSERouter(sseGroup)
	}
	// 全局访问日志 + 唯一 body/resp 捕获点（供 OperationRecord 复用）
	Router.Use(middleware.AccessLog())
	// 启动操作日志异步写入协程（幂等），供 OperationRecord 落表
	service.ServiceGroupApp.SystemServiceGroup.SysOperLogService.StartWriter()
	if gin.Mode() == gin.DebugMode {
		Router.Use(gin.Logger())
	}

	systemRouter := router.RouterGroupApp.System
	gatewayRouter := router.RouterGroupApp.Gateway
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
	// 跨域:当前同源部署(前后端经 nginx 同域反代)无需启用 CORS。
	// 若改为跨域部署,务必用 CorsByRules()(按 strict-whitelist 白名单放行);
	// 不要启用反射式的 Cors()——它回显任意 Origin 且 Allow-Credentials=true,在 httpOnly cookie 模式下有风险。
	// Router.Use(middleware.CorsByRules())
	// Swagger 仅在非 release 模式注册：生产（GIN_MODE=release）下关闭，避免未授权暴露全部接口文档
	if gin.Mode() != gin.ReleaseMode {
		docs.SwaggerInfo.BasePath = global.OPS_CONFIG.System.RouterPrefix
		Router.GET(global.OPS_CONFIG.System.RouterPrefix+"/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		logger.Bg().Mod("system").Info("register swagger handler")
	}
	// 方便统一添加路由组前缀 多服务器上线使用

	PublicGroup := Router.Group(global.OPS_CONFIG.System.RouterPrefix)
	PrivateGroup := Router.Group(global.OPS_CONFIG.System.RouterPrefix)

	// OperationRecord 置于 JWTAuth 之后:operName 已可用,且可记录授权拒绝(403)与业务结果
	PrivateGroup.Use(middleware.JWTAuth()).
		Use(middleware.OperationRecord()).
		Use(middleware.MustChangePwdGuard()).
		Use(middleware.CasbinHandler()).
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

		systemRouter.InitAuthRouter(PrivateGroup)                  // 鉴权路由(getUserInfo/logout/refreshToken)
		systemRouter.InitRouteRouter(PrivateGroup, PublicGroup)    // 路由下发(/route:getConstantRoutes 公开,getUserRoutes/isRouteExist 私有)
		systemRouter.InitDictRouter(PrivateGroup)                  // 字典管理(/system/dict/type/*、/system/dict/data/*)
		systemRouter.InitPostRouter(PrivateGroup)                  // 岗位管理(/system/post/*)
		systemRouter.InitDeptRouter(PrivateGroup)                  // 部门管理(/system/dept/*)
		systemRouter.InitMenuRouter(PrivateGroup)                  // 菜单管理(/system/menu/*)
		systemRouter.InitRoleRouter(PrivateGroup)                  // 角色管理(/system/role/*)
		systemRouter.InitUserRouter(PrivateGroup)                  // 用户管理(/system/user/*)
		systemRouter.InitLoginLogRouter(PrivateGroup)              // 登录日志(/log/loginlog/*)
		systemRouter.InitOperLogRouter(PrivateGroup)               // 操作日志(/log/operlog/*)
		systemRouter.InitNoticeRouter(PrivateGroup)                // 通知公告(/system/notice/*)
		systemRouter.InitSettingRouter(PrivateGroup, PublicGroup)  // 系统设置(/system/setting GET/PUT + /system/setting/public)
		systemRouter.InitSysErrorRouter(PrivateGroup, PublicGroup) // 错误日志(/log/sysError/* + 前端上报)
		systemRouter.InitSocialRouter(PrivateGroup, PublicGroup)   // 第三方绑定/社交登录(/auth/binding|/auth/social/callback 公开;/system/social/list|/auth/unlock 私有)
		systemRouter.InitWecomRouter(PublicGroup)                  // 企业微信扫码登录(/auth/wecomLogin|/auth/qrCodeStatus|/wecomCallback|/auth/wecomWebviewLogin 全公开)
		systemRouter.InitTimedTaskRouter(PrivateGroup)             // 定时任务(/timedTask/*)
		systemRouter.InitOnlineRouter(PrivateGroup)                // 在线设备(/monitor/online,个人中心视角:仅当前用户自己)
		gatewayRouter.InitProviderRouter(PrivateGroup)            // AI 网关·供应商管理(/gateway/provider/*)
		gatewayRouter.InitCredentialRouter(PrivateGroup)          // AI 网关·凭证管理(/gateway/credential/*)
		gatewayRouter.InitModelRouter(PrivateGroup)               // AI 网关·模型管理(/gateway/model/*)
		gatewayRouter.InitDeploymentRouter(PrivateGroup)          // AI 网关·模型部署(/gateway/model/deployment/*)
		gatewayRouter.InitAiKeyRouter(PrivateGroup)               // AI 网关·AI 密钥(/gateway/ai-key/*)
		gatewayRouter.InitUsageRouter(PrivateGroup)               // AI 网关·用量统计(/gateway/usage/*)
		gatewayRouter.InitDashboardRouter(PrivateGroup)          // AI 网关·看板(/gateway/dashboard/*)
		gatewayRouter.InitCostAnalysisRouter(PrivateGroup)       // AI 网关·成本分析(/gateway/cost/*)
		gatewayRouter.InitRouterSettingsRouter(PrivateGroup)    // AI 网关·路由策略(/gateway/router/settings)
		gatewayRouter.InitResourceApplicationRouter(PrivateGroup) // AI 网关·资源申请审批(/gateway/application/*)
		gatewayRouter.InitMCPRouter(PrivateGroup)                 // AI 网关·MCP 服务器管理(/gateway/mcp/*)
		gatewayRouter.InitSkillRouter(PrivateGroup)               // AI 网关·Skill 管理(/gateway/skill/*)
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
