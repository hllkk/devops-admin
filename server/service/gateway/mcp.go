package gateway

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/gateway"
	gatewayReq "github.com/hllkk/devops-admin/server/model/gateway/request"
	gatewayResp "github.com/hllkk/devops-admin/server/model/gateway/response"
	"github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/utils/litellm"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// McpService MCP 服务器管理(P2·AI 市场)：注册/发布/授权/工具/健康。
// 设计对齐模型发布三档可见性(all/selected/user)+需审批申请流(公共底座 resource_type=mcp)，
// 规避 AIHelms 三坑：①credentials 明文落库回显→AES 密文+掩码视图；②建站 allow_all_keys=true
// 绕过平台授权→恒 false，Key 的 allowed_mcp_servers 是唯一授权凭证；③授权只加不收→
// 发布/停用/删除全走双向对齐(sync+revoke，回收扫停用 Key)。
type McpService struct{}

// McpSvc 包内共享实例(无状态服务，供 AiKeyService 等同包服务复用)。
var McpSvc = McpService{}

// ----------------------------------------------------------------------------
// 用户侧可见口径(主 Key 自愈差集源/广场/审批校验共用)
// ----------------------------------------------------------------------------

// visibleMcpScope 按发布可见性给 MCP 查询加过滤条件(与 visibleModelScope 同构)：
// all 档直通/selected 档命中部门投影(主部门∪多部门)/user 档命中用户投影。
// 投影表带软删基座，EXISTS 子查询显式排除软删行(发布时物理删+插，活行恒 deleted_at IS NULL)。
func visibleMcpScope(db *gorm.DB, userId, deptId int64) *gorm.DB {
	return db.Where(
		`visibility_type = ?
		OR EXISTS(SELECT 1 FROM gateway_mcp_visibility v
			WHERE v.mcp_server_id = gateway_mcp_server.mcp_server_id AND v.deleted_at IS NULL
			AND (v.department_id = ?
				OR v.department_id IN (SELECT ud.sys_department_id FROM sys_user_departments ud WHERE ud.sys_user_id = ?)))
		OR EXISTS(SELECT 1 FROM gateway_mcp_visibility_user u
			WHERE u.mcp_server_id = gateway_mcp_server.mcp_server_id AND u.user_id = ? AND u.deleted_at IS NULL)`,
		gateway.VisibilityTypeAll, deptId, userId, userId,
	)
}

// visibleMcpKeys 对指定主 Key owner 可见的免审批 MCP serverName 列表(自愈差集/建主 Key
// 默认授权数据源)：个人 owner 传 (userId,主部门) 三档全生效；部门 owner 传 (0,deptId)。
func visibleMcpKeys(db *gorm.DB, userId, deptId int64) []string {
	var keys []string
	visibleMcpScope(
		db.Model(&gateway.MCPServer{}).
			Where("is_active = ? AND is_published = ? AND requires_approval = ?", true, true, false),
		userId, deptId,
	).Pluck("server_name", &keys)
	return keys
}

// approvedApplicationMcpKeys 用户已批准申请的 MCP serverName 列表(审批授权兜底)：
// MCP 须仍启用+已发布(下架/删除的授权由发布对齐回收，重新发布后自愈补回)。
func approvedApplicationMcpKeys(db *gorm.DB, userId int64) []string {
	var keys []string
	db.Table("gateway_resource_application AS a").
		Joins("JOIN gateway_mcp_server s ON s.mcp_server_id = a.resource_id AND s.deleted_at IS NULL AND s.is_active = ? AND s.is_published = ?", true, true).
		Where("a.deleted_at IS NULL AND a.user_id = ? AND a.resource_type = ? AND a.status = ?",
			userId, gateway.ApplicationResourceMcp, gateway.ApplicationStatusApproved).
		Pluck("s.server_name", &keys)
	return keys
}

// ----------------------------------------------------------------------------
// 主 Key 授权对齐(与模型 syncModelToMainKeys/revokeModelFromMainKeys 同构)
// ----------------------------------------------------------------------------

// syncMcpToMainKeys 发布免审批 MCP 时向目标活跃主 Key 集合追加 serverName 并同步
// LiteLLM(事务内，单个失败 warning 继续)。目标集合由 mainKeyScopeOf 按可见档构造。
func syncMcpToMainKeys(ctx context.Context, tx *gorm.DB, serverName string, scope func(*gorm.DB) *gorm.DB) []string {
	var keys []gateway.AiKey
	if err := scope(tx).Where("is_active = ?", true).Find(&keys).Error; err != nil {
		return []string{err.Error()}
	}
	cli := litellm.Default()
	var warnings []string
	for i := range keys {
		current := jsonToSlice(keys[i].Mcps)
		if sliceContains(current, serverName) {
			continue // 已授权
		}
		current = append(current, serverName)
		keys[i].Mcps = marshalJSONStringSlice(current)
		if err := tx.Model(&gateway.AiKey{}).Where("ai_key_id = ?", keys[i].AiKeyId).
			Update("mcps", keys[i].Mcps).Error; err != nil {
			warnings = append(warnings, fmt.Sprintf("主Key %d: %v", keys[i].AiKeyId, err))
			continue
		}
		if cli != nil && keys[i].LitellmKeyId != "" {
			if err := syncKeyToLitellm(ctx, cli, tx, &keys[i], false); err != nil {
				warnings = append(warnings, fmt.Sprintf("主Key %d: %v", keys[i].AiKeyId, err))
			}
		}
	}
	for _, w := range warnings {
		logger.WithCtx(ctx).Mod("gateway").Warn(w)
	}
	return warnings
}

