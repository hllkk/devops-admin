package server

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/server"
	serverReq "github.com/hllkk/devops-admin/server/model/server/request"
	serverResp "github.com/hllkk/devops-admin/server/model/server/response"
)

// AssetService 服务器模块统一资产管理(物理机/虚机/docker主机/db实例/网络设备)。
// monitor_status/agent_status 由采集链路回写，管理 CRUD 不触碰；
// SSH 凭据逻辑关联 server_credential(不建外键，service 层保证)。
type AssetService struct{}

// GetAssetList 分页查资产列表(对齐前端 GET /server/asset/list)。
// assetName/manageIp 模糊；assetType/env/monitorStatus/isActive 精确(指针区分未传与 false)。
func (s *AssetService) GetAssetList(ctx context.Context, q serverReq.AssetSearch) (list []server.Asset, total int64, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&server.Asset{})
	if q.AssetName != "" {
		db = db.Where("asset_name LIKE ?", "%"+q.AssetName+"%")
	}
	if q.ManageIp != "" {
		db = db.Where("manage_ip LIKE ?", "%"+q.ManageIp+"%")
	}
	if q.AssetType != "" {
		db = db.Where("asset_type = ?", q.AssetType)
	}
	if q.Env != "" {
		db = db.Where("env = ?", q.Env)
	}
	if q.MonitorStatus != "" {
		db = db.Where("monitor_status = ?", q.MonitorStatus)
	}
	if q.IsActive != nil {
		db = db.Where("is_active = ?", *q.IsActive)
	}
	var rows []server.Asset
	limit, offset := q.LimitOffset()
	if limit > 0 {
		err = db.Count(&total).Order("asset_id DESC").Limit(limit).Offset(offset).Find(&rows).Error
	} else {
		err = db.Count(&total).Order("asset_id DESC").Find(&rows).Error
	}
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// GetAsset 查资产详情(对齐前端 GET /server/asset/:id)。
func (s *AssetService) GetAsset(ctx context.Context, id int64) (server.Asset, error) {
	var a server.Asset
	if err := global.OPS_DB.WithContext(ctx).Where("asset_id = ?", id).First(&a).Error; err != nil {
		return server.Asset{}, err
	}
	return a, nil
}

// CreateAsset 新增资产；createBy 填审计字段。monitor/agent 状态走默认值(unknown/none)。
func (s *AssetService) CreateAsset(ctx context.Context, req serverReq.AssetOperateParams, createBy int64) (server.Asset, error) {
	if req.AssetName == "" {
		return server.Asset{}, errors.New("资产名称不能为空")
	}
	if !ValidateAssetType(req.AssetType) {
		return server.Asset{}, fmt.Errorf("资产类型无效: %q(physical/vm/docker_host/db_instance/net_device)", req.AssetType)
	}
	if err := s.ensureCredentialExists(ctx, req.CredentialId); err != nil {
		return server.Asset{}, err
	}
	// 未删行查重(软删不建唯一索引是项目成文决策，靠服务层保证)
	var dup int64
	if err := global.OPS_DB.WithContext(ctx).Model(&server.Asset{}).
		Where("asset_name = ?", req.AssetName).Count(&dup).Error; err != nil {
		return server.Asset{}, err
	}
	if dup > 0 {
		return server.Asset{}, fmt.Errorf("资产名 %q 已存在", req.AssetName)
	}
	if req.SshPort == 0 {
		req.SshPort = 22
	}
	a := server.Asset{
		AssetName:     req.AssetName,
		AssetType:     req.AssetType,
		ManageIp:      req.ManageIp,
		SshPort:       req.SshPort,
		OsType:        req.OsType,
		Env:           req.Env,
		Location:      req.Location,
		IsActive:      req.IsActive == nil || *req.IsActive,
		CredentialId:  req.CredentialId,
		ChannelConfig: req.ChannelConfig,
		Description:   req.Description,
	}
	a.CreateBy = createBy
	a.UpdateBy = createBy
	if err := global.OPS_DB.WithContext(ctx).Create(&a).Error; err != nil {
		return server.Asset{}, err
	}
	return a, nil
}

