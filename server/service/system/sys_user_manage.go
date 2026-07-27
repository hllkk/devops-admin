package system

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/upload"
	"gorm.io/gorm"
)

// 用户管理 CRUD(对齐前端 /system/user/* 管理员侧资源)。
// auth 链路(Login/Register/GetUserInfo/GetUserDetail)仍在 sys_user.go,本文件只含管理方法。

// userOrder 用户列表统一排序:主键 id(雪花)升序。限定 sys_users.id 以避免 roleId join 时列名歧义。
const userOrder = "sys_users.id ASC"

// GetList 分页查用户列表(对齐前端 GET /system/user/list)。
// deptId/userName/nickName/phonenumber/status 过滤;roleId>0 时 join sys_user_role;分页走 LimitOffset。
func (s *UserService) GetList(ctx context.Context, q systemReq.UserSearch) (list []system.SysUser, total int64, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&system.SysUser{})
	if q.UserName != "" {
		db = db.Where("user_name LIKE ?", "%"+q.UserName+"%")
	}
	if q.NickName != "" {
		db = db.Where("nick_name LIKE ?", "%"+q.NickName+"%")
	}
	if q.Phonenumber != "" {
		db = db.Where("phonenumber LIKE ?", "%"+q.Phonenumber+"%")
	}
	if q.Status != "" {
		db = db.Where("status = ?", q.Status)
	}
	if q.DeptId > 0 {
		db = db.Where("dept_id = ?", q.DeptId)
	}
	if q.RoleId > 0 {
		db = db.Joins("JOIN sys_user_role ON sys_user_role.sys_user_id = sys_users.id").
			Where("sys_user_role.sys_role_id = ?", q.RoleId)
	}
	limit, offset := q.LimitOffset()
	db = db.Preload("Dept").Preload("Roles") // 主部门供回填 deptName;Roles 供回填 superAdmin(任一角色 SuperAdmin=true)
	if limit > 0 {
		err = db.Count(&total).Order(userOrder).Limit(limit).Offset(offset).Find(&list).Error
	} else {
		err = db.Count(&total).Order(userOrder).Find(&list).Error
	}
	if err == nil {
		// deptName/superAdmin 均为 gorm:"-" 内存字段,Preload 后显式回填(前端列表部门列+超管保护依赖)
		for i := range list {
			list[i].DeptName = list[i].Dept.DeptName
			list[i].SuperAdmin = list[i].GetSuperAdmin()
		}
	}
	return
}

// GetDeptUserList 部门下用户列表(对齐前端 GET /system/user/list/dept/{deptId},部门负责人选择用,不分页)。
func (s *UserService) GetDeptUserList(ctx context.Context, deptId int64) (list []system.SysUser, err error) {
	err = global.OPS_DB.WithContext(ctx).Where("dept_id = ? AND status = ?", deptId, "0").Order(userOrder).Find(&list).Error
	return
}

// Create 新增用户(事务:用户名唯一校验 → 建用户[bcrypt 密码 + UUID + PasswordUpdatedAt] → 保存 sys_user_role/sys_user_post)。
func (s *UserService) Create(ctx context.Context, req systemReq.UserOperateParams, createBy int64) error {
	if req.UserName == "" {
		return errors.New("用户名不能为空")
	}
	if req.NickName == "" {
		return errors.New("昵称不能为空")
	}
	if req.Password == "" {
		return errors.New("密码不能为空")
	}
	if err := utils.ValidatePasswordComplexity(req.Password, (&SecurityConfigService{}).Current(ctx)); err != nil {
		return err
	}
	var cnt int64
	if err := global.OPS_DB.WithContext(ctx).Model(&system.SysUser{}).Where("user_name = ?", req.UserName).Count(&cnt).Error; err != nil {
		return err
	}
	if cnt > 0 {
		return errors.New("用户名已存在")
	}
	u := system.SysUser{
		DeptId:      req.DeptId.Int64(),
		UserName:    req.UserName,
		NickName:    req.NickName,
		Email:       req.Email,
		Phonenumber: req.Phonenumber,
		Sex:         req.Sex,
		Status:      req.Status,
		Remark:      req.Remark,
		UUID:        uuid.New(),
	}
	u.Password = utils.BcryptHash(req.Password)
	now := time.Now()
	u.PasswordUpdatedAt = &now
	u.CreateBy = createBy
	u.UpdateBy = createBy
	roleIds := toInt64Slice(req.RoleIds)
	if len(roleIds) > 0 {
		u.RoleId = roleIds[0] // 主角色(登录链路 claims 用),取所选角色第一个回填,避免落 default 888
	}
	postIds := toInt64Slice(req.PostIds)
	return global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&u).Error; err != nil {
			return err
		}
		if err := saveUserRoles(tx, u.UserId, roleIds); err != nil {
			return err
		}
		return saveUserPosts(tx, u.UserId, postIds)
	})
}

