package gateway

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"gorm.io/gorm"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/gateway"
	gatewayReq "github.com/hllkk/devops-admin/server/model/gateway/request"
	gatewayResp "github.com/hllkk/devops-admin/server/model/gateway/response"
	"github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// SkillService Skill 管理(P2·AI 市场收尾)：注册/发布/授权/zip 分发/使用日志。
// 与模型/MCP 同套发布三档可见性+需审批申请流(公共底座 resource_type=skill)。
// 差异：Skill 是平台自有资源，不经 LiteLLM——授权锚点=AiKey.skills JSONB
// (skill ID 字符串数组)，改授权只动本地 JSONB 不推远端；zip 包存 uploads/skills/
// (独立于静态公开的 uploads/file，防匿名直连绕过审批)，下载经本服务端点鉴权分发。
type SkillService struct{}

// SkillSvc 包内共享实例(无状态服务，供 AiKeyService/ResourceApplicationService 复用)。
var SkillSvc = SkillService{}

// skillStoreDir zip 存储目录(相对运行目录)。独立于 StaticFS 匿名公开的
// uploads/file——该目录只经 DownloadSkill 端点(登录态+授权校验)分发。
const skillStoreDir = "uploads/skills"

// MaxSkillZipBytes 单个技能包大小上限(100MB)。
const MaxSkillZipBytes = 100 << 20

// ----------------------------------------------------------------------------
// 用户侧可见口径(主 Key 自愈差集源/广场/审批校验共用)
// ----------------------------------------------------------------------------

// visibleSkillScope 按发布可见性给 Skill 查询加过滤条件(与 visibleMcpScope 同构)：
// all 档直通/selected 档命中部门投影(主部门∪多部门)/user 档命中用户投影。
func visibleSkillScope(db *gorm.DB, userId, deptId int64) *gorm.DB {
	return db.Where(
		`visibility_type = ?
		OR EXISTS(SELECT 1 FROM gateway_skill_visibility v
			WHERE v.skill_id = gateway_skill.skill_id AND v.deleted_at IS NULL
			AND (v.department_id = ?
				OR v.department_id IN (SELECT ud.sys_department_id FROM sys_user_departments ud WHERE ud.sys_user_id = ?)))
		OR EXISTS(SELECT 1 FROM gateway_skill_visibility_user u
			WHERE u.skill_id = gateway_skill.skill_id AND u.user_id = ? AND u.deleted_at IS NULL)`,
		gateway.VisibilityTypeAll, deptId, userId, userId,
	)
}

// visibleSkillKeys 对指定主 Key owner 可见的免审批 Skill 锚点(ID 字符串)列表
// (自愈差集/建主 Key 默认授权数据源)：个人 owner 传 (userId,主部门) 三档全生效；
// 部门 owner 传 (0,deptId)。锚点=skill ID 字符串，Pluck 后统一转换。
func visibleSkillKeys(db *gorm.DB, userId, deptId int64) []string {
	var ids []int64
	visibleSkillScope(
		db.Model(&gateway.Skill{}).
			Where("is_active = ? AND is_published = ? AND requires_approval = ?", true, true, false),
		userId, deptId,
	).Pluck("skill_id", &ids)
	return int64SliceToStrings(ids)
}

// approvedApplicationSkillKeys 用户已批准申请的 Skill 锚点列表(审批授权兜底)：
// Skill 须仍启用+已发布(下架/删除的授权由发布对齐回收，重新发布后自愈补回)。
func approvedApplicationSkillKeys(db *gorm.DB, userId int64) []string {
	var ids []int64
	db.Table("gateway_resource_application AS a").
		Joins("JOIN gateway_skill s ON s.skill_id = a.resource_id AND s.deleted_at IS NULL AND s.is_active = ? AND s.is_published = ?", true, true).
		Where("a.deleted_at IS NULL AND a.user_id = ? AND a.resource_type = ? AND a.status = ?",
			userId, gateway.ApplicationResourceSkill, gateway.ApplicationStatusApproved).
		Pluck("s.skill_id", &ids)
	return int64SliceToStrings(ids)
}

// int64SliceToStrings int64 切片转字符串切片(AiKey.skills 锚点统一字符串口径)。
func int64SliceToStrings(ids []int64) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, fmt.Sprintf("%d", id))
	}
	return out
}

// ----------------------------------------------------------------------------
// 主 Key 授权对齐(与 syncModelToMainKeys/revokeModelFromMainKeys 同构；
// 差异：skills 无 LiteLLM 投影，只更新本地 JSONB，不调 syncKeyToLitellm)
// ----------------------------------------------------------------------------

