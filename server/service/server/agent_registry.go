package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hllkk/devops-admin/server/global"
	servermod "github.com/hllkk/devops-admin/server/model/server"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// AgentRegistryService agent 注册与心跳（agent 主动外连，agent 侧零监听端口）。
//
// 鉴权模型：安装时经 Redis 下发一次性注册 token（TTL 30min，关联 assetId）→
// agent 启动后携 token 调 /server/agent/register 换持久 token（明文不下发回显、
// 服务端只存 sha256 hex 哈希列，心跳带明文比对——对齐 AiKey key_hash 先例）。
// Agent 端点挂 PublicGroup（无 JWT/casbin，token 自鉴权）。

const (
	// registerTokenTTL 一次性注册 token 有效期（安装后 agent 首次启动需在此窗口内完成注册）
	registerTokenTTL = 30 * time.Minute
	// registerTokenKey Redis key 前缀（value = assetId JSON）
	registerTokenPrefix = "server:agent-register:"
	// agentTokenBytes 持久令牌随机字节数（hex 后 64 字符，高熵）
	agentTokenBytes = 32
)

// registerTokenValue Redis 存储的注册上下文。
type registerTokenValue struct {
	AssetId int64 `json:"assetId"`
}

// AgentRegistryService agent 注册/心跳/失联判定（agent 主动外连架构的身份与状态层）。
type AgentRegistryService struct{}

// agentRegisterRequest agent 注册请求（PublicGroup，注册 token 鉴权）。
type AgentRegisterRequest struct {
	Token        string `json:"token" form:"token"`               // 一次性注册 token（安装时写入 systemd unit）
	Hostname     string `json:"hostname" form:"hostname"`         // 目标机主机名
	Arch         string `json:"arch" form:"arch"`                 // 目标机架构(amd64/arm64)
	Os           string `json:"os" form:"os"`                     // 目标机系统(linux)
	AgentVersion string `json:"agentVersion" form:"agentVersion"` // agent 版本
}

// agentRegisterResponse 注册成功响应（持久 token 只此一次明文下发）。
type agentRegisterResponse struct {
	AgentId           int64  `json:"agentId,string"`    // 资产ID（agent 身份锚点=资产行）
	Token             string `json:"token"`             // 持久令牌（心跳鉴权，仅注册时下发一次）
	HeartbeatInterval int    `json:"heartbeatInterval"` // 心跳间隔(秒,配置下发)
}

// agentHeartbeatRequest 心跳请求（Authorization: Bearer <token> 自鉴权）。
type AgentHeartbeatRequest struct {
	AgentVersion string        `json:"agentVersion,omitempty" form:"agentVersion"` // agent 版本(可选,有则回写)
	Metrics      *AgentMetrics `json:"metrics,omitempty" form:"metrics"`           // 轻量指标快照(可选,新版 agent 上报)
}

// CreateRegisterToken 生成一次性注册 token（安装流调用）。
func (s *AgentRegistryService) CreateRegisterToken(ctx context.Context, assetId int64) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("生成注册 token 失败: %w", err)
	}
	token := hex.EncodeToString(raw)
	val, err := json.Marshal(registerTokenValue{AssetId: assetId})
	if err != nil {
		return "", err
	}
	if err := global.OPS_REDIS.Set(ctx, registerTokenPrefix+token, val, registerTokenTTL).Err(); err != nil {
		return "", fmt.Errorf("注册 token 写入 Redis 失败: %w", err)
	}
	return token, nil
}