// Update 修改用户(事务:更新用户[password 空=不改] + 全量替换 sys_user_role/sys_user_post)。
func (s *UserService) Update(ctx context.Context, req systemReq.UserOperateParams, updateBy int64) error {
	userId := req.UserId.Int64()
	if userId == 0 {
		return errors.New("用户ID不能为空")
	}
	if req.NickName == "" {
		return errors.New("昵称不能为空")
	}
	updates := map[string]interface{}{
		"dept_id":     req.DeptId.Int64(),
		"nick_name":   req.NickName,
		"email":       req.Email,
		"phonenumber": req.Phonenumber,
		"sex":         req.Sex,
		"status":      req.Status,
		"remark":      req.Remark,
		"update_by":   updateBy,
	}
	changePwd := req.Password != ""
	if changePwd {
		if err := utils.ValidatePasswordComplexity(req.Password, (&SecurityConfigService{}).Current(ctx)); err != nil {
			return err
		}
		updates["password"] = utils.BcryptHash(req.Password)
	}
	roleIds := toInt64Slice(req.RoleIds)
	if len(roleIds) > 0 {
		updates["role_id"] = roleIds[0] // 主角色随所选角色同步,避免落 default 888
	}
	postIds := toInt64Slice(req.PostIds)
	return global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&system.SysUser{}).Where("id = ?", userId).Updates(updates).Error; err != nil {
			return err
		}
		if changePwd {
			// PasswordUpdatedAt 为指针字段,map 不便,单独 Update
			if err := tx.Model(&system.SysUser{}).Where("id = ?", userId).Update("password_updated_at", time.Now()).Error; err != nil {
				return err
			}
		}
		if err := saveUserRoles(tx, userId, roleIds); err != nil {
			return err
		}
		return saveUserPosts(tx, userId, postIds)
	})
}

// UpdateStatus 修改用户状态(对齐前端 PUT /system/user/changeStatus)。
func (s *UserService) UpdateStatus(ctx context.Context, req systemReq.UserOperateParams, updateBy int64) error {
	userId := req.UserId.Int64()
	if userId == 0 {
		return errors.New("用户ID不能为空")
	}
	return global.OPS_DB.WithContext(ctx).Model(&system.SysUser{}).Where("id = ?", userId).
		Updates(map[string]interface{}{"status": req.Status, "update_by": updateBy}).Error
}

// Delete 批量删除用户(事务:清理 sys_user_role/sys_user_post/sys_user_departments → 删用户)。
func (s *UserService) Delete(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return errors.New("未选择删除项")
	}
	return global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("sys_user_id IN ?", ids).Delete(&system.SysUserRole{}).Error; err != nil {
			return err
		}
		if err := tx.Where("sys_user_id IN ?", ids).Delete(&system.SysUserPost{}).Error; err != nil {
			return err
		}
		if err := tx.Where("sys_user_id IN ?", ids).Delete(&system.SysUserDepartment{}).Error; err != nil {
			return err
		}
		return tx.Where("id IN ?", ids).Delete(&system.SysUser{}).Error
	})
}

// GetDetail 用户详情(对齐前端 GET /system/user/{userId}):postIds/roleIds(字符串)/roles(供 drawer 回显+下拉)。
func (s *UserService) GetDetail(ctx context.Context, userId int64) (result system.UserInfo, err error) {
	var u system.SysUser
	if err = global.OPS_DB.WithContext(ctx).Preload("Roles").Where("id = ?", userId).First(&u).Error; err != nil {
		return
	}
	result.Roles = u.Roles
	var roleIds []int64
	if err = global.OPS_DB.WithContext(ctx).Model(&system.SysUserRole{}).Where("sys_user_id = ?", userId).Pluck("sys_role_id", &roleIds).Error; err != nil {
		return
	}
	result.RoleIds = int64ToStrSlice(roleIds)
	var postIds []int64
	if err = global.OPS_DB.WithContext(ctx).Model(&system.SysUserPost{}).Where("sys_user_id = ?", userId).Pluck("sys_post_id", &postIds).Error; err != nil {
		return
	}
	result.PostIds = int64ToStrSlice(postIds)
	return
}