// syncSkillToMainKeys 发布免审批 Skill 时向目标活跃主 Key 集合追加锚点(事务内，
// 单个失败 warning 继续)。目标集合由 mainKeyScopeOf 按可见档构造。
func syncSkillToMainKeys(ctx context.Context, tx *gorm.DB, skillKey string, scope func(*gorm.DB) *gorm.DB) []string {
	var keys []gateway.AiKey
	if err := scope(tx).Where("is_active = ?", true).Find(&keys).Error; err != nil {
		return []string{err.Error()}
	}
	var warnings []string
	for i := range keys {
		current := jsonToSlice(keys[i].Skills)
		if sliceContains(current, skillKey) {
			continue // 已授权
		}
		current = append(current, skillKey)
		keys[i].Skills = marshalJSONStringSlice(current)
		if err := tx.Model(&gateway.AiKey{}).Where("ai_key_id = ?", keys[i].AiKeyId).
			Update("skills", keys[i].Skills).Error; err != nil {
			warnings = append(warnings, fmt.Sprintf("主Key %d: %v", keys[i].AiKeyId, err))
		}
	}
	for _, w := range warnings {
		logger.WithCtx(ctx).Mod("gateway").Warn(w)
	}
	return warnings
}

// revokeSkillFromMainKeys Skill 授权对齐的减法半边：从不应再持有锚点的主 Key 回收
// (发布对齐/停用/删除调用)。keepScope 命中的主 Key 保留(不限启停)，nil=全部回收；
// 扫描全部主 Key 含停用。场景 Key 手工授权不在此域；与 loadMainKey 自愈差集源
// (visibleSkillKeys)同口径，回收后自愈不回加。
func revokeSkillFromMainKeys(ctx context.Context, tx *gorm.DB, skillKey string, keepScope func(*gorm.DB) *gorm.DB) []string {
	keep := map[int64]bool{}
	if keepScope != nil {
		var keepRows []gateway.AiKey
		if err := keepScope(tx).Find(&keepRows).Error; err != nil {
			return []string{err.Error()}
		}
		for i := range keepRows {
			keep[keepRows[i].AiKeyId] = true
		}
	}
	var keys []gateway.AiKey
	if err := tx.Where("key_type IN ?", []string{gateway.KeyTypePersonalMain, gateway.KeyTypeDeptMain}).
		Find(&keys).Error; err != nil {
		return []string{err.Error()}
	}
	var warnings []string
	for i := range keys {
		if keep[keys[i].AiKeyId] {
			continue
		}
		skills, changed := removeModelKey(jsonToSlice(keys[i].Skills), skillKey)
		if !changed {
			continue
		}
		keys[i].Skills = marshalJSONStringSlice(skills)
		if err := tx.Model(&gateway.AiKey{}).Where("ai_key_id = ?", keys[i].AiKeyId).
			Update("skills", keys[i].Skills).Error; err != nil {
			warnings = append(warnings, fmt.Sprintf("主Key %d: %v", keys[i].AiKeyId, err))
		}
	}
	for _, w := range warnings {
		logger.WithCtx(ctx).Mod("gateway").Warn(fmt.Sprintf("主Key 回收Skill %q 授权: %s", skillKey, w))
	}
	return warnings
}

// alignSkillAuthorization 主 Key 授权对齐(发布/启停/更新共用收尾)：
// 发布+免审批+启用 → 按可见档 sync+revoke；否则全量 revoke(未发布/需审批/停用不自动授权)。
func alignSkillAuthorization(ctx context.Context, tx *gorm.DB, s *gateway.Skill, deptIds, userIds []int64) []string {
	if s.IsPublished && !s.RequiresApproval && s.IsActive {
		// mixed 档两张投影表都读(visibilityUsesDept/User 含 mixed)；其余档行为不变
		if deptIds == nil && visibilityUsesDept(s.VisibilityType) {
			deptIds = skillVisibleDeptIds(tx, s.SkillId)
		}
		if userIds == nil && visibilityUsesUser(s.VisibilityType) {
			userIds = skillVisibleUserIds(tx, s.SkillId)
		}
		scope := mainKeyScopeOf(s.VisibilityType, deptIds, userIds)
		skillKey := SkillIdentityOf(*s)
		warnings := syncSkillToMainKeys(ctx, tx, skillKey, scope)
		return append(warnings, revokeSkillFromMainKeys(ctx, tx, skillKey, scope)...)
	}
	return revokeSkillFromMainKeys(ctx, tx, SkillIdentityOf(*s), nil)
}

