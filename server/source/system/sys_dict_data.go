package system

import (
	"context"
	"errors"
	"fmt"

	"github.com/hllkk/devops-admin/server/model/system"
	sysSvc "github.com/hllkk/devops-admin/server/service/system"
	"gorm.io/gorm"
)

const initOrderDictData = sysSvc.InitOrderSystem + 9

type initDictData struct{}

func init() { sysSvc.RegisterInit(initOrderDictData, &initDictData{}) }

func (i *initDictData) MigrateTable(ctx context.Context) (context.Context, error) {
	db, err := sysSvc.DBFromCtx(ctx)
	if err != nil {
		return ctx, err
	}
	return ctx, db.AutoMigrate(&system.SysDictData{})
}

func (i *initDictData) TableCreated(ctx context.Context) bool {
	db, err := sysSvc.DBFromCtx(ctx)
	if err != nil {
		return false
	}
	return db.Migrator().HasTable(&system.SysDictData{})
}

func (i *initDictData) InitializerName() string { return system.SysDictData{}.TableName() }

func (i *initDictData) InitializeData(ctx context.Context) (context.Context, error) {
	db, err := sysSvc.DBFromCtx(ctx)
	if err != nil {
		return ctx, err
	}
	entities := []system.SysDictData{
		// dict_type=sys_common_status：正常/成功状态。DictValue 为字符串列，故键值 "0"。
		{DictCode: 1, DictSort: 1, DictLabel: "dict.sys_common_status.success", DictValue: "0", DictType: "sys_common_status", ListClass: "primary", Remark: "正常状态"},
		{DictCode: 2, DictSort: 2, DictLabel: "dict.sys_common_status.fail", DictValue: "1", DictType: "sys_common_status", ListClass: "error", Remark: "停用状态"},
		{DictCode: 3, DictSort: 0, DictLabel: "dict.sys_device_type.pc", DictValue: "pc", DictType: "sys_device_type", ListClass: "default", Remark: "PC"},
		{DictCode: 4, DictSort: 0, DictLabel: "dict.sys_device_type.android", DictValue: "android", DictType: "sys_device_type", ListClass: "default", Remark: "安卓"},
		{DictCode: 5, DictSort: 0, DictLabel: "dict.sys_device_type.ios", DictValue: "ios", DictType: "sys_device_type", ListClass: "default", Remark: "IOS"},
		{DictCode: 6, DictSort: 0, DictLabel: "dict.sys_device_type.miniapp", DictValue: "xcx", DictType: "sys_device_type", ListClass: "default", Remark: "小程序"},
		{DictCode: 7, DictSort: 0, DictLabel: "dict.sys_grant_type.password", DictValue: "password", DictType: "sys_grant_type", CssClass: "el-check-tag", ListClass: "default", Remark: "密码认证"},
		{DictCode: 8, DictSort: 0, DictLabel: "dict.sys_grant_type.sms", DictValue: "sms", DictType: "sys_grant_type", CssClass: "el-check-tag", ListClass: "default", Remark: "短信认证"},
		{DictCode: 9, DictSort: 0, DictLabel: "dict.sys_grant_type.email", DictValue: "email", DictType: "sys_grant_type", CssClass: "el-check-tag", ListClass: "default", Remark: "邮箱认证"},
		{DictCode: 10, DictSort: 0, DictLabel: "dict.sys_grant_type.miniapp", DictValue: "xcx", DictType: "sys_grant_type", CssClass: "el-check-tag", ListClass: "default", Remark: "小程序认证"},
		{DictCode: 11, DictSort: 0, DictLabel: "dict.sys_grant_type.social", DictValue: "social", DictType: "sys_grant_type", CssClass: "el-check-tag", ListClass: "default", Remark: "三方登录认证"},
		{DictCode: 12, DictSort: 1, DictLabel: "dict.sys_normal_disable.normal", DictValue: "0", DictType: "sys_normal_disable", ListClass: "primary", IsDefault: "Y", Remark: "正常状态"},
		{DictCode: 13, DictSort: 2, DictLabel: "dict.sys_normal_disable.disable", DictValue: "1", DictType: "sys_normal_disable", ListClass: "error", Remark: "停用状态"},
		{DictCode: 14, DictSort: 1, DictLabel: "dict.sys_notice_status.normal", DictValue: "0", DictType: "sys_notice_status", ListClass: "primary", Remark: "正常状态"},
		{DictCode: 15, DictSort: 2, DictLabel: "dict.sys_notice_status.close", DictValue: "1", DictType: "sys_notice_status", ListClass: "error", Remark: "关闭状态"},
		{DictCode: 16, DictSort: 1, DictLabel: "dict.sys_notice_type.notice", DictValue: "1", DictType: "sys_notice_type", ListClass: "warning", Remark: "通知"},
		{DictCode: 17, DictSort: 2, DictLabel: "dict.sys_notice_type.announcement", DictValue: "2", DictType: "sys_notice_type", ListClass: "success", Remark: "公告"},
		{DictCode: 18, DictSort: 1, DictLabel: "dict.sys_oper_type.insert", DictValue: "1", DictType: "sys_oper_type", ListClass: "info", Remark: "新增操作"},
		{DictCode: 19, DictSort: 2, DictLabel: "dict.sys_oper_type.update", DictValue: "2", DictType: "sys_oper_type", ListClass: "info", Remark: "更新操作"},
		{DictCode: 20, DictSort: 3, DictLabel: "dict.sys_oper_type.delete", DictValue: "3", DictType: "sys_oper_type", ListClass: "error", Remark: "删除操作"},
		{DictCode: 21, DictSort: 4, DictLabel: "dict.sys_oper_type.grant", DictValue: "4", DictType: "sys_oper_type", ListClass: "primary", Remark: "授权操作"},
		{DictCode: 22, DictSort: 5, DictLabel: "dict.sys_oper_type.export", DictValue: "5", DictType: "sys_oper_type", ListClass: "warning", Remark: "导出操作"},
		{DictCode: 23, DictSort: 6, DictLabel: "dict.sys_oper_type.import", DictValue: "6", DictType: "sys_oper_type", ListClass: "warning", Remark: "导入操作"},
		{DictCode: 24, DictSort: 7, DictLabel: "dict.sys_oper_type.force", DictValue: "7", DictType: "sys_oper_type", ListClass: "error", Remark: "强退操作"},
		{DictCode: 25, DictSort: 8, DictLabel: "dict.sys_oper_type.gencode", DictValue: "8", DictType: "sys_oper_type", ListClass: "warning", Remark: "生成操作"},
		{DictCode: 26, DictSort: 9, DictLabel: "dict.sys_oper_type.clean", DictValue: "9", DictType: "sys_oper_type", ListClass: "error", Remark: "清空操作"},
		{DictCode: 27, DictSort: 99, DictLabel: "dict.sys_oper_type.other", DictValue: "other", DictType: "sys_oper_type", ListClass: "info", Remark: "其他操作"},
		{DictCode: 28, DictSort: 1, DictLabel: "dict.sys_show_hide.show", DictValue: "0", DictType: "sys_show_hide", ListClass: "primary", Remark: "显示菜单"},
		{DictCode: 29, DictSort: 2, DictLabel: "dict.sys_show_hide.hide", DictValue: "1", DictType: "sys_show_hide", ListClass: "error", Remark: "隐藏菜单"},
		{DictCode: 30, DictSort: 1, DictLabel: "dict.sys_user_sex.male", DictValue: "0", DictType: "sys_user_sex", IsDefault: "Y", Remark: "性别男"},
		{DictCode: 31, DictSort: 2, DictLabel: "dict.sys_user_sex.female", DictValue: "1", DictType: "sys_user_sex", IsDefault: "N", Remark: "性别女"},
		{DictCode: 32, DictSort: 3, DictLabel: "dict.sys_user_sex.unknown", DictValue: "2", DictType: "sys_user_sex", Remark: "未知性别"},
		{DictCode: 33, DictSort: 1, DictLabel: "dict.sys_yes_no.yes", DictValue: "Y", DictType: "sys_yes_no", ListClass: "primary", IsDefault: "Y", Remark: "系统默认是"},
		{DictCode: 34, DictSort: 2, DictLabel: "dict.sys_yes_no.no", DictValue: "N", DictType: "sys_yes_no", ListClass: "error", Remark: "系统默认否"},
		{DictCode: 35, DictSort: 1, DictLabel: "dict.wf_business_status.revoked", DictValue: "cancel", DictType: "wf_business_status", ListClass: "error", Remark: "已撤销"},
		{DictCode: 36, DictSort: 2, DictLabel: "dict.wf_business_status.draft", DictValue: "draft", DictType: "wf_business_status", ListClass: "info", Remark: "草稿"},
		{DictCode: 37, DictSort: 3, DictLabel: "dict.wf_business_status.pending", DictValue: "pending", DictType: "wf_business_status", ListClass: "primary", Remark: "待审核"},
		{DictCode: 38, DictSort: 4, DictLabel: "dict.wf_business_status.completed", DictValue: "completed", DictType: "wf_business_status", ListClass: "success", Remark: "已完成"},
		{DictCode: 39, DictSort: 5, DictLabel: "dict.wf_business_status.cancelled", DictValue: "cancelled", DictType: "wf_business_status", ListClass: "error", Remark: "已作废"},
		{DictCode: 40, DictSort: 6, DictLabel: "dict.wf_business_status.returned", DictValue: "returned", DictType: "wf_business_status", ListClass: "error", Remark: "已退回"},
		{DictCode: 41, DictSort: 7, DictLabel: "dict.wf_business_status.terminated", DictValue: "terminated", DictType: "wf_business_status", ListClass: "error", Remark: "已终止"},
		{DictCode: 42, DictSort: 1, DictLabel: "dict.wf_form_type.custom_form", DictValue: "static", DictType: "wf_form_type", ListClass: "success", Remark: "自定义表单"},
		{DictCode: 43, DictSort: 2, DictLabel: "dict.wf_form_type.dynamic_form", DictValue: "dynamic", DictType: "wf_form_type", ListClass: "primary", Remark: "动态表单"},
		{DictCode: 44, DictSort: 1, DictLabel: "dict.wf_task_status.revoke", DictValue: "cancel", DictType: "wf_task_status", ListClass: "error", Remark: "撤销"},
		{DictCode: 45, DictSort: 2, DictLabel: "dict.wf_task_status.pass", DictValue: "pass", DictType: "wf_task_status", ListClass: "success", Remark: "通过"},
		{DictCode: 46, DictSort: 3, DictLabel: "dict.wf_task_status.pending_review", DictValue: "waiting", DictType: "wf_task_status", ListClass: "primary", Remark: "待审核"},
		{DictCode: 47, DictSort: 4, DictLabel: "dict.wf_task_status.cancel", DictValue: "invalid", DictType: "wf_task_status", ListClass: "error", Remark: "作废"},
		{DictCode: 48, DictSort: 5, DictLabel: "dict.wf_task_status.return", DictValue: "back", DictType: "wf_task_status", ListClass: "error", Remark: "退回"},
		{DictCode: 49, DictSort: 6, DictLabel: "dict.wf_task_status.terminate", DictValue: "termination", DictType: "wf_task_status", ListClass: "error", Remark: "终止"},
		{DictCode: 50, DictSort: 7, DictLabel: "dict.wf_task_status.transfer", DictValue: "transfer", DictType: "wf_task_status", ListClass: "primary", Remark: "转办"},
		{DictCode: 51, DictSort: 8, DictLabel: "dict.wf_task_status.delegate", DictValue: "depute", DictType: "wf_task_status", ListClass: "primary", Remark: "委托"},
		{DictCode: 52, DictSort: 9, DictLabel: "dict.wf_task_status.copy", DictValue: "copy", DictType: "wf_task_status", ListClass: "primary", Remark: "抄送"},
		{DictCode: 53, DictSort: 10, DictLabel: "dict.wf_task_status.add_sign", DictValue: "sign", DictType: "wf_task_status", ListClass: "primary", Remark: "加签"},
	}
	if err := db.Create(&entities).Error; err != nil {
		return ctx, fmt.Errorf("%s 表数据初始化失败: %w", i.InitializerName(), err)
	}
	return context.WithValue(ctx, i.InitializerName(), entities), nil
}

func (i *initDictData) DataInserted(ctx context.Context) bool {
	db, err := sysSvc.DBFromCtx(ctx)
	if err != nil {
		return false
	}
	return !errors.Is(db.Where("dict_code = ?", 1).First(&system.SysDictData{}).Error, gorm.ErrRecordNotFound)
}