// ResetPwd 重置用户密码(对齐前端 PUT /system/user/resetPwd;明文 bcrypt 存储,无加密中间件)。
func (s *UserService) ResetPwd(ctx context.Context, req systemReq.ResetUserPwdParams, updateBy int64) error {
	userId := req.UserId.Int64()
	if userId == 0 {
		return errors.New("用户ID不能为空")
	}
	if req.Password == "" {
		return errors.New("密码不能为空")
	}
	if err := utils.ValidatePasswordComplexity(req.Password, (&SecurityConfigService{}).Current(ctx)); err != nil {
		return err
	}
	if err := global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&system.SysUser{}).Where("id = ?", userId).
			Updates(map[string]interface{}{"password": utils.BcryptHash(req.Password), "update_by": updateBy}).Error; err != nil {
			return err
		}
		return tx.Model(&system.SysUser{}).Where("id = ?", userId).Update("password_updated_at", time.Now()).Error
	}); err != nil {
		return err
	}
	// 密码重置成功:吊销该用户所有旧会话,旧 token 立即失效,需用新密码重新登录
	_ = (&OnlineService{}).RevokeUserSessions(ctx, userId)
	return nil
}

// ChangeMyPassword 当前用户自助改密(密码过期强制改密场景的唯一入口)。
// 流程:校验旧密码 → 新旧不同 → 复杂度校验 → bcrypt 新密码 → 刷 PasswordUpdatedAt。
// Preload Roles 以保证 GetSuperAdmin 正确(超管改密重发 token 需 SuperAdmin=true)。
// 返回带 Roles 的 user,供 handler 重发 MustChangePwd=false token。
func (s *UserService) ChangeMyPassword(ctx context.Context, userId int64, oldPwd, newPwd string) (system.SysUser, error) {
	var u system.SysUser
	if err := global.OPS_DB.WithContext(ctx).Preload("Roles").Where("id = ?", userId).First(&u).Error; err != nil {
		return u, errors.New("用户不存在")
	}
	if !utils.BcryptCheck(oldPwd, u.Password) {
		return u, errors.New("旧密码错误")
	}
	if oldPwd == newPwd {
		return u, errors.New("新密码不能与旧密码相同")
	}
	if err := utils.ValidatePasswordComplexity(newPwd, (&SecurityConfigService{}).Current(ctx)); err != nil {
		return u, err
	}
	now := time.Now()
	if err := global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&system.SysUser{}).Where("id = ?", userId).
			Update("password", utils.BcryptHash(newPwd)).Error; err != nil {
			return err
		}
		return tx.Model(&system.SysUser{}).Where("id = ?", userId).
			Update("password_updated_at", now).Error
	}); err != nil {
		return u, err
	}
	// 改密成功:吊销其他旧会话(多端旧设备立即失效);当前会话由 handler 重发新 token(jti 不同,不受影响)
	_ = (&OnlineService{}).RevokeUserSessions(ctx, userId)
	return u, nil
}

// UpdateMyProfile 当前用户自助修改基本资料(对齐前端 PUT /system/user/profile)。
// 仅写 nick_name/email/phonenumber/sex + update_by;userName/角色/部门/状态不在自助范围,走管理员侧接口。
func (s *UserService) UpdateMyProfile(ctx context.Context, userId int64, req systemReq.UpdateMyProfileParams) error {
	return global.OPS_DB.WithContext(ctx).Model(&system.SysUser{}).Where("id = ?", userId).
		Updates(map[string]interface{}{
			"nick_name":   req.NickName,
			"email":       req.Email,
			"phonenumber": req.Phonenumber,
			"sex":         req.Sex,
			"update_by":   userId,
		}).Error
}

// UpdateMyAvatar 当前用户自助上传头像(对齐前端 POST /system/user/profile/avatar,字段名 avatarfile)。
// 经统一 OSS 抽象落存储,把返回 url 写回 SysUser.Avatar;local 模式 url 形态与 media 一致(原样存,不补前缀)。
// 头像场景仅允许图片后缀,杜绝借头像接口传任意可执行/文档文件。
func (s *UserService) UpdateMyAvatar(ctx context.Context, userId int64, file *multipart.FileHeader) (string, error) {
	if file.Size > MaxAvatarBytes {
		return "", fmt.Errorf("头像文件不能超过 %dMB", MaxAvatarBytes>>20)
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
	default:
		return "", errors.New("仅支持 jpg/jpeg/png/gif/webp 格式的图片")
	}
	oss := upload.NewOss()
	url, _, err := oss.UploadFile(ctx, file)
	if err != nil {
		return "", errors.New("头像上传失败: " + err.Error())
	}
	if err = global.OPS_DB.WithContext(ctx).Model(&system.SysUser{}).Where("id = ?", userId).
		Updates(map[string]interface{}{"avatar": url, "update_by": userId}).Error; err != nil {
		return "", err
	}
	return url, nil
}