// revokeMcpFromMainKeys MCP 授权对齐的减法半边：从不应再持有 serverName 的主 Key 回收
// (发布对齐/停用/删除调用)。keepScope 命中的主 Key 保留(不限启停)，nil=全部回收；
// 扫描全部主 Key 含停用(停用 Key 不回收会在重新启用后死灰复燃)。
// 场景 Key 手工授权不在此域；与 loadMainKey 自愈差集源(visibleMcpKeys)同口径，回收后自愈不回加。
func revokeMcpFromMainKeys(ctx context.Context, tx *gorm.DB, serverName string, keepScope func(*gorm.DB) *gorm.DB) []string {
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
	cli := litellm.Default()
	var warnings []string
	for i := range keys {
		if keep[keys[i].AiKeyId] {
			continue
		}
		mcps, changed := removeModelKey(jsonToSlice(keys[i].Mcps), serverName)
		if !changed {
			continue
		}
		keys[i].Mcps = marshalJSONStringSlice(mcps)
		if err := tx.Model(&gateway.AiKey{}).Where("ai_key_id = ?", keys[i].AiKeyId).
			Update("mcps", keys[i].Mcps).Error; err != nil {
			warnings = append(warnings, fmt.Sprintf("主Key %d: %v", keys[i].AiKeyId, err))
			continue
		}
		if cli != nil && keys[i].LitellmKeyId != "" {
			if err := syncKeyToLitellm(ctx, cli, tx, &keys[i], false); err != nil {
				warnings = append(warnings, fmt.Sprintf("主Key %d: %v", keys[i].AiKeyId, err))
			}
		}
	}
	for _, w := range warnings {
		logger.WithCtx(ctx).Mod("gateway").Warn(fmt.Sprintf("主Key 回收MCP %q 授权: %s", serverName, w))
	}
	return warnings
}

// alignMCPAuthorization 主 Key 授权对齐(发布/启停/更新共用收尾)：
// 发布+免审批+启用 → 按可见档 sync+revoke；否则全量 revoke(未发布/需审批/停用不自动授权)。
// publish 时传本次提交的 deptIds/userIds(投影行同事务重建)；其余场景传 nil(从投影表读现值)。
func alignMCPAuthorization(ctx context.Context, tx *gorm.DB, s *gateway.MCPServer, deptIds, userIds []int64) []string {
	if s.IsPublished && !s.RequiresApproval && s.IsActive {
		if deptIds == nil && s.VisibilityType == gateway.VisibilityTypeSelected {
			deptIds = mcpVisibleDeptIds(tx, s.McpServerId)
		}
		if userIds == nil && s.VisibilityType == gateway.VisibilityTypeUser {
			userIds = mcpVisibleUserIds(tx, s.McpServerId)
		}
		scope := mainKeyScopeOf(s.VisibilityType, deptIds, userIds)
		warnings := syncMcpToMainKeys(ctx, tx, s.ServerName, scope)
		return append(warnings, revokeMcpFromMainKeys(ctx, tx, s.ServerName, scope)...)
	}
	return revokeMcpFromMainKeys(ctx, tx, s.ServerName, nil)
}

// mcpVisibleDeptIds / mcpVisibleUserIds 读投影表现值(Update/停用路径无前端提交列表)。
func mcpVisibleDeptIds(db *gorm.DB, mcpServerId int64) []int64 {
	var ids []int64
	db.Model(&gateway.MCPVisibility{}).Where("mcp_server_id = ?", mcpServerId).Pluck("department_id", &ids)
	return ids
}

func mcpVisibleUserIds(db *gorm.DB, mcpServerId int64) []int64 {
	var ids []int64
	db.Model(&gateway.MCPVisibilityUser{}).Where("mcp_server_id = ?", mcpServerId).Pluck("user_id", &ids)
	return ids
}

// ----------------------------------------------------------------------------
// 管理员 CRUD
// ----------------------------------------------------------------------------

// GetMCPServerList 分页查 MCP 服务器列表(含掩码凭据视图 + 工具数)。
func (s *McpService) GetMCPServerList(ctx context.Context, q gatewayReq.MCPServerSearch) (list []gatewayResp.MCPServerView, total int64, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&gateway.MCPServer{})
	if q.Name != "" {
		like := "%" + q.Name + "%"
		db = db.Where("name ILIKE ? OR server_name ILIKE ?", like, like)
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
	if q.HealthStatus != "" {
		db = db.Where("health_status = ?", q.HealthStatus)
	}
	var rows []gateway.MCPServer
	limit, offset := q.LimitOffset()
	if limit > 0 {
		err = db.Count(&total).Order("mcp_server_id DESC").Limit(limit).Offset(offset).Find(&rows).Error
	} else {
		err = db.Count(&total).Order("mcp_server_id DESC").Find(&rows).Error
	}
	if err != nil {
		return nil, 0, err
	}
	list = make([]gatewayResp.MCPServerView, 0, len(rows))
	for i := range rows {
		list = append(list, s.toMCPServerView(ctx, &rows[i]))
	}
	s.fillToolCounts(ctx, rows, list)
	return list, total, nil
}

