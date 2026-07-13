package initialize

import (
	"github.com/hllkk/devops-admin/server/global"

	_ "github.com/hllkk/devops-admin/server/source/system" // 触发 seed initializer 的 init() 自注册
)

func bizModel() error {
	db := global.OPS_DB
	err := db.AutoMigrate()
	if err != nil {
		return err
	}
	return nil
}
