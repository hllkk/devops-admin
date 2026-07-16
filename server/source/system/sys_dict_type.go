package system

import (
	"context"
	"errors"
	"fmt"

	"github.com/hllkk/devops-admin/server/model/system"
	sysSvc "github.com/hllkk/devops-admin/server/service/system"
	"gorm.io/gorm"
)

const initOrderDictType = sysSvc.InitOrderSystem + 8

type initDictType struct{}

func init() { sysSvc.RegisterInit(initOrderDictType, &initDictType{}) }

func (i *initDictType) MigrateTable(ctx context.Context) (context.Context, error) {
	db, err := sysSvc.DBFromCtx(ctx)
	if err != nil {
		return ctx, err
	}
	return ctx, db.AutoMigrate(&system.SysDictType{})
}

func (i *initDictType) TableCreated(ctx context.Context) bool {
	db, err := sysSvc.DBFromCtx(ctx)
	if err != nil {
		return false
	}
	return db.Migrator().HasTable(&system.SysDictType{})
}

func (i *initDictType) InitializerName() string { return system.SysDictType{}.TableName() }

func (i *initDictType) InitializeData(ctx context.Context) (context.Context, error) {
	db, err := sysSvc.DBFromCtx(ctx)
	if err != nil {
		return ctx, err
	}
	entities := []system.SysDictType{
		{DictId: 1, DictName: "系统状态", DictType: "sys_common_status", Remark: "登录状态列表"},
		{DictId: 2, DictName: "设备类型", DictType: "sys_device_type", Remark: "客户端设备类型"},
		{DictId: 3, DictName: "授权类型", DictType: "sys_grant_type", Remark: "认证授权类型"},
		{DictId: 4, DictName: "系统开关", DictType: "sys_normal_disable", Remark: "系统开关列表"},
		{DictId: 5, DictName: "通知状态", DictType: "sys_notice_status", Remark: "通知状态列表"},
		{DictId: 6, DictName: "通知类型", DictType: "sys_notice_type", Remark: "通知类型列表"},
		{DictId: 7, DictName: "操作类型", DictType: "sys_oper_type", Remark: "操作类型列表"},
		{DictId: 8, DictName: "菜单状态", DictType: "sys_show_hide", Remark: "菜单状态列表"},
		{DictId: 9, DictName: "用户性别", DictType: "sys_user_sex", Remark: "用户性别列表"},
		{DictId: 10, DictName: "系统是否", DictType: "sys_yes_no", Remark: "系统是否列表"},
		{DictId: 11, DictName: "业务状态", DictType: "wf_business_status", Remark: "业务状态列表"},
		{DictId: 12, DictName: "表单类型", DictType: "wf_form_type", Remark: "表单类型列表"},
		{DictId: 13, DictName: "任务状态", DictType: "wf_task_status", Remark: "任务状态列表"},
	}
	if err := db.Create(&entities).Error; err != nil {
		return ctx, fmt.Errorf("%s 表数据初始化失败: %w", i.InitializerName(), err)
	}
	return context.WithValue(ctx, i.InitializerName(), entities), nil
}

func (i *initDictType) DataInserted(ctx context.Context) bool {
	db, err := sysSvc.DBFromCtx(ctx)
	if err != nil {
		return false
	}
	return !errors.Is(db.Where("dict_type = ?", "sys_common_status").First(&system.SysDictType{}).Error, gorm.ErrRecordNotFound)
}
