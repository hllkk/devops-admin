package initialize

import (
	"github.com/hllkk/devops-admin/server/global"
)

// bizModel 业务表建表(纯业务表、无种子数据、不走 /initdb 的表在此注册)。
// 与 initialize/gorm.go 的 RegisterTables 内部散表落点分离,避免建表清单漂移。
// 详见 aiDoc/modules/backend-layer-rules.md「表注册与新增 model 的建表维护点」。
func bizModel() error {
	db := global.OPS_DB
	err := db.AutoMigrate()
	if err != nil {
		return err
	}
	return nil
}