// Register agent 注册：一次性 token 换持久 token（token 只能用一次）。
// 幂等语义：重复注册（agent 重装复用旧 unit 里的注册 token）→ Redis 已删则拒绝，
// agent 需重新触发安装获取新 token。
func (s *AgentRegistryService) Register(ctx context.Context, req AgentRegisterRequest) (agentRegisterResponse, error) {
	if req.Token == "" {
		return agentRegisterResponse{}, errors.New("注册 token 不能为空")
	}
	key := registerTokenPrefix + req.Token
	raw, err := global.OPS_REDIS.Get(ctx, key).Result()
	if err != nil {
		return agentRegisterResponse{}, errors.New("注册 token 无效或已过期(请重新安装 agent)")
	}
	// 立即删除（一次性）
	global.OPS_REDIS.Del(ctx, key)

	var tv registerTokenValue
	if err := json.Unmarshal([]byte(raw), &tv); err != nil {
		return agentRegisterResponse{}, fmt.Errorf("注册 token 解析失败: %w", err)
	}
	var asset servermod.Asset
	if err := global.OPS_DB.WithContext(ctx).Where("asset_id = ?", tv.AssetId).First(&asset).Error; err != nil {
		return agentRegisterResponse{}, errors.New("关联资产不存在")
	}
	// 生成持久 token（明文只此一次下发，库内存 sha256 hex）
	tk := make([]byte, agentTokenBytes)
	if _, err := rand.Read(tk); err != nil {
		return agentRegisterResponse{}, fmt.Errorf("生成持久 token 失败: %w", err)
	}
	token := hex.EncodeToString(tk)
	tokenHash := sha256Hex(token)
	if err := global.OPS_DB.WithContext(ctx).Model(&servermod.Asset{}).
		Where("asset_id = ?", asset.AssetId).
		Updates(map[string]any{
			"agent_token_hash":  tokenHash,
			"agent_version":     req.AgentVersion,
			"agent_hostname":    req.Hostname,
			"agent_status":      servermod.AgentStatusRunning,
			"last_heartbeat_at": time.Now().UTC(),
		}).Error; err != nil {
		return agentRegisterResponse{}, err
	}
	interval := s.heartbeatInterval()
	return agentRegisterResponse{
		AgentId:           asset.AssetId,
		Token:             token,
		HeartbeatInterval: interval,
	}, nil
}

// Heartbeat agent 心跳：持久 token 鉴权 → 回写 running + 时间戳。
func (s *AgentRegistryService) Heartbeat(ctx context.Context, token string, req AgentHeartbeatRequest) error {
	if token == "" {
		return errors.New("token 不能为空")
	}
	updates := map[string]any{
		"agent_status":      servermod.AgentStatusRunning,
		"last_heartbeat_at": time.Now().UTC(),
	}
	if req.AgentVersion != "" {
		updates["agent_version"] = req.AgentVersion
	}
	var assetId int64
	if err := global.OPS_DB.WithContext(ctx).Model(&servermod.Asset{}).
		Select("asset_id").
		Where("agent_token_hash = ?", sha256Hex(token)).Scan(&assetId).Error; err != nil {
		return err
	}
	if assetId == 0 {
		return errors.New("token 无效")
	}
	res := global.OPS_DB.WithContext(ctx).Model(&servermod.Asset{}).
		Where("asset_id = ?", assetId).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if req.Metrics != nil {
		s.saveSnapshot(ctx, assetId, *req.Metrics)
	}
	return nil
}

// CheckLostAgents 失联扫描（定时任务）：超过阈值未心跳的 running 资产置 lost。
// 仅扫启用中资产（停用资产采集停，状态不参与失联判定）。
func (s *AgentRegistryService) CheckLostAgents(ctx context.Context) error {
	threshold := s.lostThreshold()
	cutoff := time.Now().UTC().Add(-time.Duration(threshold) * time.Second)
	res := global.OPS_DB.WithContext(ctx).Model(&servermod.Asset{}).
		Where("agent_status = ? AND is_active = ?", servermod.AgentStatusRunning, true).
		Where("last_heartbeat_at < ? OR last_heartbeat_at IS NULL", cutoff).
		Update("agent_status", servermod.AgentStatusLost)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected > 0 {
		logger.WithCtx(ctx).Mod("server").Field("count", res.RowsAffected).Info("agent 失联标记")
	}
	return nil
}

// heartbeatInterval 心跳间隔(秒)，带默认兜底。
func (s *AgentRegistryService) heartbeatInterval() int {
	if v := global.OPS_CONFIG.ServerModule.Agent.HeartbeatInterval; v > 0 {
		return v
	}
	return 30
}

