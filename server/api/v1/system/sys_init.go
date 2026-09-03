package system

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/common/response"
	"github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/model/system/request"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

type DBApi struct{}

// InitDB
// @Tags     SysInit
// @Summary  初始化用户数据库
// @Produce  application/json
// @Param    data  body      request.InitDB                  true  "初始化数据库参数"
// @Success  200   {object}  response.Response{data=string}  "初始化用户数据库"
// @Router   /init/initdb [post]
func (i *DBApi) InitDB(c *gin.Context) {
	if global.OPS_DB != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Error("已存在数据库配置!")
		response.FailWithMessage("已存在数据库配置", c)
		return
	}
	var dbInfo request.InitDB
	if err := c.ShouldBindJSON(&dbInfo); err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("参数校验不通过!")
		response.FailWithMessage("参数校验不通过", c)
		return
	}
	if err := initDBService.InitDB(dbInfo); err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("自动创建数据库失败!")
		response.FailWithMessage("自动创建数据库失败，请查看后台日志，检查后在进行初始化", c)
		return
	}
	response.OkWithMessage("自动创建数据库成功", c)
}

// CheckDB
// @Tags     SysInit
// @Summary  检查数据库是否需要初始化
// @Produce  application/json
// @Success  200  {object}  response.Response{data=map[string]interface{},msg=string}  "检查数据库是否需要初始化结果"
// @Router   /init/checkdb [post]
func (i *DBApi) CheckDB(c *gin.Context) {
	var (
		message     = "前往初始化数据库"
		needInit    = true
		autoInit    = false
		configReady = false
	)

	isDockerEnv := os.Getenv("DOCKER_ENV") == "true"

	// 检查是否已初始化：以 SysUser 表是否有数据为准（AutoMigrate 会建空表，库连上不等于已初始化）
	if global.OPS_DB != nil {
		var count int64
		global.OPS_DB.Model(&system.SysUser{}).Count(&count)
		if count > 0 {
			message = "数据库无需初始化"
			needInit = false
		}
	}

	// Docker 环境：检查配置是否完整，支持自动初始化
	if isDockerEnv && needInit {
		configReady = initDBService.CheckDockerConfigReady()
		if configReady {
			autoInit = true
			message = "Docker 环境配置完整，支持自动初始化"
		} else {
			message = "Docker 环境配置不完整，需要手动初始化"
		}
	}

	logger.WithCtx(c.Request.Context()).Mod("biz").Info(message)
	response.OkWithDetailed(gin.H{"needInit": needInit, "autoInit": autoInit, "configReady": configReady}, message, c)
}

// PingDB 测试数据库连接（仅在系统未初始化时可用；不建库、不落盘）
// @Tags     SysInit
// @Summary  测试数据库连接
// @Produce  application/json
// @Param    data  body      request.DBConnTest               true  "数据库连接参数"
// @Success  200   {object}  response.Response{data=string}  "连接成功"
// @Router   /init/conn-test [post]
func (i *DBApi) PingDB(c *gin.Context) {
	if global.OPS_DB != nil {
		var count int64
		global.OPS_DB.Model(&system.SysUser{}).Count(&count)
		if count > 0 {
			response.FailWithMessage("系统已初始化，无需测试连接", c)
			return
		}
	}
	var conf request.DBConnTest
	if err := c.ShouldBindJSON(&conf); err != nil {
		response.FailWithMessage("参数校验不通过", c)
		return
	}
	if err := initDBService.PingDB(conf); err != nil {
		logger.WithCtx(c.Request.Context()).Mod("init").Err(err).Error("数据库连接测试失败!")
		response.FailWithMessage("数据库连接失败："+err.Error(), c)
		return
	}
	response.OkWithMessage("数据库连接成功", c)
}

// PingRedis 测试 Redis 连接（仅在系统未初始化时可用；不落盘）
// @Tags     SysInit
// @Summary  测试 Redis 连接
// @Produce  application/json
// @Param    data  body      request.PingRedis                true  "Redis 连接参数"
// @Success  200   {object}  response.Response{data=string}  "连接成功"
// @Router   /init/redis/ping [post]
func (i *DBApi) PingRedis(c *gin.Context) {
	if global.OPS_DB != nil {
		var count int64
		global.OPS_DB.Model(&system.SysUser{}).Count(&count)
		if count > 0 {
			response.FailWithMessage("系统已初始化，无需测试连接", c)
			return
		}
	}
	var conf request.PingRedis
	if err := c.ShouldBindJSON(&conf); err != nil {
		response.FailWithMessage("参数校验不通过", c)
		return
	}
	if err := initDBService.PingRedis(conf); err != nil {
		logger.WithCtx(c.Request.Context()).Mod("init").Err(err).Error("Redis 连接测试失败!")
		response.FailWithMessage("Redis 连接失败："+err.Error(), c)
		return
	}
	response.OkWithMessage("Redis 连接成功", c)
}

// AutoInitDB Docker 环境自动初始化数据库
// 用挂载 config.yaml 的 DB 配置（敏感项经 env 覆盖）+ 环境变量 INIT_ADMIN_PASSWORD，
// 自动建库（幂等）/建表/建管理员，无需向导手填。
// 判断与执行逻辑在 service 层（AutoInitDBFromDockerConfig），与启动期自动初始化共用。
// @Tags     SysInit
// @Summary  Docker 环境自动初始化
// @Produce  application/json
// @Success  200  {object}  response.Response{data=string}  "自动初始化结果"
// @Router   /init/autoInitDB [post]
func (i *DBApi) AutoInitDB(c *gin.Context) {
	skipReason, err := initDBService.AutoInitDBFromDockerConfig()
	switch {
	case err != nil:
		logger.WithCtx(c.Request.Context()).Mod("init").Err(err).Error("自动初始化数据库失败")
		response.FailWithMessage("自动初始化失败："+err.Error(), c)
	case skipReason != "":
		response.FailWithMessage(skipReason, c)
	default:
		response.OkWithMessage("自动初始化成功", c)
	}
}
