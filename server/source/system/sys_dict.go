package system

import (
	"context"

	sysModel "github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/service/system"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 字典无跨初始化器依赖,排到单链末尾,避免与 sys_role(casbin+1)撞号。
const initOrderDict = initOrderDepartment + 1

type initDict struct{}

// auto run
func init() {
	system.RegisterInit(initOrderDict, &initDict{})
}

func (i *initDict) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}
	return ctx, db.AutoMigrate(&sysModel.SysDictType{}, &sysModel.SysDictData{})
}

func (i *initDict) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	return db.Migrator().HasTable(&sysModel.SysDictType{}) && db.Migrator().HasTable(&sysModel.SysDictData{})
}

func (i *initDict) InitializerName() string {
	return sysModel.SysDictType{}.TableName()
}

func (i *initDict) InitializeData(ctx context.Context) (next context.Context, err error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}
	dictTypes := []sysModel.SysDictType{
		{DictName: "系统状态", DictType: "sys_common_status", Remark: "登录状态列表"},
		{DictName: "设备类型", DictType: "sys_device_type", Remark: "客户端设备类型"},
		{DictName: "授权类型", DictType: "sys_grant_type", Remark: "认证授权类型"},
		{DictName: "系统开关", DictType: "sys_normal_disable", Remark: "系统开关列表"},
		{DictName: "通知状态", DictType: "sys_notice_status", Remark: "通知状态列表"},
		{DictName: "通知类型", DictType: "sys_notice_type", Remark: "通知类型列表"},
		{DictName: "操作类型", DictType: "sys_oper_type", Remark: "操作类型列表"},
		{DictName: "菜单状态", DictType: "sys_show_hide", Remark: "菜单状态列表"},
		{DictName: "用户性别", DictType: "sys_user_sex", Remark: "用户性别列表"},
		{DictName: "系统是否", DictType: "sys_yes_no", Remark: "系统是否列表"},
		{DictName: "业务状态", DictType: "wf_business_status", Remark: "业务状态列表"},
		{DictName: "表单类型", DictType: "wf_form_type", Remark: "表单类型列表"},
		{DictName: "任务状态", DictType: "wf_task_status", Remark: "任务状态"},
	}

	// DictType 有 uniqueIndex,用 OnConflict DoNothing 避免中间态重跑时撞唯一键,使 InitDB 可自愈。
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&dictTypes).Error; err != nil {
		return ctx, errors.Wrap(err, sysModel.SysDictType{}.TableName()+"字典类型数据初始化失败!")
	}

	dictDatas := []sysModel.SysDictData{
		{DictSort: 1, DictLabel: "dict.sys_common_status.success", DictValue: "0", DictType: "sys_common_status", ListClass: "primary", Remark: "正常状态"},
		{DictSort: 2, DictLabel: "dict.sys_common_status.fail", DictValue: "1", DictType: "sys_common_status", ListClass: "error", Remark: "停用状态"},
		{DictSort: 1, DictLabel: "dict.sys_device_type.pc", DictValue: "pc", DictType: "sys_device_type", ListClass: "default", Remark: "PC"},
		{DictSort: 2, DictLabel: "dict.sys_device_type.android", DictValue: "android", DictType: "sys_device_type", ListClass: "default", Remark: "安卓"},
		{DictSort: 3, DictLabel: "dict.sys_device_type.ios", DictValue: "ios", DictType: "sys_device_type", ListClass: "default", Remark: "IOS"},
		{DictSort: 4, DictLabel: "dict.sys_device_type.miniapp", DictValue: "xcx", DictType: "sys_device_type", ListClass: "default", Remark: "小程序"},
		{DictSort: 1, DictLabel: "dict.sys_grant_type.password", DictValue: "password", DictType: "sys_grant_type", CssClass: "el-check-tag", ListClass: "default", Remark: "密码认证"},
		{DictSort: 2, DictLabel: "dict.sys_grant_type.sms", DictValue: "sms", DictType: "sys_grant_type", CssClass: "el-check-tag", ListClass: "default", Remark: "短信认证"},
		{DictSort: 3, DictLabel: "dict.sys_grant_type.email", DictValue: "email", DictType: "sys_grant_type", CssClass: "el-check-tag", ListClass: "default", Remark: "邮箱认证"},
		{DictSort: 4, DictLabel: "dict.sys_grant_type.miniapp", DictValue: "xcx", DictType: "sys_grant_type", CssClass: "el-check-tag", ListClass: "default", Remark: "小程序认证"},
		{DictSort: 5, DictLabel: "dict.sys_grant_type.social", DictValue: "social", DictType: "sys_grant_type", CssClass: "el-check-tag", ListClass: "default", Remark: "三方登录认证"},
		{DictSort: 1, DictLabel: "dict.sys_normal_disable.normal", DictValue: "0", DictType: "sys_normal_disable", ListClass: "primary", IsDefault: "Y", Remark: "正常状态"},
		{DictSort: 2, DictLabel: "dict.sys_normal_disable.disable", DictValue: "1", DictType: "sys_normal_disable", ListClass: "error", IsDefault: "N", Remark: "停用状态"},
		{DictSort: 1, DictLabel: "dict.sys_notice_status.normal", DictValue: "0", DictType: "sys_notice_status", ListClass: "primary", IsDefault: "Y", Remark: "正常状态"},
		{DictSort: 2, DictLabel: "dict.sys_notice_status.close", DictValue: "1", DictType: "sys_notice_status", ListClass: "error", IsDefault: "N", Remark: "关闭状态"},
		{DictSort: 1, DictLabel: "dict.sys_notice_type.notice", DictValue: "1", DictType: "sys_notice_type", ListClass: "warning", Remark: "通知"},
		{DictSort: 2, DictLabel: "dict.sys_notice_type.announcement", DictValue: "2", DictType: "sys_notice_type", ListClass: "success", Remark: "公告"},
		{DictSort: 1, DictLabel: "dict.sys_oper_type.insert", DictValue: "1", DictType: "sys_oper_type", ListClass: "info", Remark: "新增操作"},
		{DictSort: 2, DictLabel: "dict.sys_oper_type.update", DictValue: "2", DictType: "sys_oper_type", ListClass: "info", Remark: "修改操作"},
		{DictSort: 3, DictLabel: "dict.sys_oper_type.delete", DictValue: "3", DictType: "sys_oper_type", ListClass: "error", Remark: "删除操作"},
		{DictSort: 4, DictLabel: "dict.sys_oper_type.grant", DictValue: "4", DictType: "sys_oper_type", ListClass: "primary", Remark: "授权操作"},
		{DictSort: 5, DictLabel: "dict.sys_oper_type.export", DictValue: "5", DictType: "sys_oper_type", ListClass: "warning", Remark: "导出操作"},
		{DictSort: 6, DictLabel: "dict.sys_oper_type.import", DictValue: "6", DictType: "sys_oper_type", ListClass: "warning", Remark: "导入操作"},
		{DictSort: 7, DictLabel: "dict.sys_oper_type.force", DictValue: "7", DictType: "sys_oper_type", ListClass: "error", Remark: "强退操作"},
		{DictSort: 8, DictLabel: "dict.sys_oper_type.gencode", DictValue: "8", DictType: "sys_oper_type", ListClass: "warning", Remark: "生成操作"},
		{DictSort: 9, DictLabel: "dict.sys_oper_type.clean", DictValue: "9", DictType: "sys_oper_type", ListClass: "error", Remark: "清空操作"},
		{DictSort: 10, DictLabel: "dict.sys_oper_type.other", DictValue: "0", DictType: "sys_oper_type", ListClass: "info", Remark: "其他操作"},
		{DictSort: 1, DictLabel: "dict.sys_show_hide.show", DictValue: "0", DictType: "sys_show_hide", ListClass: "primary", IsDefault: "Y", Remark: "显示菜单"},
		{DictSort: 2, DictLabel: "dict.sys_show_hide.hide", DictValue: "1", DictType: "sys_show_hide", ListClass: "error", IsDefault: "N", Remark: "隐藏菜单"},
		{DictSort: 1, DictLabel: "dict.sys_user_sex.male", DictValue: "0", DictType: "sys_user_sex", IsDefault: "Y", Remark: "性别男"},
		{DictSort: 2, DictLabel: "dict.sys_user_sex.female", DictValue: "1", DictType: "sys_user_sex", Remark: "性别女"},
		{DictSort: 3, DictLabel: "dict.sys_user_sex.unknown", DictValue: "2", DictType: "sys_user_sex", Remark: "未知性别"},
		{DictSort: 1, DictLabel: "dict.sys_yes_no.yes", DictValue: "Y", DictType: "sys_yes_no", ListClass: "primary", IsDefault: "Y", Remark: "系统默认是"},
		{DictSort: 2, DictLabel: "dict.sys_yes_no.no", DictValue: "N", DictType: "sys_yes_no", ListClass: "error", Remark: "系统默认否"},
		{DictSort: 1, DictLabel: "dict.wf_business_status.revoked", DictValue: "cancel", DictType: "wf_business_status", ListClass: "error", Remark: "已撤销"},
		{DictSort: 2, DictLabel: "dict.wf_business_status.draft", DictValue: "draft", DictType: "wf_business_status", ListClass: "info", Remark: "草稿"},
		{DictSort: 3, DictLabel: "dict.wf_business_status.pending", DictValue: "waiting", DictType: "wf_business_status", ListClass: "primary", Remark: "待审核"},
		{DictSort: 4, DictLabel: "dict.wf_business_status.completed", DictValue: "finish", DictType: "wf_business_status", ListClass: "success", Remark: "已完成"},
		{DictSort: 5, DictLabel: "dict.wf_business_status.cancelled", DictValue: "invalid", DictType: "wf_business_status", ListClass: "error", Remark: "已作废"},
		{DictSort: 6, DictLabel: "dict.wf_business_status.returned", DictValue: "back", DictType: "wf_business_status", ListClass: "error", Remark: "已退回"},
		{DictSort: 7, DictLabel: "dict.wf_business_status.terminated", DictValue: "termination", DictType: "wf_business_status", ListClass: "error", Remark: "已终止"},
		{DictSort: 1, DictLabel: "dict.wf_form_type.custom_form", DictValue: "static", DictType: "wf_form_type", ListClass: "success", Remark: "自定义表单"},
		{DictSort: 2, DictLabel: "dict.wf_form_type.dynamic_form", DictValue: "dynamic", DictType: "wf_form_type", ListClass: "primary", Remark: "动态表单"},
		{DictSort: 1, DictLabel: "dict.wf_task_status.revoke", DictValue: "cancel", DictType: "wf_task_status", ListClass: "error", Remark: "已撤销"},
		{DictSort: 2, DictLabel: "dict.wf_task_status.pass", DictValue: "pass", DictType: "wf_task_status", ListClass: "success", Remark: "已通过"},
		{DictSort: 3, DictLabel: "dict.wf_task_status.pending_review", DictValue: "waiting", DictType: "wf_task_status", ListClass: "primary", Remark: "待审核"},
		{DictSort: 4, DictLabel: "dict.wf_task_status.cancel", DictValue: "invalid", DictType: "wf_task_status", ListClass: "error", Remark: "作废"},
		{DictSort: 5, DictLabel: "dict.wf_task_status.return", DictValue: "back", DictType: "wf_task_status", ListClass: "success", Remark: "退回"},
		{DictSort: 6, DictLabel: "dict.wf_task_status.terminate", DictValue: "termination", DictType: "wf_task_status", ListClass: "error", Remark: "终止"},
		{DictSort: 7, DictLabel: "dict.wf_task_status.transfer", DictValue: "transfer", DictType: "wf_task_status", ListClass: "primary", Remark: "转办"},
		{DictSort: 8, DictLabel: "dict.wf_task_status.delegate", DictValue: "depute", DictType: "wf_task_status", ListClass: "primary", Remark: "委托"},
		{DictSort: 9, DictLabel: "dict.wf_task_status.copy", DictValue: "copy", DictType: "wf_task_status", ListClass: "primary", Remark: "抄送"},
		{DictSort: 10, DictLabel: "dict.wf_task_status.add_sign", DictValue: "sign", DictType: "wf_task_status", ListClass: "primary", Remark: "加签"},
	}

	// 初始化字典数据。SysDictData 无业务唯一键,直接 Create 重跑会产生重复行;
	// 故先按本次涉及的 dict_type 清旧(软删除基座会软删,默认查询已过滤),再批量插入,保证幂等。
	dictTypeNames := make([]string, 0, len(dictTypes))
	for _, t := range dictTypes {
		dictTypeNames = append(dictTypeNames, t.DictType)
	}
	if err := db.Where("dict_type IN ?", dictTypeNames).Delete(&sysModel.SysDictData{}).Error; err != nil {
		return ctx, errors.Wrap(err, sysModel.SysDictData{}.TableName()+"清理旧字典数据失败!")
	}
	if err := db.Create(&dictDatas).Error; err != nil {
		return ctx, errors.Wrap(err, sysModel.SysDictData{}.TableName()+"字典数据初始化失败!")
	}

	// 字典类型回填到 ctx(键=表名),与 sys_user 等初始化器模式一致;
	// 字典数据的 DictType 为硬编码字符串,不依赖字典类型回填 ID,故不合并两类实体。
	next = context.WithValue(ctx, i.InitializerName(), dictTypes)
	return next, nil
}

func (i *initDict) DataInserted(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	if errors.Is(db.Where("dict_type = ?", "sys_common_status").
		First(&sysModel.SysDictData{}).Error, gorm.ErrRecordNotFound) {
		return false
	}
	return true
}