// UpdateAsset 修改资产；assetId 必填。monitor_status/agent_status 为采集回写域，不在此接收。
func (s *AssetService) UpdateAsset(ctx context.Context, req serverReq.AssetOperateParams, updateBy int64) error {
	if req.AssetId == 0 {
		return errors.New("资产ID不能为空")
	}
	if req.AssetType != "" && !ValidateAssetType(req.AssetType) {
		return fmt.Errorf("资产类型无效: %q(physical/vm/docker_host/db_instance/net_device)", req.AssetType)
	}
	if err := s.ensureCredentialExists(ctx, req.CredentialId); err != nil {
		return err
	}
	var old server.Asset
	if err := global.OPS_DB.WithContext(ctx).Where("asset_id = ?", req.AssetId).First(&old).Error; err != nil {
		return err
	}
	// 同名查重(改名与其他未软删行冲突)
	if req.AssetName != "" && req.AssetName != old.AssetName {
		var dup int64
		if err := global.OPS_DB.WithContext(ctx).Model(&server.Asset{}).
			Where("asset_name = ? AND asset_id <> ?", req.AssetName, req.AssetId).
			Count(&dup).Error; err != nil {
			return err
		}
		if dup > 0 {
			return fmt.Errorf("资产名 %q 已存在", req.AssetName)
		}
	}
	updates := map[string]any{
		"update_by": updateBy,
	}
	if req.AssetName != "" {
		updates["asset_name"] = req.AssetName
	}
	if req.AssetType != "" {
		updates["asset_type"] = req.AssetType
	}
	updates["manage_ip"] = req.ManageIp
	if req.SshPort != 0 {
		updates["ssh_port"] = req.SshPort
	}
	updates["os_type"] = req.OsType
	updates["env"] = req.Env
	updates["location"] = req.Location
	updates["credential_id"] = req.CredentialId
	updates["channel_config"] = req.ChannelConfig
	updates["description"] = req.Description
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	return global.OPS_DB.WithContext(ctx).Model(&server.Asset{}).
		Where("asset_id = ?", req.AssetId).Updates(updates).Error
}

// DeleteAsset 批量删除资产(软删除)。
// 后续 slice 的采集任务按资产行驱动，删资产即自然停采(任务下轮不再命中)。
func (s *AssetService) DeleteAsset(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return errors.New("未选择删除项")
	}
	return global.OPS_DB.WithContext(ctx).Where("asset_id IN ?", ids).Delete(&server.Asset{}).Error
}

// ensureCredentialExists 校验关联凭据存在且启用(纯逻辑关联不建外键，service 层保证)。
func (s *AssetService) ensureCredentialExists(ctx context.Context, credentialId int64) error {
	if credentialId == 0 {
		return nil // 0=未关联，允许
	}
	var c server.Credential
	if err := global.OPS_DB.WithContext(ctx).Select("credential_id", "is_active").
		Where("credential_id = ?", credentialId).First(&c).Error; err != nil {
		return errors.New("关联凭据不存在")
	}
	if !c.IsActive {
		return errors.New("关联凭据已停用")
	}
	return nil
}

// GetAssetOverview 服务器概览统计(模块顶层概览页)。
// ByType/ByEnv 按未软删全量聚合；MonitorStatus/AgentStatus 只统计启用中
// (停用资产不采集,状态恒 unknown 无统计意义)。
func (s *AssetService) GetAssetOverview(ctx context.Context) (serverResp.AssetOverview, error) {
	overview := serverResp.AssetOverview{
		ByType:        map[string]int64{},
		MonitorStatus: map[string]int64{},
		AgentStatus:   map[string]int64{},
		ByEnv:         map[string]int64{},
	}
	db := global.OPS_DB.WithContext(ctx).Model(&server.Asset{})
	// 总数 + 启用数
	if err := db.Count(&overview.Total).Error; err != nil {
		return overview, err
	}
	if err := db.Where("is_active = ?", true).Count(&overview.ActiveTotal).Error; err != nil {
		return overview, err
	}
	// 各类型数量(未软删全量)
	var byType []struct {
		AssetType string
		Cnt       int64
	}
	if err := db.Select("asset_type, COUNT(*) AS cnt").Group("asset_type").Find(&byType).Error; err != nil {
		return overview, err
	}
	for _, r := range byType {
		overview.ByType[r.AssetType] = r.Cnt
	}
	// 环境分布(空 env 归 unknown;NULL/空串统一)
	var byEnv []struct {
		Env string
		Cnt int64
	}
	if err := db.Select("COALESCE(NULLIF(env, ''), 'unknown') AS env, COUNT(*) AS cnt").
		Group("COALESCE(NULLIF(env, ''), 'unknown')").Find(&byEnv).Error; err != nil {
		return overview, err
	}
	for _, r := range byEnv {
		overview.ByEnv[r.Env] = r.Cnt
	}
	// 监控/Agent 状态(仅启用中)
	active := db.Where("is_active = ?", true)
	var byMonitor []struct {
		Status string
		Cnt    int64
	}
	if err := active.Session(&gorm.Session{}).Select("monitor_status AS status, COUNT(*) AS cnt").
		Group("monitor_status").Find(&byMonitor).Error; err != nil {
		return overview, err
	}
	for _, r := range byMonitor {
		overview.MonitorStatus[r.Status] = r.Cnt
	}
	var byAgent []struct {
		Status string
		Cnt    int64
	}
	if err := active.Session(&gorm.Session{}).Select("agent_status AS status, COUNT(*) AS cnt").
		Group("agent_status").Find(&byAgent).Error; err != nil {
		return overview, err
	}
	for _, r := range byAgent {
		overview.AgentStatus[r.Status] = r.Cnt
	}
	return overview, nil
}