// skillVisibleDeptIds / skillVisibleUserIds 读投影表现值(更新/停用路径无前端提交列表)。
func skillVisibleDeptIds(db *gorm.DB, skillId int64) []int64 {
	var ids []int64
	db.Model(&gateway.SkillVisibility{}).Where("skill_id = ?", skillId).Pluck("department_id", &ids)
	return ids
}

func skillVisibleUserIds(db *gorm.DB, skillId int64) []int64 {
	var ids []int64
	db.Model(&gateway.SkillVisibilityUser{}).Where("skill_id = ?", skillId).Pluck("user_id", &ids)
	return ids
}

// ----------------------------------------------------------------------------
// 管理员 CRUD
// ----------------------------------------------------------------------------

// GetSkillList 分页查 Skill 列表(含 zip 包状态字段)。
func (s *SkillService) GetSkillList(ctx context.Context, q gatewayReq.SkillSearch) (list []gatewayResp.SkillView, total int64, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&gateway.Skill{})
	if q.Name != "" {
		like := "%" + q.Name + "%"
		db = db.Where("name ILIKE ? OR author ILIKE ?", like, like)
	}
	if q.Category != "" {
		db = db.Where("category = ?", q.Category)
	}
	if q.IsActive != nil {
		db = db.Where("is_active = ?", *q.IsActive)
	}
	if q.IsPublished != nil {
		db = db.Where("is_published = ?", *q.IsPublished)
	}
	var rows []gateway.Skill
	limit, offset := q.LimitOffset()
	if limit > 0 {
		err = db.Count(&total).Order("skill_id DESC").Limit(limit).Offset(offset).Find(&rows).Error
	} else {
		err = db.Count(&total).Order("skill_id DESC").Find(&rows).Error
	}
	if err != nil {
		return nil, 0, err
	}
	list = make([]gatewayResp.SkillView, 0, len(rows))
	for i := range rows {
		list = append(list, toSkillView(rows[i]))
	}
	return list, total, nil
}

// GetSkill 详情。
func (s *SkillService) GetSkill(ctx context.Context, id int64) (gatewayResp.SkillView, error) {
	var row gateway.Skill
	if err := global.OPS_DB.WithContext(ctx).Where("skill_id = ?", id).First(&row).Error; err != nil {
		return gatewayResp.SkillView{}, errors.New("Skill 不存在")
	}
	return toSkillView(row), nil
}

// CreateSkill 注册 Skill(仅元数据，zip 包另走 UploadSkillPackage)：名称必填+唯一 →
// 落库。创建不发布(发布走 PublishSkill)。
func (s *SkillService) CreateSkill(ctx context.Context, req gatewayReq.SkillOperateParams, createBy int64) (gatewayResp.SkillView, error) {
	if req.Name == "" {
		return gatewayResp.SkillView{}, errors.New("技能名称不能为空")
	}
	var cnt int64
	if err := global.OPS_DB.WithContext(ctx).Model(&gateway.Skill{}).
		Where("name = ?", req.Name).Count(&cnt).Error; err != nil {
		return gatewayResp.SkillView{}, err
	}
	if cnt > 0 {
		return gatewayResp.SkillView{}, errors.New("该技能名称已存在")
	}
	row, err := buildSkillRow(req, 0)
	if err != nil {
		return gatewayResp.SkillView{}, err
	}
	row.CreateBy = createBy
	row.UpdateBy = createBy
	if err := global.OPS_DB.WithContext(ctx).Create(row).Error; err != nil {
		return gatewayResp.SkillView{}, err
	}
	return toSkillView(*row), nil
}

