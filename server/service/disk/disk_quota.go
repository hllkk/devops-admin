package disk

import (
	"context"
	"errors"
	"strings"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/service/system"
)

// 网盘配额对账(H3):上传 Merge 原子预占 take_up_space,删除 MoveToTrash 释放。
// 配额上限取自 sys_disk_config.StorageQuota(默认 1GB);<=0 表示不限。
// 超级管理员(绑定任意 SysRole.SuperAdmin=true 的角色)不受配额限制。
// 字段 take_up_space 加在 sys_users(AutoMigrate 自动加列),与 remote 蓝本同款语义。

// isSuperAdmin 判定用户是否绑定任意 SuperAdmin 角色(配额/限流豁免用)。
// 查连接表 sys_user_role + sys_roles(super_admin),不依赖 gorm:"-" 的内存字段。
func isSuperAdmin(ctx context.Context, userId int64) bool {
	var cnt int64
	global.OPS_DB.WithContext(ctx).Table("sys_user_role").
		Joins("JOIN sys_roles ON sys_roles.role_id = sys_user_role.sys_role_id").
		Where("sys_user_role.sys_user_id = ? AND sys_roles.super_admin = ?", userId, true).
		Count(&cnt)
	return cnt > 0
}

// diskQuotaLimitBytes 从 sys_disk_config 读用户存储配额上限(字节),0=不限。
func diskQuotaLimitBytes(ctx context.Context) int64 {
	cfg := (&system.DiskConfigService{}).Current(ctx)
	if cfg.StorageQuota <= 0 {
		return 0
	}
	var unit float64
	switch strings.ToUpper(cfg.StorageQuotaUnit) {
	case "MB":
		unit = 1 << 20
	case "TB":
		unit = 1 << 40
	default: // GB 或未知单位兜底
		unit = 1 << 30
	}
	return int64(cfg.StorageQuota * unit)
}

// reserveUserSpace 原子预占配额:take_up_space + size <= quota 才扣减,防 TOCTOU。
// size<=0 / 超管 / 配额不限 → 放行。返回 nil 表示成功(或豁免)。
func reserveUserSpace(ctx context.Context, userId int64, size int64) error {
	if size <= 0 {
		return nil
	}
	if isSuperAdmin(ctx, userId) {
		return nil
	}
	quota := diskQuotaLimitBytes(ctx)
	if quota <= 0 {
		return nil
	}
	res := global.OPS_DB.WithContext(ctx).Exec(
		"UPDATE sys_users SET take_up_space = take_up_space + ? WHERE id = ? AND (take_up_space + ?) <= ?",
		size, userId, size, quota,
	)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("存储空间不足")
	}
	return nil
}

// releaseUserSpace 释放配额(删除/移入回收站用):take_up_space - size,GREATEST 下限兜底防负。
// best effort:失败仅记录,不阻断删除流程(配额不准优于删不掉)。
func releaseUserSpace(ctx context.Context, userId int64, size int64) {
	if size <= 0 {
		return
	}
	global.OPS_DB.WithContext(ctx).Exec(
		"UPDATE sys_users SET take_up_space = GREATEST(take_up_space - ?, 0) WHERE id = ?",
		size, userId,
	)
}
