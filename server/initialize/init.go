package initialize

import (
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/service/system"
	"github.com/hllkk/devops-admin/server/utils"
)

// 初始化全局函数
func SetupHandlers() {
	// 注册系统重载处理函数
	utils.GlobalSystemEvents.RegisterReloadHandler(func() error {
		return Reload()
	})
	// 注入首次初始化数据库路径（/init/initdb）的雪花回调注册：
	// sys_init.go 在 OPS_DB 就绪后调用该钩子，确保首初始化创建的表也能生成雪花主键。
	system.SetDBReadyCallback(func() {
		RegisterCallbacks(global.OPS_DB)
	})
}
