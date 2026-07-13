package system

import (
	"testing"

	"gorm.io/gorm"
)

// ExportedInitializers 暴露 service/system 包内已注册的 initializers（由 source/system 的
// init() 注册），供外部测试包（system_test）驱动 e2e 验证。
var ExportedInitializers = &initializers

// SetupTestDBExternal 暴露 setupTestDB 给外部测试包。
var SetupTestDBExternal = func(t *testing.T) *gorm.DB { return setupTestDB(t) }