// lostThreshold 失联阈值(秒)，带默认兜底。
func (s *AgentRegistryService) lostThreshold() int {
	if v := global.OPS_CONFIG.ServerModule.Agent.LostThreshold; v > 0 {
		return v
	}
	return 90
}

// sha256Hex 明文 → sha256 hex（agent token 哈希存储口径，对齐 AiKey key_hash）。
func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// ----------------------------------------------------------------------------
// 轻量指标快照(slice3 正式管道前的查看抽屉数据)：agent 心跳顺带上报一帧，
// 服务端落 Redis(TTL 随心跳间隔,心跳停则过期——「暂无数据」),不进 PG。
// 历史趋势/告警评估留给 slice3 的完整管道(Redis 热窗口+PG 降采样)。
// ----------------------------------------------------------------------------

// agentSnapshotKey 快照 Redis key。
func agentSnapshotKey(assetId int64) string {
	return fmt.Sprintf("server:asset-snapshot:%d", assetId)
}

// AgentMetrics agent 上报的轻量快照(心跳请求体可选字段;旧版 agent 不带)。
type AgentMetrics struct {
	CpuPercent     float64 `json:"cpuPercent"`     // CPU使用率%(最近心跳周期差值)
	Loadavg1       float64 `json:"loadavg1"`       // 1分钟负载
	MemTotalKB     uint64  `json:"memTotalKb"`     // 内存总量KB
	MemAvailableKB uint64  `json:"memAvailableKb"` // 内存可用KB
	DiskTotalBytes uint64  `json:"diskTotalBytes"` // 根分区总量
	DiskUsedBytes  uint64  `json:"diskUsedBytes"`  // 根分区已用
	NetInBps       float64 `json:"netInBps"`       // 网络入速率 B/s(心跳周期差值)
	NetOutBps      float64 `json:"netOutBps"`      // 网络出速率 B/s
	UptimeSec      uint64  `json:"uptimeSec"`      // 系统运行时长(秒)
}

// AgentSnapshot 快照出网视图(带资产口径派生值)。
type AgentSnapshot struct {
	AgentMetrics
	ReportedAt time.Time `json:"reportedAt"` // 上报时间(UTC)
	Stale      bool      `json:"stale"`      // 快照过期标记(取不到时 true,数据为零值)
}

// saveSnapshot 心跳时写快照(TTL=2×心跳间隔+60s 兜底;at 时间键同 TTL)。
func (s *AgentRegistryService) saveSnapshot(ctx context.Context, assetId int64, m AgentMetrics) {
	raw, err := json.Marshal(m)
	if err != nil {
		return
	}
	ttl := time.Duration(2*s.heartbeatInterval()+60) * time.Second
	global.OPS_REDIS.Set(ctx, agentSnapshotKey(assetId), raw, ttl)
	global.OPS_REDIS.Set(ctx, agentSnapshotKey(assetId)+":at", time.Now().UTC().Format(time.RFC3339), ttl)
}

// GetSnapshot 资产查看接口读快照;无数据返回 Stale=true(前端显示「暂无数据/agent 未上报」)。
func (s *AgentRegistryService) GetSnapshot(ctx context.Context, assetId int64) AgentSnapshot {
	snap := AgentSnapshot{Stale: true}
	raw, err := global.OPS_REDIS.Get(ctx, agentSnapshotKey(assetId)).Result()
	if err != nil {
		return snap
	}
	if err := json.Unmarshal([]byte(raw), &snap.AgentMetrics); err != nil {
		return snap
	}
	snap.Stale = false
	// 记录写入时间未知,用 TTL 反推不精确;另存 reportedAt(写快照时单独存一版时间)
	if at, err := global.OPS_REDIS.Get(ctx, agentSnapshotKey(assetId)+":at").Result(); err == nil {
		if t, err := time.Parse(time.RFC3339, at); err == nil {
			snap.ReportedAt = t
		}
	}
	return snap
}
