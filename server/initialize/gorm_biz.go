package initialize

import (
	"github.com/hllkk/devops-admin/server/global"
)

func bizModel() error {
	db := global.OPS_DB
	err := db.AutoMigrate()
	if err != nil {
		return err
	}
	return nil
}