// GetMCPServer 详情(含工具列表)。
func (s *McpService) GetMCPServer(ctx context.Context, id int64) (gatewayResp.MCPServerDetail, error) {
	var row gateway.MCPServer
	if err := global.OPS_DB.WithContext(ctx).Where("mcp_server_id = ?", id).First(&row).Error; err != nil {
		return gatewayResp.MCPServerDetail{}, errors.New("MCP 服务器不存在")
	}
	detail := gatewayResp.MCPServerDetail{MCPServerView: s.toMCPServerView(ctx, &row)}
	var tools []gateway.MCPTool
	if err := global.OPS_DB.WithContext(ctx).Where("mcp_server_id = ?", id).Order("mcp_tool_id").Find(&tools).Error; err != nil {
		return gatewayResp.MCPServerDetail{}, err
	}
	detail.Tools = make([]gatewayResp.MCPToolView, 0, len(tools))
	for i := range tools {
		detail.Tools = append(detail.Tools, gatewayResp.MCPToolView{MCPTool: tools[i]})
	}
	detail.ToolCount = int64(len(detail.Tools))
	return detail, nil
}

// CreateMCPServer 注册 MCP 服务器：路由名合法+唯一校验 → LiteLLM 内联推送(失败回滚，
// 自定义 server_id=gw_mcp_{雪花} 作归因锚点) → 本地落库。创建不发布(发布走 publish)。
func (s *McpService) CreateMCPServer(ctx context.Context, req gatewayReq.MCPServerOperateParams, createBy int64) (gatewayResp.MCPServerView, error) {
	row, credentials, err := s.validateOperate(ctx, req, 0)
	if err != nil {
		return gatewayResp.MCPServerView{}, err
	}
	// 凭据 AES 密文落库(明文只进 LiteLLM 投影)，规避 AIHelms 明文落库坑
	if len(credentials) > 0 {
		enc, err := encryptCredentialValues(credentials)
		if err != nil {
			return gatewayResp.MCPServerView{}, fmt.Errorf("凭据加密失败: %w", err)
		}
		row.Credentials = enc
	}
	row.CreateBy = createBy
	row.UpdateBy = createBy

	cli := litellm.Default()
	err = global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(row).Error; err != nil {
			return err
		}
		if cli == nil {
			return nil // 单机模式：只落库不同步
		}
		body := buildMCPLitellmBody(row, credentials, nil, false)
		serverId, err := cli.CreateMCPServer(ctx, body)
		if err != nil {
			return fmt.Errorf("MCP 服务器同步 LiteLLM 失败: %w", err)
		}
		row.LitellmServerId = serverId
		row.LitellmSynced = true
		return tx.Model(&gateway.MCPServer{}).Where("mcp_server_id = ?", row.McpServerId).
			Update("litellm_server_id", serverId).Error
	})
	if err != nil {
		return gatewayResp.MCPServerView{}, err
	}
	view := s.toMCPServerView(ctx, row)
	view.Credentials = MaskCredentialValues(credentials)
	return view, nil
}

// UpdateMCPServer 修改：serverName 拒绝改名(LiteLLM 路由键)；凭据掩码回传保留旧明文
// (切 none=显式清空凭据)；增量更新 + 投影重推 LiteLLM；启停翻转型联动授权对齐
// (停用全量回收/恢复按发布档重授)。
func (s *McpService) UpdateMCPServer(ctx context.Context, req gatewayReq.MCPServerOperateParams, updateBy int64) (gatewayResp.MCPServerView, error) {
	if req.McpServerId == 0 {
		return gatewayResp.MCPServerView{}, errors.New("MCP 服务器ID不能为空")
	}
	var old gateway.MCPServer
	if err := global.OPS_DB.WithContext(ctx).Where("mcp_server_id = ?", req.McpServerId).First(&old).Error; err != nil {
		return gatewayResp.MCPServerView{}, errors.New("MCP 服务器不存在")
	}
	if req.ServerName != "" && req.ServerName != old.ServerName {
		return gatewayResp.MCPServerView{}, errors.New("路由名不可修改(客户端接入配置与授权锚点均基于它)")
	}
	row, incomingCredentials, err := s.validateOperate(ctx, req, req.McpServerId)
	if err != nil {
		return gatewayResp.MCPServerView{}, err
	}
	// 凭据合并语义：none=清空；nil=未提交凭据域(保留旧值)；掩码回传=保留旧明文，新值=覆盖
	oldCredentials, err := s.decryptCredentials(old.Credentials)
	if err != nil {
		return gatewayResp.MCPServerView{}, err
	}
	var credentials map[string]any
	switch {
	case row.AuthType == gateway.MCPAuthNone:
		credentials = nil
	case incomingCredentials == nil:
		credentials = oldCredentials
	default:
		credentials = MergeMCPCredentials(oldCredentials, incomingCredentials)
	}
	if len(credentials) > 0 {
		enc, err := encryptCredentialValues(credentials)
		if err != nil {
			return gatewayResp.MCPServerView{}, fmt.Errorf("凭据加密失败: %w", err)
		}
		row.Credentials = enc
	}
	isActive := old.IsActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	row.UpdateBy = updateBy
	activeToggled := isActive != old.IsActive

	// 工具级定价 map(mcp_info 投影用)
	toolCosts := s.toolCostMap(ctx, row.McpServerId)

	cli := litellm.Default()
	err = global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 显式增量更新(不用 Save，避免审计字段被零值覆盖)
		if err := tx.Model(&gateway.MCPServer{}).Where("mcp_server_id = ?", row.McpServerId).Updates(map[string]any{
			"name":                  row.Name,
			"url":                   row.Url,
			"transport":             row.Transport,
			"auth_type":             row.AuthType,
			"credentials":           row.Credentials,
			"description":           row.Description,
			"instructions":          row.Instructions,
			"category":              row.Category,
			"author":                row.Author,
			"icon_url":              row.IconUrl,
			"documentation_url":     row.DocumentationUrl,
			"billing_type":          row.BillingType,
			"external_cost_per_call": row.ExternalCostPerCall,
			"is_active":             isActive,
			"update_by":             updateBy,
		}).Error; err != nil {
			return err
		}
		if cli != nil && old.LitellmServerId != "" {
			body := buildMCPLitellmBody(row, credentials, toolCosts, true)
			body["server_id"] = old.LitellmServerId
			if err := cli.UpdateMCPServer(ctx, body); err != nil {
				return fmt.Errorf("MCP 服务器同步 LiteLLM 失败: %w", err)
			}
		}
		// 启停翻转型联动：停用=全量回收(免得停用的 MCP 还能被已授权 Key 调用)；
		// 恢复=按当前发布档重授(与 PublishModel 收尾同口径)
		if activeToggled {
			row.IsActive = isActive
			alignMCPAuthorization(ctx, tx, row, nil, nil)
		}
		return nil
	})
	if err != nil {
		return gatewayResp.MCPServerView{}, err
	}
	// 回填终态(视图与联动基于最新行)
	row.LitellmServerId = old.LitellmServerId
	row.LitellmSynced = old.LitellmSynced
	row.IsPublished = old.IsPublished
	row.VisibilityType = old.VisibilityType
	row.RequiresApproval = old.RequiresApproval
	row.HealthStatus = old.HealthStatus
	row.LastHealthCheck = old.LastHealthCheck
	row.HealthCheckError = old.HealthCheckError
	row.IsActive = isActive
	view := s.toMCPServerView(ctx, row)
	view.Credentials = MaskCredentialValues(credentials)
	return view, nil
}