// UpdateSkill 修改元数据(不改发布配置，发布走 PublishSkill)：名称唯一(排除自身)+
// 启停翻转型联动授权对齐(停用全量回收/恢复按发布档重授)。
func (s *SkillService) UpdateSkill(ctx context.Context, req gatewayReq.SkillOperateParams, updateBy int64) (gatewayResp.SkillView, error) {
	if req.SkillId == 0 {
		return gatewayResp.SkillView{}, errors.New("技能ID不能为空")
	}
	var old gateway.Skill
	if err := global.OPS_DB.WithContext(ctx).Where("skill_id = ?", req.SkillId).First(&old).Error; err != nil {
		return gatewayResp.SkillView{}, errors.New("Skill 不存在")
	}
	if req.Name != old.Name {
		if req.Name == "" {
			return gatewayResp.SkillView{}, errors.New("技能名称不能为空")
		}
		var cnt int64
		if err := global.OPS_DB.WithContext(ctx).Model(&gateway.Skill{}).
			Where("name = ? AND skill_id <> ?", req.Name, req.SkillId).Count(&cnt).Error; err != nil {
			return gatewayResp.SkillView{}, err
		}
		if cnt > 0 {
			return gatewayResp.SkillView{}, errors.New("该技能名称已存在")
		}
	}
	row, err := buildSkillRow(req, req.SkillId)
	if err != nil {
		return gatewayResp.SkillView{}, err
	}
	isActive := old.IsActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	activeToggled := isActive != old.IsActive

	err = global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&gateway.Skill{}).Where("skill_id = ?", req.SkillId).Updates(map[string]any{
			"name":                  row.Name,
			"version":               row.Version,
			"author":                row.Author,
			"description":           row.Description,
			"category":              row.Category,
			"tags":                  row.Tags,
			"icon_url":              row.IconUrl,
			"documentation_url":     row.DocumentationUrl,
			"agent_install_prompt":  row.AgentInstallPrompt,
			"usage_instructions":    row.UsageInstructions,
			"is_active":             isActive,
			"update_by":             updateBy,
		}).Error; err != nil {
			return err
		}
		// 启停翻转型联动：停用=全量回收；恢复=按当前发布档重授(与 PublishSkill 收尾同口径)
		if activeToggled {
			row.IsActive = isActive
			row.IsPublished = old.IsPublished
			row.VisibilityType = old.VisibilityType
			row.RequiresApproval = old.RequiresApproval
			alignSkillAuthorization(ctx, tx, row, nil, nil)
		}
		return nil
	})
	if err != nil {
		return gatewayResp.SkillView{}, err
	}
	var fresh gateway.Skill
	if err := global.OPS_DB.WithContext(ctx).Where("skill_id = ?", req.SkillId).First(&fresh).Error; err != nil {
		return gatewayResp.SkillView{}, err
	}
	return toSkillView(fresh), nil
}

// DeleteSkills 批量删除：事务内全量回收主 Key 授权+物理删投影行+软删主行，
// 尽力删 zip 文件(失败仅告警，行照删)。使用日志保留(审计)。
func (s *SkillService) DeleteSkills(ctx context.Context, ids []int64) error {
	for _, id := range ids {
		var row gateway.Skill
		if err := global.OPS_DB.WithContext(ctx).Where("skill_id = ?", id).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return err
		}
		err := global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			alignSkillAuthorization(ctx, tx, &row, nil, nil) // 全量回收(行将删，keep=nil)
			if err := tx.Unscoped().Where("skill_id = ?", id).Delete(&gateway.SkillVisibility{}).Error; err != nil {
				return err
			}
			if err := tx.Unscoped().Where("skill_id = ?", id).Delete(&gateway.SkillVisibilityUser{}).Error; err != nil {
				return err
			}
			return tx.Delete(&row).Error
		})
		if err != nil {
			return err
		}
		removeSkillZip(ctx, row.ZipFilename)
	}
	return nil
}

// GetSkillPublish 查 Skill 发布设置(含 selected/user 模式的可见部门与可见用户)。
func (s *SkillService) GetSkillPublish(ctx context.Context, id int64) (gatewayResp.SkillPublishView, error) {
	var row gateway.Skill
	if err := global.OPS_DB.WithContext(ctx).Where("skill_id = ?", id).First(&row).Error; err != nil {
		return gatewayResp.SkillPublishView{}, errors.New("Skill 不存在")
	}
	view := gatewayResp.SkillPublishView{
		SkillId:          row.SkillId,
		IsPublished:      row.IsPublished,
		VisibilityType:   row.VisibilityType,
		RequiresApproval: row.RequiresApproval,
		DepartmentIds:    []int64{},
		UserIds:          []int64{},
	}
	view.DepartmentIds = append(view.DepartmentIds, skillVisibleDeptIds(global.OPS_DB.WithContext(ctx), id)...)
	view.UserIds = append(view.UserIds, skillVisibleUserIds(global.OPS_DB.WithContext(ctx), id)...)
	return view, nil
}

