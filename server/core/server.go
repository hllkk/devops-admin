package core

import (
	"context"
	"fmt"
	"time"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/initialize"

	mcpTool "github.com/hllkk/devops-admin/server/mcp"
	"github.com/hllkk/devops-admin/server/service/system"
	"go.uber.org/zap"
)

func RunServer() {
	if global.OPS_CONFIG.System.UseRedis {
		initialize.Redis()
		if global.OPS_CONFIG.System.UseMultipoint {
			initialize.RedisList()
		}
	}

	// 初始化通用缓存（必须在 Redis 之后：有 Redis 用 Redis，否则用内存）
	initialize.InitOpsCache()

	if global.OPS_CONFIG.System.UseMongo {
		if err := initialize.Mongo.Initialization(); err != nil {
			zap.L().Error(fmt.Sprintf("%+v", err))
		}
	}

	if global.OPS_DB != nil {
		system.LoadAll(context.Background())
		(&system.SecurityConfigService{}).LoadAll(context.Background())
		(&system.GeneralConfigService{}).LoadAll(context.Background())
		(&system.LdapConfigService{}).LoadAll(context.Background())
		(&system.NotifyConfigService{}).LoadAll(context.Background())
	}

	Router := initialize.Routers()
	address := fmt.Sprintf(":%d", global.OPS_CONFIG.System.Addr)
	mcpBaseURL := mcpTool.ResolveMCPServiceURL()

	fmt.Printf(`
	欢迎使用 devops-admin-server
	当前版本:%s
	项目地址:https://github.com/hllkk/devops-admin
	插件市场:https://plugin.devops-admin.com
	默认自动化文档地址:http://127.0.0.1%s/swagger/index.html
	MCP 独立服务请手动启动: go run ./cmd/mcp -config ./cmd/mcp/config.yaml
	默认MCP StreamHTTP地址:%s
	默认前端文件运行地址:http://127.0.0.1:8080
`, global.Version, address, mcpBaseURL)

	initServer(address, Router, 10*time.Minute, 10*time.Minute)
}