// saveUserRoles 全量替换用户角色(删后批量插 sys_user_role)。
func saveUserRoles(tx *gorm.DB, userId int64, roleIds []int64) error {
	if err := tx.Where("sys_user_id = ?", userId).Delete(&system.SysUserRole{}).Error; err != nil {
		return err
	}
	rows := make([]system.SysUserRole, 0, len(roleIds))
	for _, rid := range roleIds {
		if rid > 0 {
			rows = append(rows, system.SysUserRole{SysUserId: userId, SysRoleId: rid})
		}
	}
	if len(rows) == 0 {
		return nil
	}
	return tx.Create(&rows).Error
}

// saveUserPosts 全量替换用户岗位(删后批量插 sys_user_post)。
func saveUserPosts(tx *gorm.DB, userId int64, postIds []int64) error {
	if err := tx.Where("sys_user_id = ?", userId).Delete(&system.SysUserPost{}).Error; err != nil {
		return err
	}
	rows := make([]system.SysUserPost, 0, len(postIds))
	for _, pid := range postIds {
		if pid > 0 {
			rows = append(rows, system.SysUserPost{SysUserId: userId, SysPostId: pid})
		}
	}
	if len(rows) == 0 {
		return nil
	}
	return tx.Create(&rows).Error
}

// int64ToStrSlice 将 []int64 转为 []string(对齐前端 string[])。
func int64ToStrSlice(ids []int64) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, strconv.FormatInt(id, 10))
	}
	return out
}

// 导入导出相关常量。
const (
	// ExportMaxRows 单表导出上限,防止误导全表打爆内存(管理后台场景足够)。
	ExportMaxRows = 10000
	// DefaultImportPassword 导入新用户的初始密码(满足默认复杂度:大小写+数字+特殊+8位)。
	// PasswordUpdatedAt 留零值,登录链路 MustChangePwdGuard 会强制首登改密,故明文常量无暴露风险。
	DefaultImportPassword = "User@1234"
	// MaxAvatarBytes 头像上传大小上限(5MB),防已登录用户上传超大文件耗尽内存/OSS 配额。
	MaxAvatarBytes = 5 << 20
)

// ExportList 按列表查询条件导出用户(全量,不分页;过滤条件与 GetList 保持一致,加导出上限防过大)。
// Preload Dept/Roles 以回填 deptName/superAdmin(gorm:"-" 内存字段,导出列依赖)。
func (s *UserService) ExportList(ctx context.Context, q systemReq.UserSearch) (list []system.SysUser, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&system.SysUser{})
	if q.UserName != "" {
		db = db.Where("user_name LIKE ?", "%"+q.UserName+"%")
	}
	if q.NickName != "" {
		db = db.Where("nick_name LIKE ?", "%"+q.NickName+"%")
	}
	if q.Phonenumber != "" {
		db = db.Where("phonenumber LIKE ?", "%"+q.Phonenumber+"%")
	}
	if q.Status != "" {
		db = db.Where("status = ?", q.Status)
	}
	if q.DeptId > 0 {
		db = db.Where("dept_id = ?", q.DeptId)
	}
	if q.RoleId > 0 {
		db = db.Joins("JOIN sys_user_role ON sys_user_role.sys_user_id = sys_users.id").
			Where("sys_user_role.sys_role_id = ?", q.RoleId)
	}
	db = db.Preload("Dept").Preload("Roles")
	err = db.Order(userOrder).Limit(ExportMaxRows).Find(&list).Error
	if err == nil {
		for i := range list {
			list[i].DeptName = list[i].Dept.DeptName
			list[i].SuperAdmin = list[i].GetSuperAdmin()
		}
	}
	return
}