// PublishSkill 发布设置：四档可见性校验与投影行重建(物理删+插)同 PublishModel/
// PublishMCPServer；授权对齐走 alignSkillAuthorization。
func (s *SkillService) PublishSkill(ctx context.Context, req gatewayReq.SkillPublishParams, updateBy int64) error {
	if req.SkillId == 0 {
		return errors.New("技能ID不能为空")
	}
	visibility := req.VisibilityType
	if visibility == "" {
		visibility = gateway.VisibilityTypeAll
	}
	if visibility != gateway.VisibilityTypeAll && visibility != gateway.VisibilityTypeSelected && visibility != gateway.VisibilityTypeUser && visibility != gateway.VisibilityTypeMixed {
		return errors.New("可见范围取值非法(all/selected/user/mixed)")
	}
	if req.IsPublished && visibility == gateway.VisibilityTypeSelected && len(req.DepartmentIds) == 0 {
		return errors.New("指定部门可见时必须选择至少一个部门")
	}
	if req.IsPublished && visibility == gateway.VisibilityTypeUser && len(req.UserIds) == 0 {
		return errors.New("指定用户可见时必须选择至少一个用户")
	}
	if req.IsPublished && visibility == gateway.VisibilityTypeMixed && len(req.DepartmentIds) == 0 && len(req.UserIds) == 0 {
		return errors.New("部门+用户可见时必须选择至少一个部门或用户")
	}
	var row gateway.Skill
	if err := global.OPS_DB.WithContext(ctx).Where("skill_id = ?", req.SkillId).First(&row).Error; err != nil {
		return errors.New("Skill 不存在")
	}
	if req.IsPublished && row.ZipFilename == "" {
		return errors.New("尚未上传技能包(zip)，不能发布")
	}
	if len(req.DepartmentIds) > 0 {
		var cnt int64
		if err := global.OPS_DB.WithContext(ctx).Model(&system.SysDepartment{}).
			Where("dept_id IN ?", req.DepartmentIds).Count(&cnt).Error; err != nil {
			return err
		}
		if cnt != int64(len(req.DepartmentIds)) {
			return errors.New("可见部门列表包含不存在的部门")
		}
	}
	if len(req.UserIds) > 0 {
		var cnt int64
		if err := global.OPS_DB.WithContext(ctx).Model(&system.SysUser{}).
			Where("id IN ?", req.UserIds).Count(&cnt).Error; err != nil {
			return err
		}
		if cnt != int64(len(req.UserIds)) {
			return errors.New("可见用户列表包含不存在的用户")
		}
	}
	return global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&gateway.Skill{}).Where("skill_id = ?", req.SkillId).Updates(map[string]any{
			"is_published":      req.IsPublished,
			"visibility_type":   visibility,
			"requires_approval": req.RequiresApproval,
			"update_by":         updateBy,
		}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("skill_id = ?", req.SkillId).Delete(&gateway.SkillVisibility{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("skill_id = ?", req.SkillId).Delete(&gateway.SkillVisibilityUser{}).Error; err != nil {
			return err
		}
		// mixed 档两张投影表都写(允许只填一类，空表跳过)；查询侧 OR 语义同 PublishModel
		if req.IsPublished && visibilityUsesDept(visibility) && len(req.DepartmentIds) > 0 {
			rows := make([]gateway.SkillVisibility, 0, len(req.DepartmentIds))
			for _, deptId := range req.DepartmentIds {
				rows = append(rows, gateway.SkillVisibility{SkillId: req.SkillId, DepartmentId: deptId})
			}
			if err := tx.Create(&rows).Error; err != nil {
				return err
			}
		}
		if req.IsPublished && visibilityUsesUser(visibility) && len(req.UserIds) > 0 {
			userRows := make([]gateway.SkillVisibilityUser, 0, len(req.UserIds))
			for _, userId := range req.UserIds {
				userRows = append(userRows, gateway.SkillVisibilityUser{SkillId: req.SkillId, UserId: userId})
			}
			if err := tx.Create(&userRows).Error; err != nil {
				return err
			}
		}
		row.IsPublished = req.IsPublished
		row.VisibilityType = visibility
		row.RequiresApproval = req.RequiresApproval
		alignSkillAuthorization(ctx, tx, &row, req.DepartmentIds, req.UserIds)
		return nil
	})
}

// ----------------------------------------------------------------------------
// zip 包上传与下载
// ----------------------------------------------------------------------------

