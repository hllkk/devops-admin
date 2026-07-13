// Package system 是 seed initializer 的聚合入口。各 initializer（sys_dept/sys_role/...）
// 在 init() 中自注册到 service/system.RegisterInit。本包必须被 main 导入链引用才会触发
// init() —— 由 initialize/gorm_biz.go 的 blank import `_ ".../source/system"` 保证。
package system
