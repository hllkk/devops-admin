package server

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/hllkk/devops-admin/server/global"
	servermod "github.com/hllkk/devops-admin/server/model/server"
	"github.com/hllkk/devops-admin/server/utils/sshkey"
)

// agent_ops.go agent 运维操作(重启/卸载)，与安装流共用 SSH 公钥通道与
// 异步任务状态框架(Redis server:agent-install:<taskId>，前端同一轮询接口)。
// 入口统一在资产查看抽屉:未装=安装；已装=重启/卸载。

// StartRestart 重启目标机 agent(异步):公钥连接 → systemctl restart → 校验 active。
func (s *AgentInstallService) StartRestart(ctx context.Context, assetId int64) (string, error) {
	asset, taskId, _, err := s.prepareOps(ctx, assetId, "重启 agent")
	if err != nil {
		return "", err
	}
	go s.runRestart(assetId, asset.SshUsername, asset.ManageIp, s.portOf(asset), taskId)
	return taskId, nil
}

// StartUninstall 卸载目标机 agent(异步):stop/disable → 删 unit/二进制/token 缓存
// → 服务端资产行 agent 字段清零(状态回 none,保留重装入口)。公钥本身不回收
// (平台与该机的 SSH 信任保留——重装/SSH 运维仍可用)。
func (s *AgentInstallService) StartUninstall(ctx context.Context, assetId int64) (string, error) {
	asset, taskId, _, err := s.prepareOps(ctx, assetId, "卸载 agent")
	if err != nil {
		return "", err
	}
	go s.runUninstall(assetId, asset.SshUsername, asset.ManageIp, s.portOf(asset), taskId)
	return taskId, nil
}

// prepareOps 运维操作前置校验 + 任务状态初始化(installing 状态不复用——
// 重启/卸载不改 agent_status 语义,任务态独立于资产行)。
func (s *AgentInstallService) prepareOps(ctx context.Context, assetId int64, step string) (servermod.Asset, string, AgentInstallStatus, error) {
	var asset servermod.Asset
	if err := global.OPS_DB.WithContext(ctx).Where("asset_id = ?", assetId).First(&asset).Error; err != nil {
		return asset, "", AgentInstallStatus{}, errors.New("资产不存在")
	}
	if !asset.SshVerified || asset.SshUsername == "" || asset.ManageIp == "" {
		return asset, "", AgentInstallStatus{}, errors.New("资产未完成 SSH 验证(无法执行 agent 运维操作)")
	}
	if asset.AgentStatus == servermod.AgentStatusNone {
		return asset, "", AgentInstallStatus{}, errors.New("该资产未安装 agent")
	}
	taskId := sha256Hex(fmt.Sprintf("ops-%d-%d", assetId, time.Now().UnixNano()))[:16]
	status := AgentInstallStatus{
		TaskId:    taskId,
		AssetId:   assetId,
		Status:    "running",
		Step:      step,
		StartedAt: time.Now().UTC(),
	}
	if err := s.saveStatus(ctx, status); err != nil {
		return asset, "", status, err
	}
	return asset, taskId, status, nil
}

// runRestart 重启流。
func (s *AgentInstallService) runRestart(assetId int64, sshUser, host string, port int, taskId string) {
	ctx := context.Background()
	status := AgentInstallStatus{TaskId: taskId, AssetId: assetId, Status: "running", StartedAt: time.Now().UTC()}
	fail := func(step, msg string) {
		now := time.Now().UTC()
		status.Status, status.Step, status.Message, status.FinishedAt = "failed", step, msg, &now
		_ = s.saveStatus(ctx, status)
	}
	client, err := s.dialPublicKey(sshUser, host, port)
	if err != nil {
		fail("连接目标机(公钥认证)", fmt.Sprintf("%v(公钥可能被移除,请编辑资产重新验证)", err))
		return
	}
	defer client.Close()
	status.Step = "重启 aiops-agent 服务"
	_ = s.saveStatus(ctx, status)
	if out, err := sshRun(client, "systemctl restart aiops-agent 2>&1"); err != nil {
		fail(status.Step, fmt.Sprintf("%v; %s", err, out))
		return
	}
	status.Step = "校验服务状态"
	_ = s.saveStatus(ctx, status)
	out, err := sshRun(client, "systemctl is-active aiops-agent 2>&1")
	if err != nil || strings.TrimSpace(out) != "active" {
		fail(status.Step, fmt.Sprintf("服务未处于 active: %s", strings.TrimSpace(out)))
		return
	}
	now := time.Now().UTC()
	status.Status, status.Message, status.FinishedAt = "success", "重启完成,服务运行中", &now
	_ = s.saveStatus(ctx, status)
}

// runUninstall 卸载流 + 服务端字段清零。
func (s *AgentInstallService) runUninstall(assetId int64, sshUser, host string, port int, taskId string) {
	ctx := context.Background()
	status := AgentInstallStatus{TaskId: taskId, AssetId: assetId, Status: "running", StartedAt: time.Now().UTC()}
	fail := func(step, msg string) {
		now := time.Now().UTC()
		status.Status, status.Step, status.Message, status.FinishedAt = "failed", step, msg, &now
		_ = s.saveStatus(ctx, status)
	}
	client, err := s.dialPublicKey(sshUser, host, port)
	if err != nil {
		fail("连接目标机(公钥认证)", fmt.Sprintf("%v(公钥可能被移除,请编辑资产重新验证)", err))
		return
	}
	defer client.Close()
	status.Step = "停止并禁用服务"
	_ = s.saveStatus(ctx, status)
	if out, err := sshRun(client, "systemctl disable --now aiops-agent 2>&1"); err != nil {
		fail(status.Step, fmt.Sprintf("%v; %s", err, out))
		return
	}
	status.Step = "清理文件(unit/二进制/令牌缓存)"
	_ = s.saveStatus(ctx, status)
	if out, err := sshRun(client, fmt.Sprintf("rm -f %s %s && rm -rf /var/lib/aiops-agent 2>&1", remoteUnitPath, remoteBinaryPath)); err != nil {
		fail(status.Step, fmt.Sprintf("%v; %s", err, out))
		return
	}
	status.Step = "清理平台侧记录"
	_ = s.saveStatus(ctx, status)
	if err := global.OPS_DB.WithContext(ctx).Model(&servermod.Asset{}).
		Where("asset_id = ?", assetId).
		Updates(map[string]any{
			"agent_token_hash":  "",
			"agent_version":     "",
			"agent_hostname":    "",
			"agent_status":      servermod.AgentStatusNone,
			"last_heartbeat_at": nil,
		}).Error; err != nil {
		fail(status.Step, err.Error())
		return
	}
	_ = global.OPS_REDIS.Del(ctx, agentSnapshotKey(assetId)).Err()
	now := time.Now().UTC()
	status.Status, status.Message, status.FinishedAt = "success", "卸载完成(平台公钥保留,可随时重装)", &now
	_ = s.saveStatus(ctx, status)
}

// dialPublicKey 公钥认证连接(运维操作用,超时上限 30s——纯命令操作无需长等)。
func (s *AgentInstallService) dialPublicKey(sshUser, host string, port int) (*ssh.Client, error) {
	signer, err := sshkey.Signer(s.sshKeyDir())
	if err != nil {
		return nil, fmt.Errorf("平台密钥初始化失败: %w", err)
	}
	timeout := 30
	return sshDialPublicKey(host, port, sshUser, signer, timeout)
}

func (s *AgentInstallService) portOf(asset servermod.Asset) int {
	if asset.SshPort > 0 {
		return asset.SshPort
	}
	return 22
}