// DeleteMCPServers 批量删除：逐个先删 LiteLLM(失败该条中止本地不动)→事务内全量回收
// 主 Key 授权+本地软删(工具/可见性行物理删)。LiteLLM 未同步(单机模式)直接本地删。
func (s *McpService) DeleteMCPServers(ctx context.Context, ids []int64) error {
	for _, id := range ids {
		var row gateway.MCPServer
		if err := global.OPS_DB.WithContext(ctx).Where("mcp_server_id = ?", id).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return err
		}
		cli := litellm.Default()
		if cli != nil && row.LitellmServerId != "" {
			if err := cli.DeleteMCPServer(ctx, row.LitellmServerId); err != nil {
				return fmt.Errorf("MCP 服务器 %s 从 LiteLLM 删除失败: %w", row.ServerName, err)
			}
		}
		err := global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			alignMCPAuthorization(ctx, tx, &row, nil, nil) // 全量回收(行将删，keep=nil)
			if err := tx.Unscoped().Where("mcp_server_id = ?", id).Delete(&gateway.MCPTool{}).Error; err != nil {
				return err
			}
			if err := tx.Unscoped().Where("mcp_server_id = ?", id).Delete(&gateway.MCPVisibility{}).Error; err != nil {
				return err
			}
			if err := tx.Unscoped().Where("mcp_server_id = ?", id).Delete(&gateway.MCPVisibilityUser{}).Error; err != nil {
				return err
			}
			return tx.Delete(&row).Error
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// GetMCPPublish 查 MCP 发布设置(含 selected/user 模式的可见部门与可见用户)。
func (s *McpService) GetMCPPublish(ctx context.Context, id int64) (gatewayResp.MCPPublishView, error) {
	var row gateway.MCPServer
	if err := global.OPS_DB.WithContext(ctx).Where("mcp_server_id = ?", id).First(&row).Error; err != nil {
		return gatewayResp.MCPPublishView{}, errors.New("MCP 服务器不存在")
	}
	view := gatewayResp.MCPPublishView{
		McpServerId:      row.McpServerId,
		IsPublished:      row.IsPublished,
		VisibilityType:   row.VisibilityType,
		RequiresApproval: row.RequiresApproval,
		DepartmentIds:    []int64{},
		UserIds:          []int64{},
	}
	view.DepartmentIds = append(view.DepartmentIds, mcpVisibleDeptIds(global.OPS_DB.WithContext(ctx), id)...)
	view.UserIds = append(view.UserIds, mcpVisibleUserIds(global.OPS_DB.WithContext(ctx), id)...)
	return view, nil
}

// PublishMCPServer 发布设置：三档可见性校验与投影行重建(物理删+插)同 PublishModel；
// serverName 恒非空(创建必填唯一)，授权对齐无"缺路由名"分支。
func (s *McpService) PublishMCPServer(ctx context.Context, req gatewayReq.MCPPublishParams, updateBy int64) error {
	if req.McpServerId == 0 {
		return errors.New("MCP 服务器ID不能为空")
	}
	visibility := req.VisibilityType
	if visibility == "" {
		visibility = gateway.VisibilityTypeAll
	}
	if visibility != gateway.VisibilityTypeAll && visibility != gateway.VisibilityTypeSelected && visibility != gateway.VisibilityTypeUser {
		return errors.New("可见范围取值非法(all/selected/user)")
	}
	if req.IsPublished && visibility == gateway.VisibilityTypeSelected && len(req.DepartmentIds) == 0 {
		return errors.New("指定部门可见时必须选择至少一个部门")
	}
	if req.IsPublished && visibility == gateway.VisibilityTypeUser && len(req.UserIds) == 0 {
		return errors.New("指定用户可见时必须选择至少一个用户")
	}
	var row gateway.MCPServer
	if err := global.OPS_DB.WithContext(ctx).Where("mcp_server_id = ?", req.McpServerId).First(&row).Error; err != nil {
		return errors.New("MCP 服务器不存在")
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
		// SysUser 主键 DB 列复用 id(gorm column:id)，非 user_id
		if err := global.OPS_DB.WithContext(ctx).Model(&system.SysUser{}).
			Where("id IN ?", req.UserIds).Count(&cnt).Error; err != nil {
			return err
		}
		if cnt != int64(len(req.UserIds)) {
			return errors.New("可见用户列表包含不存在的用户")
		}
	}
	return global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&gateway.MCPServer{}).Where("mcp_server_id = ?", req.McpServerId).Updates(map[string]any{
			"is_published":      req.IsPublished,
			"visibility_type":   visibility,
			"requires_approval": req.RequiresApproval,
			"update_by":         updateBy,
		}).Error; err != nil {
			return err
		}
		// 可见性行重建(物理删：软删行会占唯一索引挡住同组合重新发布)
		if err := tx.Unscoped().Where("mcp_server_id = ?", req.McpServerId).Delete(&gateway.MCPVisibility{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("mcp_server_id = ?", req.McpServerId).Delete(&gateway.MCPVisibilityUser{}).Error; err != nil {
			return err
		}
		if req.IsPublished && visibility == gateway.VisibilityTypeSelected {
			rows := make([]gateway.MCPVisibility, 0, len(req.DepartmentIds))
			for _, deptId := range req.DepartmentIds {
				rows = append(rows, gateway.MCPVisibility{McpServerId: req.McpServerId, DepartmentId: deptId})
			}
			if err := tx.Create(&rows).Error; err != nil {
				return err
			}
		}
		if req.IsPublished && visibility == gateway.VisibilityTypeUser {
			userRows := make([]gateway.MCPVisibilityUser, 0, len(req.UserIds))
			for _, userId := range req.UserIds {
				userRows = append(userRows, gateway.MCPVisibilityUser{McpServerId: req.McpServerId, UserId: userId})
			}
			if err := tx.Create(&userRows).Error; err != nil {
				return err
			}
		}
		// 主 Key 授权双向对齐(与 PublishModel 同口径)：发布免审批 → 按可见档追加+回收
		// 范围外持有者；取消发布/转需审批 → 全量回收。需审批 MCP 走申请流(resource_type=mcp)。
		row.IsPublished = req.IsPublished
		row.VisibilityType = visibility
		row.RequiresApproval = req.RequiresApproval
		alignMCPAuthorization(ctx, tx, &row, req.DepartmentIds, req.UserIds)
		return nil
	})
}