// ImportUsers 批量导入用户(由 api 层调用 excel.Parse 得到记录后传入)。
// 已存在判定用 Unscoped 查询(跨软删除,保证 user_name 语义唯一),分三种:
//   - 活用户:updateSupport=true 更新字段,false 跳过;
//   - 软删除用户:复活(清 deleted_at + 覆盖字段 + 重置密码强制首登改密),计入 update;
//   - 都没命中:新建。
//
// 新建/复活密码统一用 DefaultImportPassword(bcrypt),PasswordUpdatedAt 留空触发首登强制改密。
// 返回 insert/update/skip/fail 计数与失败明细(供前端逐行展示)。
func (s *UserService) ImportUsers(ctx context.Context, rows []map[string]string, updateSupport bool, createBy int64) (insert, updateCnt, skip, fail int, failMsgs []string) {
	// 默认密码取常规配置(运维可配),空则回退内置常量;不校验复杂度(首登 MustChangePwd 强制改密)
	defaultPwd := (&GeneralConfigService{}).Current(ctx).DefaultPassword
	if defaultPwd == "" {
		defaultPwd = DefaultImportPassword
	}
	pwd := utils.BcryptHash(defaultPwd)
	for i, row := range rows {
		lineNo := i + 2 // Excel 行号(第 1 行为表头,数据从第 2 行起)
		userName := strings.TrimSpace(row["UserName"])
		if userName == "" {
			fail++
			failMsgs = append(failMsgs, fmt.Sprintf("第 %d 行:用户名不能为空", lineNo))
			continue
		}
		// Unscoped 查询同时覆盖"活用户"与"软删除用户"(user_name 跨软删除语义唯一)
		var exist system.SysUser
		found := global.OPS_DB.WithContext(ctx).Unscoped().Where("user_name = ?", userName).First(&exist).Error == nil
		if found && exist.DeletedAt.Valid {
			// 软删除用户:复活——清 deleted_at、覆盖字段、重置密码;PasswordUpdatedAt 留空触发首登强制改密
			revive := map[string]interface{}{
				"deleted_at":          nil,
				"update_by":           createBy,
				"password":            pwd,
				"password_updated_at": nil,
			}
			applyImportFields(revive, row)
			if err := global.OPS_DB.WithContext(ctx).Unscoped().Model(&system.SysUser{}).Where("id = ?", exist.UserId).Updates(revive).Error; err != nil {
				fail++
				failMsgs = append(failMsgs, fmt.Sprintf("第 %d 行:复活失败 %s(%v)", lineNo, userName, err))
			} else {
				updateCnt++
			}
			continue
		}
		if found {
			// 活用户:按 updateSupport 决定更新或跳过
			if !updateSupport {
				skip++
				continue
			}
			updates := map[string]interface{}{"update_by": createBy}
			applyImportFields(updates, row)
			if err := global.OPS_DB.WithContext(ctx).Model(&system.SysUser{}).Where("id = ?", exist.UserId).Updates(updates).Error; err != nil {
				fail++
				failMsgs = append(failMsgs, fmt.Sprintf("第 %d 行:更新失败 %s(%v)", lineNo, userName, err))
			} else {
				updateCnt++
			}
			continue
		}
		// 新建:昵称默认取用户名;性别/状态默认 0;部门 ID 可选
		nickName := strings.TrimSpace(row["NickName"])
		if nickName == "" {
			nickName = userName
		}
		sex := strings.TrimSpace(row["Sex"])
		if sex == "" {
			sex = "0"
		}
		status := strings.TrimSpace(row["Status"])
		if status == "" {
			status = "0"
		}
		u := system.SysUser{
			UserName:    userName,
			NickName:    nickName,
			Email:       strings.TrimSpace(row["Email"]),
			Phonenumber: strings.TrimSpace(row["Phonenumber"]),
			Sex:         sex,
			Status:      status,
			UUID:        uuid.New(),
		}
		u.Password = pwd
		u.CreateBy = createBy
		u.UpdateBy = createBy
		if deptId, perr := strconv.ParseInt(strings.TrimSpace(row["DeptId"]), 10, 64); perr == nil && deptId > 0 {
			u.DeptId = deptId
		}
		if err := global.OPS_DB.WithContext(ctx).Create(&u).Error; err != nil {
			fail++
			failMsgs = append(failMsgs, fmt.Sprintf("第 %d 行:新增失败 %s(%v)", lineNo, userName, err))
		} else {
			insert++
		}
	}
	return
}

// applyImportFields 把导入行的可更新字段写入 map(空值跳过),供活用户更新与软删除复活复用。
func applyImportFields(m map[string]interface{}, row map[string]string) {
	if v := strings.TrimSpace(row["NickName"]); v != "" {
		m["nick_name"] = v
	}
	if v := strings.TrimSpace(row["Email"]); v != "" {
		m["email"] = v
	}
	if v := strings.TrimSpace(row["Phonenumber"]); v != "" {
		m["phonenumber"] = v
	}
	if v := strings.TrimSpace(row["Sex"]); v != "" {
		m["sex"] = v
	}
	if v := strings.TrimSpace(row["Status"]); v != "" {
		m["status"] = v
	}
	if deptId, perr := strconv.ParseInt(strings.TrimSpace(row["DeptId"]), 10, 64); perr == nil && deptId > 0 {
		m["dept_id"] = deptId
	}
}
