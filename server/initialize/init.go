package initialize

import (
	"github.com/hllkk/devops-admin/server/utils"
)

// 初始化全局函数
func SetupHandlers() {
	// 注册系统重载处理函数
	utils.GlobalSystemEvents.RegisterReloadHandler(func() error {
		return Reload()
	})
}
