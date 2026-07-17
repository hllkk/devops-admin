package system

import (
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
		message  = "前往初始化数据库"
		needInit = true
	)

	if global.OPS_DB != nil {
		message = "数据库无需初始化"
		needInit = false
	}
	logger.WithCtx(c.Request.Context()).Mod("biz").Info(message)
	response.OkWithDetailed(gin.H{"needInit": needInit}, message, c)
}

// PingDB 测试数据库连接（仅在系统未初始化时可用；不建库、不落盘）
// @Tags     SysInit
// @Summary  测试数据库连接
// @Produce  application/json
// @Param    data  body      request.DBConnTest               true  "数据库连接参数"
// @Success  200   {object}  response.Response{data=string}  "连接成功"
// @Router   /init/db/ping [post]
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