// UploadSkillPackage 上传/替换技能包(multipart)：校验 .zip 后缀与大小上限 →
// 存 uploads/skills/{skillId}_{时间戳}.zip → 更新行 zip 三字段 → 删旧包。
// 发布中的 Skill 允许换包(版本升级场景)；包替换不触发授权变化。
func (s *SkillService) UploadSkillPackage(ctx context.Context, skillId int64, fileHeader *multipart.FileHeader, updateBy int64) (gatewayResp.SkillView, error) {
	var row gateway.Skill
	if err := global.OPS_DB.WithContext(ctx).Where("skill_id = ?", skillId).First(&row).Error; err != nil {
		return gatewayResp.SkillView{}, errors.New("Skill 不存在")
	}
	if !ValidSkillUploadFilename(fileHeader.Filename) {
		return gatewayResp.SkillView{}, errors.New("仅支持 .zip 文件(文件名不含路径)")
	}
	if fileHeader.Size <= 0 {
		return gatewayResp.SkillView{}, errors.New("上传文件为空")
	}
	if fileHeader.Size > MaxSkillZipBytes {
		return gatewayResp.SkillView{}, fmt.Errorf("技能包超过大小上限(%dMB)", MaxSkillZipBytes>>20)
	}
	filename := SkillZipFilename(skillId, time.Now())
	if err := os.MkdirAll(skillStoreDir, 0o755); err != nil {
		return gatewayResp.SkillView{}, fmt.Errorf("创建存储目录失败: %w", err)
	}
	dst := filepath.Join(skillStoreDir, filename)
	src, err := fileHeader.Open()
	if err != nil {
		return gatewayResp.SkillView{}, fmt.Errorf("读取上传文件失败: %w", err)
	}
	defer src.Close()
	out, err := os.Create(dst)
	if err != nil {
		return gatewayResp.SkillView{}, fmt.Errorf("保存技能包失败: %w", err)
	}
	defer out.Close()
	if _, err := io.Copy(out, src); err != nil {
		_ = os.Remove(dst)
		return gatewayResp.SkillView{}, fmt.Errorf("保存技能包失败: %w", err)
	}
	out.Close()
	// 落盘后解包校验结构(须含 SKILL.md)：不合法即删临时文件拒绝入库，
	// 避免坏包占住 zip_filename 让后续发布/下载拿到无效内容。
	if err := ValidateSkillZipStructure(dst); err != nil {
		_ = os.Remove(dst)
		return gatewayResp.SkillView{}, err
	}
	oldZip := row.ZipFilename
	if err := global.OPS_DB.WithContext(ctx).Model(&gateway.Skill{}).Where("skill_id = ?", skillId).
		Updates(map[string]any{
			"zip_filename":    filename,
			"zip_origin_name": fileHeader.Filename,
			"zip_size":        fileHeader.Size,
			"update_by":       updateBy,
		}).Error; err != nil {
		_ = os.Remove(dst)
		return gatewayResp.SkillView{}, err
	}
	if oldZip != "" && oldZip != filename {
		removeSkillZip(ctx, oldZip)
	}
	var fresh gateway.Skill
	if err := global.OPS_DB.WithContext(ctx).Where("skill_id = ?", skillId).First(&fresh).Error; err != nil {
		return gatewayResp.SkillView{}, err
	}
	return toSkillView(fresh), nil
}

// DownloadSkill 下载技能包(用户侧)：须启用+已发布+已上传包；需审批 Skill 须持有
// 授权(主 Key skills 含锚点——审批落主 Key，loadMainKey 顺带自愈)。计数与
// usage log 尽力而为(失败不阻塞下载)。返回文件路径与下载回显名。
func (s *SkillService) DownloadSkill(ctx context.Context, id, userId int64) (filePath, originName string, err error) {
	var row gateway.Skill
	if err := global.OPS_DB.WithContext(ctx).Where("skill_id = ?", id).First(&row).Error; err != nil {
		return "", "", errors.New("Skill 不存在")
	}
	if !row.IsActive || !row.IsPublished {
		return "", "", errors.New("该技能未开放下载")
	}
	if row.ZipFilename == "" {
		return "", "", errors.New("该技能尚未上传技能包")
	}
	if row.RequiresApproval {
		k, err := loadMainKey(ctx, userId)
		if err != nil {
			return "", "", err
		}
		if k == nil || !sliceContains(jsonToSlice(k.Skills), SkillIdentityOf(row)) {
			return "", "", errors.New("该技能需审批授权后方可下载，请先在广场申请")
		}
	}
	p := filepath.Join(skillStoreDir, row.ZipFilename)
	if _, err := os.Stat(p); err != nil {
		return "", "", errors.New("技能包文件缺失，请联系管理员")
	}
	name := row.ZipOriginName
	if name == "" {
		name = row.Name + ".zip"
	}
	// 计数+留痕(尽力而为)
	if err := global.OPS_DB.WithContext(ctx).Model(&gateway.Skill{}).Where("skill_id = ?", id).
		UpdateColumn("install_count", gorm.Expr("install_count + 1")).Error; err != nil {
		logger.WithCtx(ctx).Mod("gateway").Err(err).Field("skillId", id).Warn("skill: 下载计数失败")
	}
	if err := global.OPS_DB.WithContext(ctx).Create(&gateway.SkillUsageLog{
		UserId:  userId,
		SkillId: id,
		Action:  gateway.SkillActionDownload,
	}).Error; err != nil {
		logger.WithCtx(ctx).Mod("gateway").Err(err).Field("skillId", id).Warn("skill: 使用日志落库失败")
	}
	return p, name, nil
}