// ----------------------------------------------------------------------------
// 工具与健康
// ----------------------------------------------------------------------------

// RefreshMCPTools 从远端拉全量工具重建本地表(按 tool_name 保留计费配置)并重推 mcp_info。
// 远端拉取失败(不可达/鉴权失败)报错返回，本地不动。
func (s *McpService) RefreshMCPTools(ctx context.Context, id int64) ([]gatewayResp.MCPToolView, error) {
	var row gateway.MCPServer
	if err := global.OPS_DB.WithContext(ctx).Where("mcp_server_id = ?", id).First(&row).Error; err != nil {
		return nil, errors.New("MCP 服务器不存在")
	}
	cli := litellm.Default()
	if cli == nil {
		return nil, errors.New("LiteLLM 未启用，无法拉取工具列表")
	}
	credentials, err := s.decryptCredentials(row.Credentials)
	if err != nil {
		return nil, err
	}
	remote, err := cli.ListMCPToolsFromServer(ctx, row.Url, litellmMCPTransport(row.Transport), row.AuthType, credentials)
	if err != nil {
		return nil, fmt.Errorf("拉取工具列表失败: %w", err)
	}
	if len(remote) == 0 {
		return nil, errors.New("远端未返回任何工具，请检查 MCP 端点与传输协议")
	}

	var existing []gateway.MCPTool
	if err := global.OPS_DB.WithContext(ctx).Where("mcp_server_id = ?", id).Find(&existing).Error; err != nil {
		return nil, err
	}
	billing := CollectMCPToolBilling(existing)
	tools := BuildMCPTools(id, row.ServerName, remote, billing)

	err = global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 工具是远端投影数据：物理删重建(软删行会占 (server_id, tool_name) 唯一索引)
		if err := tx.Unscoped().Where("mcp_server_id = ?", id).Delete(&gateway.MCPTool{}).Error; err != nil {
			return err
		}
		if len(tools) > 0 {
			if err := tx.Create(&tools).Error; err != nil {
				return err
			}
		}
		// 工具计费可能变化 → 重推 mcp_info 投影
		if cli != nil && row.LitellmServerId != "" {
			body := buildMCPLitellmBody(&row, credentials, s.toolCostMapFromRows(tools), true)
			if err := cli.UpdateMCPServer(ctx, body); err != nil {
				logger.WithCtx(ctx).Mod("gateway").Warn(fmt.Sprintf("MCP %s 工具计费投影重推失败: %v", row.ServerName, err))
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	views := make([]gatewayResp.MCPToolView, 0, len(tools))
	for i := range tools {
		views = append(views, gatewayResp.MCPToolView{MCPTool: tools[i]})
	}
	return views, nil
}

// UpdateMCPToolBilling 工具级计费覆盖(空 billingType/nil cost=清为继承服务器)并重推 mcp_info。
func (s *McpService) UpdateMCPToolBilling(ctx context.Context, toolId int64, billingType string, cost *float64) (gatewayResp.MCPToolView, error) {
	var tool gateway.MCPTool
	if err := global.OPS_DB.WithContext(ctx).Where("mcp_tool_id = ?", toolId).First(&tool).Error; err != nil {
		return gatewayResp.MCPToolView{}, errors.New("MCP 工具不存在")
	}
	if billingType != "" && billingType != gateway.MCPBillingPerCall && billingType != gateway.MCPBillingFree {
		return gatewayResp.MCPToolView{}, errors.New("计费类型非法(per_call/free)")
	}
	tool.BillingType = billingType
	tool.ExternalCostPerCall = cost
	if err := global.OPS_DB.WithContext(ctx).Model(&gateway.MCPTool{}).Where("mcp_tool_id = ?", toolId).
		Updates(map[string]any{"billing_type": billingType, "external_cost_per_call": cost}).Error; err != nil {
		return gatewayResp.MCPToolView{}, err
	}
	// 重推所属服务器 mcp_info(工具计费参与投影)
	var row gateway.MCPServer
	if err := global.OPS_DB.WithContext(ctx).Where("mcp_server_id = ?", tool.McpServerId).First(&row).Error; err == nil {
		if cli := litellm.Default(); cli != nil && row.LitellmServerId != "" {
			credentials, err := s.decryptCredentials(row.Credentials)
			if err == nil {
				tools, _ := s.mcpTools(ctx, row.McpServerId)
				body := buildMCPLitellmBody(&row, credentials, s.toolCostMapFromRows(tools), true)
				if err := cli.UpdateMCPServer(ctx, body); err != nil {
					logger.WithCtx(ctx).Mod("gateway").Warn(fmt.Sprintf("MCP %s 工具计费投影重推失败: %v", row.ServerName, err))
				}
			}
		}
	}
	return gatewayResp.MCPToolView{MCPTool: tool}, nil
}

// HealthCheckMCPServer 健康检查：经 LiteLLM 服务端代理探测(平台不直连 MCP 端点)，
// 结果落库(healthy/unhealthy+错误信息+时间)。
func (s *McpService) HealthCheckMCPServer(ctx context.Context, id int64) (gatewayResp.MCPServerView, error) {
	var row gateway.MCPServer
	if err := global.OPS_DB.WithContext(ctx).Where("mcp_server_id = ?", id).First(&row).Error; err != nil {
		return gatewayResp.MCPServerView{}, errors.New("MCP 服务器不存在")
	}
	cli := litellm.Default()
	status := gateway.MCPHealthUnhealthy
	var checkErr string
	if cli == nil {
		checkErr = "LiteLLM 未启用，无法探测"
	} else {
		credentials, err := s.decryptCredentials(row.Credentials)
		if err != nil {
			checkErr = err.Error()
		} else if message, err := cli.TestMCPConnection(ctx, row.Url, litellmMCPTransport(row.Transport), row.AuthType, credentials); err != nil {
			checkErr = err.Error()
		} else if strings.HasPrefix(message, "success") {
			status = gateway.MCPHealthHealthy
		} else {
			checkErr = message
		}
	}
	now := time.Now()
	updates := map[string]any{
		"health_status":      status,
		"last_health_check":  now,
		"health_check_error": checkErr,
	}
	if err := global.OPS_DB.WithContext(ctx).Model(&gateway.MCPServer{}).
		Where("mcp_server_id = ?", id).Updates(updates).Error; err != nil {
		return gatewayResp.MCPServerView{}, err
	}
	row.HealthStatus = status
	row.LastHealthCheck = &now
	row.HealthCheckError = checkErr
	return s.toMCPServerView(ctx, &row), nil
}

// ----------------------------------------------------------------------------
// 用户侧接口(广场/授权下拉)
// ----------------------------------------------------------------------------

// GetAvailableMcps 可授权 MCP(管理员授权下拉用)：全部启用服务器，不按可见性过滤
// (与 GetAvailableModels 同语义)。
func (s *McpService) GetAvailableMcps(ctx context.Context) ([]gatewayResp.AvailableMcpView, error) {
	return s.listMcpsAsAvailable(ctx, func(db *gorm.DB) *gorm.DB { return db })
}

// GetActiveMcps 用户侧可见 MCP(按发布可见性三档过滤)：广场/接入页数据源，
// 与 GetActiveModels 同口径恒按申请人对申请人可见性过滤(管理端全量走 GetAvailableMcps)。
func (s *McpService) GetActiveMcps(ctx context.Context, userId int64) ([]gatewayResp.AvailableMcpView, error) {
	db := global.OPS_DB.WithContext(ctx)
	return s.listMcpsAsAvailable(ctx, func(q *gorm.DB) *gorm.DB {
		return visibleMcpScope(q, userId, userDeptIdOf(db, userId))
	})
}

// listMcpsAsAvailable 启用已发布 MCP → AvailableMcpView 贫字段列表(scope 注入过滤条件)。
func (s *McpService) listMcpsAsAvailable(ctx context.Context, scope func(*gorm.DB) *gorm.DB) ([]gatewayResp.AvailableMcpView, error) {
	var rows []gateway.MCPServer
	q := global.OPS_DB.WithContext(ctx).Model(&gateway.MCPServer{}).
		Where("is_active = ? AND is_published = ?", true, true)
	if err := scope(q).Order("mcp_server_id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]gatewayResp.AvailableMcpView, 0, len(rows))
	for i := range rows {
		list = append(list, gatewayResp.AvailableMcpView{
			McpServerId:      rows[i].McpServerId,
			ServerName:       rows[i].ServerName,
			Name:             rows[i].Name,
			Description:      rows[i].Description,
			Category:         rows[i].Category,
			Author:           rows[i].Author,
			IconUrl:          rows[i].IconUrl,
			DocumentationUrl: rows[i].DocumentationUrl,
			RequiresApproval: rows[i].RequiresApproval,
		})
	}
	s.fillToolCounts(ctx, rows, nil)
	for i := range list {
		for j := range rows {
			if rows[j].McpServerId == list[i].McpServerId {
				list[i].ToolCount = rows[j].ToolCount
			}
		}
	}
	return list, nil
}

// GetMCPConnectConfig 用户接入配置：mcpServers JSON(主 Key 明文作 Bearer)+工具清单+说明。
// 未开通主 Key 返回 keyValue 空(前端提示先开通)，不阻塞配置预览。
func (s *McpService) GetMCPConnectConfig(ctx context.Context, id int64, userId int64) (gatewayResp.MCPConnectConfigView, error) {
	var row gateway.MCPServer
	if err := global.OPS_DB.WithContext(ctx).Where("mcp_server_id = ?", id).First(&row).Error; err != nil {
		return gatewayResp.MCPConnectConfigView{}, errors.New("MCP 服务器不存在")
	}
	tools, _ := s.mcpTools(ctx, id)
	briefs := make([]gatewayResp.MCPToolBrief, 0, len(tools))
	for i := range tools {
		name := tools[i].DisplayName
		if name == "" {
			name = tools[i].ToolName
		}
		briefs = append(briefs, gatewayResp.MCPToolBrief{Name: name, Description: tools[i].Description})
	}

	mcpUrl := strings.TrimRight(global.OPS_CONFIG.Litellm.PublicURL, "/") + "/" + row.ServerName + "/mcp"
	view := gatewayResp.MCPConnectConfigView{
		Name:             row.Name,
		ServerName:       row.ServerName,
		McpUrl:           mcpUrl,
		Description:      row.Description,
		Instructions:     row.Instructions,
		DocumentationUrl: row.DocumentationUrl,
		Tools:            briefs,
	}
	if k, err := loadMainKey(ctx, userId); err == nil && k != nil && k.KeyValue != "" {
		if values, err := decryptCredentialValues(k.KeyValue); err == nil {
			if plain, _ := values["k"].(string); plain != "" {
				view.Config = map[string]any{
					"mcpServers": map[string]any{
						row.ServerName: map[string]any{
							"url":     mcpUrl,
							"name":    row.Name,
							"headers": map[string]any{"x-litellm-api-key": "Bearer " + plain},
						},
					},
				}
			}
		}
	}
	return view, nil
}

// ----------------------------------------------------------------------------
// 内部工具
// ----------------------------------------------------------------------------

// validateOperate 操作参数校验与归一：名称/路由名/URL 必填，路由名合法+唯一(排除自身)，
// 传输/鉴权/计费归一，凭据加密。返回待落库行与解密后凭据明文(投影用)。
func (s *McpService) validateOperate(ctx context.Context, req gatewayReq.MCPServerOperateParams, excludeId int64) (*gateway.MCPServer, map[string]any, error) {
	if req.Name == "" || req.ServerName == "" || req.Url == "" {
		return nil, nil, errors.New("名称/路由名/端点URL不能为空")
	}
	if !ValidMCPServerName(req.ServerName) {
		return nil, nil, errors.New("路由名仅允许字母/数字/下划线(LiteLLM 路由限制)")
	}
	var cnt int64
	q := global.OPS_DB.WithContext(ctx).Model(&gateway.MCPServer{}).Where("server_name = ?", req.ServerName)
	if excludeId > 0 {
		q = q.Where("mcp_server_id <> ?", excludeId)
	}
	if err := q.Count(&cnt).Error; err != nil {
		return nil, nil, err
	}
	if cnt > 0 {
		return nil, nil, errors.New("该路由名已存在")
	}
	transport, err := NormalizeMCPTransport(req.Transport)
	if err != nil {
		return nil, nil, err
	}
	authType := req.AuthType
	if authType == "" {
		authType = gateway.MCPAuthNone
	}
	if !ValidMCPAuthType(authType) {
		return nil, nil, errors.New("鉴权方式仅支持 none/api_key/bearer_token")
	}
	if authType != gateway.MCPAuthNone && len(req.Credentials) == 0 && excludeId == 0 {
		return nil, nil, errors.New("启用鉴权时必须提供凭据")
	}
	billing := req.BillingType
	if billing == "" {
		billing = gateway.MCPBillingFree
	}
	if billing != gateway.MCPBillingPerCall && billing != gateway.MCPBillingFree {
		return nil, nil, errors.New("计费类型非法(per_call/free)")
	}
	if billing == gateway.MCPBillingPerCall && req.ExternalCostPerCall == nil {
		return nil, nil, errors.New("按次计费必须填写单次调用价格")
	}

	row := &gateway.MCPServer{
		McpServerId:         req.McpServerId,
		Name:                req.Name,
		ServerName:          req.ServerName,
		Url:                 req.Url,
		Transport:           transport,
		AuthType:            authType,
		Description:         req.Description,
		Instructions:        req.Instructions,
		Category:            req.Category,
		Author:              req.Author,
		IconUrl:             req.IconUrl,
		DocumentationUrl:    req.DocumentationUrl,
		BillingType:         billing,
		ExternalCostPerCall: req.ExternalCostPerCall,
		IsActive:            req.IsActive == nil || *req.IsActive,
		HealthStatus:        gateway.MCPHealthUnknown,
	}
	if row.Category == "" {
		row.Category = "general"
	}
	return row, req.Credentials, nil
}

// toMCPServerView 出网视图：解密凭据掩码(空凭据给 nil)。
func (s *McpService) toMCPServerView(ctx context.Context, row *gateway.MCPServer) gatewayResp.MCPServerView {
	view := gatewayResp.MCPServerView{MCPServer: *row}
	if credentials, err := s.decryptCredentials(row.Credentials); err == nil && len(credentials) > 0 {
		view.Credentials = MaskCredentialValues(credentials)
	}
	return view
}

// decryptCredentials 密文列 → 明文 map(空串给 nil map)。
func (s *McpService) decryptCredentials(enc string) (map[string]any, error) {
	if enc == "" {
		return nil, nil
	}
	values, err := decryptCredentialValues(enc)
	if err != nil {
		return nil, fmt.Errorf("凭据解密失败: %w", err)
	}
	return values, nil
}

// toolCostMap 服务器现有工具级定价 map(toolName→cost，仅收录有配置的)。
func (s *McpService) toolCostMap(ctx context.Context, mcpServerId int64) map[string]*float64 {
	tools, _ := s.mcpTools(ctx, mcpServerId)
	return s.toolCostMapFromRows(tools)
}

func (s *McpService) toolCostMapFromRows(tools []gateway.MCPTool) map[string]*float64 {
	out := map[string]*float64{}
	for i := range tools {
		if tools[i].ExternalCostPerCall != nil {
			out[tools[i].ToolName] = tools[i].ExternalCostPerCall
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// mcpTools 服务器工具行。
func (s *McpService) mcpTools(ctx context.Context, mcpServerId int64) ([]gateway.MCPTool, error) {
	var tools []gateway.MCPTool
	err := global.OPS_DB.WithContext(ctx).Where("mcp_server_id = ?", mcpServerId).Find(&tools).Error
	return tools, err
}

// fillToolCounts 批量填充工具数(一次 GROUP BY，防逐行 N+1)；list 为 nil 时只回填 rows。
func (s *McpService) fillToolCounts(ctx context.Context, rows []gateway.MCPServer, list []gatewayResp.MCPServerView) {
	if len(rows) == 0 {
		return
	}
	ids := make([]int64, 0, len(rows))
	for i := range rows {
		ids = append(ids, rows[i].McpServerId)
	}
	type countRow struct {
		McpServerId int64
		Cnt         int64
	}
	var counts []countRow
	if err := global.OPS_DB.WithContext(ctx).Model(&gateway.MCPTool{}).
		Select("mcp_server_id, COUNT(*) AS cnt").
		Where("mcp_server_id IN ?", ids).
		Group("mcp_server_id").Scan(&counts).Error; err != nil {
		return // 统计失败不影响主流程
	}
	countMap := map[int64]int64{}
	for _, c := range counts {
		countMap[c.McpServerId] = c.Cnt
	}
	for i := range rows {
		rows[i].ToolCount = countMap[rows[i].McpServerId]
		if list != nil {
			for j := range list {
				if list[j].McpServerId == rows[i].McpServerId {
					list[j].ToolCount = rows[i].ToolCount
				}
			}
		}
	}
}

// buildMCPLitellmBody 构建发往 LiteLLM 的 MCP server 投影体(投影原则：派生值只在此产生)。
// allow_all_keys 恒 false——平台侧 Key.allowed_mcp_servers 是唯一授权凭证(AIHelms 坑规避)；
// update 时 server_id 必带；create 时用自定义 server_id=gw_mcp_{雪花} 作归因锚点；
// 无鉴权时显式下发 credentials:null 清残留；mcp_info 为 nil 时下发 null 清空计费投影。
func buildMCPLitellmBody(row *gateway.MCPServer, credentials map[string]any, toolCosts map[string]*float64, update bool) map[string]any {
	body := map[string]any{
		"server_name":    row.ServerName,
		"url":            row.Url,
		"transport":      litellmMCPTransport(row.Transport),
		"allow_all_keys": false,
		"auth_type":      row.AuthType,
		"credentials":    nil, // 无鉴权/无凭据 → null 清残留
	}
	if row.AuthType != gateway.MCPAuthNone && len(credentials) > 0 {
		body["credentials"] = credentials
	}
	if row.Description != "" {
		body["description"] = row.Description
	}
	if row.Instructions != "" {
		body["instructions"] = row.Instructions
	}
	info := MCPCostInfo(row.BillingType, row.ExternalCostPerCall, toolCosts, row.Description, global.OPS_CONFIG.Litellm.UsdToCnyRate)
	body["mcp_info"] = info
	if update {
		body["server_id"] = row.LitellmServerId
	} else {
		body["server_id"] = fmt.Sprintf("gw_mcp_%d", row.McpServerId)
	}
	return body
}
