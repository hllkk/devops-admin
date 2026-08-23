package main

import (
	"go.uber.org/zap"

	"github.com/hllkk/devops-admin/server/core"
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/initialize"
)

//go:generate go env -w GO111MODULE=on
//go:generate go env -w GOPROXY=https://goproxy.cn,direct
//go:generate go mod tidy
//go:generate go mod download

// 这部分 @Tag 设置用于排序, 需要排序的接口请按照下面的格式添加
// swag init 对 @Tag 只会从入口文件解析, 默认 main.go
// 也可通过 --generalInfo flag 指定其他文件
// @Tag.Name        Base
// @Tag.Description 基础接口

// @Tag.Name        SysInit
// @Tag.Description 初始化数据库

// @Tag.Name        SysUser
// @Tag.Description 用户

// @title                       Devops-Admin Swagger API接口文档
// @version                     v0.2.0
// @description                 使用devops-admin进行开发基础平台
// @securityDefinitions.apikey  ApiKeyAuth
// @in                          header
// @name                        x-token
// @BasePath                    /

func main() {
	// 初始化系统
	initializeSystem()
	// 运行服务器
	core.RunServer()
}

// initializeSystem 初始化系统所有组件
// 提取为单独函数以便于系统重载时调用
func initializeSystem() {
	global.OPS_VP = core.Viper() // 初始化Viper
	initialize.OtherInit()
	global.OPS_LOG = core.Zap() // 初始化zap日志库
	zap.ReplaceGlobals(global.OPS_LOG)
	global.OPS_DB = initialize.Gorm() // gorm连接数据库
	global.OPS_SPEND_DB = initialize.GormSpend() // litellm spend logs 只读连接(用量回流)
	initialize.Timer()
	initialize.DBList()
	initialize.SetupHandlers() // 注册全局函数

	if global.OPS_DB != nil {
		initialize.InitSnowflake()              // 初始化雪花算法(yitter 全局单例)
		initialize.RegisterDataScopeCallbacks() // 注册数据权限 GORM 回调
		initialize.RegisterSnowflakeCallbacks() // 注册雪花 ID GORM 回调(须在建表前)
		initialize.RegisterTables()             // 初始化表
		initialize.LoadTimedTasks()             // 从 DB 恢复定时任务调度(必须在建表后)
	}
}