// AdminDownloadSkill 下载技能包(管理端)：仅校验存在与文件在盘，不做用户侧的
// 发布/授权校验(管理员常需核对未发布草稿的包)；不计下载次数、不留使用日志。
func (s *SkillService) AdminDownloadSkill(ctx context.Context, id int64) (filePath, originName string, err error) {
	var row gateway.Skill
	if err := global.OPS_DB.WithContext(ctx).Where("skill_id = ?", id).First(&row).Error; err != nil {
		return "", "", errors.New("Skill 不存在")
	}
	if row.ZipFilename == "" {
		return "", "", errors.New("该技能尚未上传技能包")
	}
	p := filepath.Join(skillStoreDir, row.ZipFilename)
	if _, err := os.Stat(p); err != nil {
		return "", "", errors.New("技能包文件缺失，请联系管理员")
	}
	name := row.ZipOriginName
	if name == "" {
		name = row.Name + ".zip"
	}
	return p, name, nil
}

// removeSkillZip 删除 zip 存储文件(尽力而为，失败仅告警)。
func removeSkillZip(ctx context.Context, filename string) {
	if filename == "" {
		return
	}
	p := filepath.Join(skillStoreDir, filename)
	// 二次校验：存储键由平台生成({id}_{ts}.zip)，防御异常数据携带路径段
	if filepath.Dir(p) != filepath.Clean(skillStoreDir) {
		logger.WithCtx(ctx).Mod("gateway").Field("zip", filename).Warn("skill: 异常存储键拒绝删除")
		return
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		logger.WithCtx(ctx).Mod("gateway").Err(err).Field("zip", filename).Warn("skill: 删除技能包文件失败")
	}
}

// ----------------------------------------------------------------------------
// 用户侧(广场)与管理端下拉
// ----------------------------------------------------------------------------

// GetAvailableSkills 可授权 Skill(管理员 Key 授权下拉用)：全部启用 Skill，
// 不按可见性过滤(与 GetAvailableModels/GetAvailableMcps 同语义)。
func (s *SkillService) GetAvailableSkills(ctx context.Context) ([]gatewayResp.AvailableSkillView, error) {
	return s.listSkillsAsAvailable(ctx, func(q *gorm.DB) *gorm.DB { return q })
}

// GetSkillCategories 分类去重列表(管理端下拉受控数据源)：非空分类去重升序，
// 轻量受控——不建分类表，选项随存量数据生长，新分类经 NSelect tag 输入产生。
func (s *SkillService) GetSkillCategories(ctx context.Context) ([]string, error) {
	var cats []string
	if err := global.OPS_DB.WithContext(ctx).Model(&gateway.Skill{}).
		Where("category <> ''").Distinct("category").
		Order("category").Pluck("category", &cats).Error; err != nil {
		return nil, err
	}
	return cats, nil
}

// GetActiveSkills 用户侧可见 Skill(按发布可见性三档过滤)：广场数据源，
// 与 GetActiveModels/GetActiveMcps 同口径。
func (s *SkillService) GetActiveSkills(ctx context.Context, userId int64) ([]gatewayResp.AvailableSkillView, error) {
	db := global.OPS_DB.WithContext(ctx)
	return s.listSkillsAsAvailable(ctx, func(q *gorm.DB) *gorm.DB {
		return visibleSkillScope(q, userId, userDeptIdOf(db, userId))
	})
}

// listSkillsAsAvailable 启用已发布 Skill → AvailableSkillView 贫字段列表。
func (s *SkillService) listSkillsAsAvailable(ctx context.Context, scope func(*gorm.DB) *gorm.DB) ([]gatewayResp.AvailableSkillView, error) {
	var rows []gateway.Skill
	q := global.OPS_DB.WithContext(ctx).Model(&gateway.Skill{}).
		Where("is_active = ? AND is_published = ?", true, true)
	if err := scope(q).Order("skill_id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]gatewayResp.AvailableSkillView, 0, len(rows))
	for i := range rows {
		r := rows[i]
		list = append(list, gatewayResp.AvailableSkillView{
			SkillId:          r.SkillId,
			Name:             r.Name,
			Version:          r.Version,
			Author:           r.Author,
			Description:      r.Description,
			Category:         r.Category,
			Tags:             UnmarshalSkillTags(r.Tags),
			IconUrl:          r.IconUrl,
			DocumentationUrl: r.DocumentationUrl,
			RequiresApproval: r.RequiresApproval,
			HasPackage:       r.ZipFilename != "",
			InstallCount:     r.InstallCount,
		})
	}
	return list, nil
}

// ----------------------------------------------------------------------------
// 使用日志
// ----------------------------------------------------------------------------

// GetSkillUsageList 使用日志分页(管理端)：回填用户名/技能名(各一次 IN 查询防 N+1)。
func (s *SkillService) GetSkillUsageList(ctx context.Context, q gatewayReq.SkillUsageSearch) (list []gatewayResp.SkillUsageView, total int64, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&gateway.SkillUsageLog{})
	if q.SkillId != 0 {
		db = db.Where("skill_id = ?", q.SkillId)
	}
	if q.UserId != 0 {
		db = db.Where("user_id = ?", q.UserId)
	}
	if q.Action != "" {
		db = db.Where("action = ?", q.Action)
	}
	var rows []gateway.SkillUsageLog
	limit, offset := q.LimitOffset()
	if limit > 0 {
		err = db.Count(&total).Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error
	} else {
		err = db.Count(&total).Order("id DESC").Find(&rows).Error
	}
	if err != nil {
		return nil, 0, err
	}
	userNames, skillNames := s.usageNames(ctx, rows)
	list = make([]gatewayResp.SkillUsageView, 0, len(rows))
	for i := range rows {
		list = append(list, gatewayResp.SkillUsageView{
			Id:         rows[i].Id,
			UserId:     rows[i].UserId,
			UserName:   userNames[rows[i].UserId],
			SkillId:    rows[i].SkillId,
			SkillName:  skillNames[rows[i].SkillId],
			Action:     rows[i].Action,
			CreateTime: rows[i].CreatedAt,
		})
	}
	return list, total, nil
}

// usageNames 批量回填用户名/技能名 map(各一次 IN 查询)。
func (s *SkillService) usageNames(ctx context.Context, rows []gateway.SkillUsageLog) (userNames, skillNames map[int64]string) {
	userNames = map[int64]string{}
	skillNames = map[int64]string{}
	userIds := map[int64]struct{}{}
	skillIds := map[int64]struct{}{}
	for i := range rows {
		userIds[rows[i].UserId] = struct{}{}
		skillIds[rows[i].SkillId] = struct{}{}
	}
	if len(userIds) > 0 {
		ids := make([]int64, 0, len(userIds))
		for id := range userIds {
			ids = append(ids, id)
		}
		var us []system.SysUser
		if err := global.OPS_DB.WithContext(ctx).Select("id, nick_name").Where("id IN ?", ids).Find(&us).Error; err == nil {
			for _, u := range us {
				userNames[u.UserId] = u.NickName
			}
		}
	}
	if len(skillIds) > 0 {
		ids := make([]int64, 0, len(skillIds))
		for id := range skillIds {
			ids = append(ids, id)
		}
		var ss []gateway.Skill
		if err := global.OPS_DB.WithContext(ctx).Select("skill_id, name").Where("skill_id IN ?", ids).Find(&ss).Error; err == nil {
			for _, r := range ss {
				skillNames[r.SkillId] = r.Name
			}
		}
	}
	return
}

// ----------------------------------------------------------------------------
// 内部工具
// ----------------------------------------------------------------------------

// buildSkillRow 操作参数 → 待落库行(校验/归一/标签清洗)。
func buildSkillRow(req gatewayReq.SkillOperateParams, skillId int64) (*gateway.Skill, error) {
	version, err := NormalizeSkillVersion(req.Version)
	if err != nil {
		return nil, err
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	row := &gateway.Skill{
		SkillId:            skillId,
		Name:               req.Name,
		Version:            version,
		Author:             req.Author,
		Description:        req.Description,
		Category:           NormalizeSkillCategory(req.Category),
		Tags:               MarshalSkillTags(CleanSkillTags(req.Tags)),
		IconUrl:            req.IconUrl,
		DocumentationUrl:   req.DocumentationUrl,
		AgentInstallPrompt: req.AgentInstallPrompt,
		UsageInstructions:  req.UsageInstructions,
		IsActive:           isActive,
	}
	return row, nil
}

// toSkillView 模型转出网视图(tags JSONB → 字符串数组)。
func toSkillView(row gateway.Skill) gatewayResp.SkillView {
	return gatewayResp.SkillView{Skill: row, Tags: UnmarshalSkillTags(row.Tags)}
}
