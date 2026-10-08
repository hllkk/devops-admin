package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"github.com/hllkk/devops-admin/server/global"
	servermod "github.com/hllkk/devops-admin/server/model/server"
	"github.com/hllkk/devops-admin/server/utils/logger"
	"github.com/hllkk/devops-admin/server/utils/sshkey"
)

// AgentInstallService agent 自动安装流（前端「安装 agent」触发，异步任务 + 状态轮询）。
//
// 全程 agentless 一次性 SSH 密码认证（密码仅函数参数内存态，绝不落库/日志/任务状态）：
// 密码登录 → 平台公钥写入 authorized_keys（幂等，转公钥认证）→ sftp 上传 agent 二进制
// （按目标机架构选托管文件）→ 写 systemd unit（含一次性注册 token）→ enable --now。
// agent 启动后携 token 调 /server/agent/register 换持久 token，注册成功状态转 running。
//
// 公钥部署函数按「可写任意公钥」设计（平台公钥/用户个人公钥同函数，P4 登录授权零返工）。

const (
	// installTaskKeyPrefix 安装任务状态 Redis key 前缀（value = 状态 JSON，TTL 2h）
	installTaskKeyPrefix = "server:agent-install:"
	// installTaskTTL 安装任务状态保留时长（轮询窗口，过期自动清理）
	installTaskTTL = 2 * time.Hour
	// remoteBinaryPath 目标机 agent 二进制路径
	remoteBinaryPath = "/usr/local/bin/aiops-agent"
	// remoteUnitPath 目标机 systemd unit 路径
	remoteUnitPath = "/etc/systemd/system/aiops-agent.service"
)

// AgentInstallService 安装编排服务。
type AgentInstallService struct{}

// registryService 同包注册服务实例（安装流生成注册 token 用）。
var registryService = &AgentRegistryService{}

// AgentInstallStatus 异步安装任务状态（Redis 存储 + 轮询出网）。
type AgentInstallStatus struct {
	TaskId     string     `json:"taskId"`            // 任务ID
	AssetId    int64      `json:"assetId,string"`    // 资产ID
	Status     string     `json:"status"`            // running/success/failed
	Step       string     `json:"step"`              // 当前步骤(部署公钥/上传二进制/注册服务/启动)
	Message    string     `json:"message,omitempty"` // 失败原因/成功摘要
	StartedAt  time.Time  `json:"startedAt"`         // 开始时间(UTC)
	FinishedAt *time.Time `json:"finishedAt"`        // 结束时间(UTC,空=进行中)
}

// StartInstall 启动异步安装（纯公钥模式：SSH 信任已在资产录入验证时建立——
// 平台公钥已部署,安装不再需要密码）：置 installing + 生成注册 token + 后台 goroutine 执行。
// 立即返回任务ID，前端轮询 GetInstallStatus。
func (s *AgentInstallService) StartInstall(ctx context.Context, assetId int64) (string, error) {
	var asset servermod.Asset
	if err := global.OPS_DB.WithContext(ctx).Where("asset_id = ?", assetId).First(&asset).Error; err != nil {
		return "", errors.New("资产不存在")
	}
	if asset.AssetType != servermod.AssetTypePhysical && asset.AssetType != servermod.AssetTypeVm {
		return "", fmt.Errorf("资产类型 %q 不支持安装 agent(仅物理机/虚拟机)", asset.AssetType)
	}
	if !asset.SshVerified {
		return "", errors.New("资产未完成 SSH 验证(请编辑资产输入密码完成录入验证后再安装)")
	}
	if asset.SshUsername == "" || asset.ManageIp == "" {
		return "", errors.New("资产缺少 SSH 用户名或管理IP(请编辑资产补全)")
	}
	port := asset.SshPort
	if port == 0 {
		port = 22
	}
	// 一次性注册 token（30min 窗口，agent 首启换持久 token）
	registerToken, err := registryService.CreateRegisterToken(ctx, assetId)
	if err != nil {
		return "", err
	}
	// 任务状态 + 资产置 installing
	taskId := sha256Hex(fmt.Sprintf("%d-%d", assetId, time.Now().UnixNano()))[:16]
	status := AgentInstallStatus{
		TaskId:    taskId,
		AssetId:   assetId,
		Status:    "running",
		Step:      "连接目标机",
		StartedAt: time.Now().UTC(),
	}
	if err := s.saveStatus(ctx, status); err != nil {
		return "", err
	}
	if err := global.OPS_DB.WithContext(ctx).Model(&servermod.Asset{}).
		Where("asset_id = ?", assetId).
		Update("agent_status", servermod.AgentStatusInstalling).Error; err != nil {
		return "", err
	}

	go s.runInstall(assetId, asset.SshUsername, asset.ManageIp, port, registerToken, taskId)
	return taskId, nil
}

// GetInstallStatus 轮询安装任务状态。
func (s *AgentInstallService) GetInstallStatus(ctx context.Context, taskId string) (AgentInstallStatus, error) {
	var status AgentInstallStatus
	raw, err := global.OPS_REDIS.Get(ctx, installTaskKeyPrefix+taskId).Result()
	if err != nil {
		return AgentInstallStatus{}, errors.New("安装任务不存在或已过期")
	}
	if err := json.Unmarshal([]byte(raw), &status); err != nil {
		return AgentInstallStatus{}, err
	}
	return status, nil
}

// runInstall 安装流主体（goroutine；公钥认证连接，无密码入参）。
func (s *AgentInstallService) runInstall(assetId int64, sshUser, host string, port int, registerToken, taskId string) {
	ctx := context.Background()
	status := AgentInstallStatus{
		TaskId:    taskId,
		AssetId:   assetId,
		Status:    "running",
		StartedAt: time.Now().UTC(),
	}
	fail := func(step, msg string) {
		now := time.Now().UTC()
		status.Status, status.Step, status.Message, status.FinishedAt = "failed", step, msg, &now
		_ = s.saveStatus(ctx, status)
		// 安装失败回 none（保留重试入口）
		global.OPS_DB.Model(&servermod.Asset{}).Where("asset_id = ?", assetId).
			Update("agent_status", servermod.AgentStatusNone)
		logger.WithCtx(ctx).Mod("server").Field("assetId", assetId).Field("taskId", taskId).
			Field("step", step).Error("agent 安装失败: " + msg)
	}
	finish := func(msg string) {
		now := time.Now().UTC()
		status.Status, status.Message, status.FinishedAt = "success", msg, &now
		_ = s.saveStatus(ctx, status)
	}

	// ── 1. SSH 公钥认证连接（信任已在录入验证时建立） ──
	status.Step = "连接目标机(公钥认证)"
	_ = s.saveStatus(ctx, status)
	signer, err := sshkey.Signer(s.sshKeyDir())
	if err != nil {
		fail(status.Step, fmt.Sprintf("平台密钥初始化失败: %v", err))
		return
	}
	client, err := sshDialPublicKey(host, port, sshUser, signer, s.installTimeout())
	if err != nil {
		fail(status.Step, fmt.Sprintf("SSH 公钥连接失败: %v(公钥可能被移除,请编辑资产重新验证)", err))
		return
	}
	defer client.Close()

	// ── 2. 探测架构 + 上传 agent 二进制(sftp) ──
	status.Step = "探测系统架构"
	_ = s.saveStatus(ctx, status)
	arch, err := sshRun(client, "uname -m")
	if err != nil {
		fail(status.Step, fmt.Sprintf("探测架构失败: %v", err))
		return
	}
	goarch := mapUnameToGoarch(strings.TrimSpace(arch))
	status.Step = "上传 agent 二进制(" + goarch + ")"
	_ = s.saveStatus(ctx, status)
	binaryPath, err := s.agentBinaryPath(goarch)
	if err != nil {
		fail(status.Step, err.Error())
		return
	}
	if err := sftpUpload(client, binaryPath, remoteBinaryPath); err != nil {
		fail(status.Step, fmt.Sprintf("上传二进制失败: %v", err))
		return
	}

	// ── 4. 写 systemd unit + 启动 ──
	status.Step = "注册 systemd 服务"
	_ = s.saveStatus(ctx, status)
	unit, err := s.buildUnit(registerToken)
	if err != nil {
		fail(status.Step, err.Error())
		return
	}
	if err := sshRunWriteFile(client, remoteUnitPath, unit); err != nil {
		fail(status.Step, fmt.Sprintf("写 unit 失败: %v", err))
		return
	}
	status.Step = "启动服务"
	_ = s.saveStatus(ctx, status)
	if out, err := sshRun(client, "systemctl daemon-reload && systemctl enable --now aiops-agent 2>&1"); err != nil {
		fail(status.Step, fmt.Sprintf("启动失败: %v; %s", err, out))
		return
	}
	// 状态转 running 由 agent 注册回调驱动；安装流程至此成功（状态仍是 installing，
	// agent 首启注册成功后 AgentRegistryService.Register 置 running）
	finish("安装完成,等待 agent 注册(状态将自动转为运行中)")
	logger.WithCtx(ctx).Mod("server").Field("assetId", assetId).Field("taskId", taskId).Info("agent 安装成功")
}

// saveStatus 任务状态写 Redis（TTL 2h）。
func (s *AgentInstallService) saveStatus(ctx context.Context, status AgentInstallStatus) error {
	raw, err := json.Marshal(status)
	if err != nil {
		return err
	}
	return global.OPS_REDIS.Set(ctx, installTaskKeyPrefix+status.TaskId, raw, installTaskTTL).Err()
}

// buildUnit 生成 systemd unit（环境变量注入 server 地址/注册 token/心跳间隔）。
func (s *AgentInstallService) buildUnit(registerToken string) (string, error) {
	serverURL := agentServerURL()
	if serverURL == "" {
		return "", errors.New("agent 接入地址未配置(agent.server-url)")
	}
	return fmt.Sprintf(`[Unit]
Description=AIOps Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%s
Restart=always
RestartSec=10
Environment=AIOPS_SERVER_URL=%s
Environment=AIOPS_REGISTER_TOKEN=%s

[Install]
WantedBy=multi-user.target
`, remoteBinaryPath, serverURL, registerToken), nil
}

// agentBinaryPath 按目标架构找托管二进制（aiops-agent-<ver>-linux-<arch>）。
func (s *AgentInstallService) agentBinaryPath(goarch string) (string, error) {
	dir := s.binaryDir()
	if dir == "" {
		dir = "resource/agent"
	}
	ver := agentVersion()
	candidates := []string{
		filepath.Join(dir, fmt.Sprintf("aiops-agent-%s-linux-%s", ver, goarch)),
		filepath.Join(dir, fmt.Sprintf("aiops-agent-linux-%s", goarch)), // 无版本号兜底(开发期手放)
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("agent 二进制缺失: %s(先执行 scripts/build-agent.sh 构建托管)", candidates[0])
}

// agentServerURL agent 回连地址（agent 侧需要能解析到达；默认本机:port 推导，
// 跨网段部署显式配置 server.agent.server-url）。
func agentServerURL() string {
	if v := global.OPS_CONFIG.ServerModule.Agent.ServerURL; v != "" {
		return v
	}
	// 未配置：用本机非回环地址 + 后端端口推导（dev 单机 go run 场景够用）
	port := global.OPS_CONFIG.System.Addr
	if ip := outboundIP(); ip != "" && port > 0 {
		return fmt.Sprintf("http://%s%s", net.JoinHostPort(ip, fmt.Sprintf("%d", port)), global.OPS_CONFIG.System.RouterPrefix)
	}
	return ""
}

// outboundIP 本机主要出网地址（UDP connect 技巧，不发包）。
func outboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		return ""
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}

// installTimeout 安装单步超时（秒）。
func (s *AgentInstallService) installTimeout() int {
	if v := global.OPS_CONFIG.ServerModule.Agent.InstallTimeout; v > 0 {
		return v
	}
	return 120
}

func (s *AgentInstallService) binaryDir() string {
	return global.OPS_CONFIG.ServerModule.Agent.BinaryDir
}
func (s *AgentInstallService) sshKeyDir() string {
	return global.OPS_CONFIG.ServerModule.Agent.SSHKeyDir
}

// ----------------------------------------------------------------------------
// SSH 底层工具（密码认证/公钥部署/命令执行/sftp 上传）
// ----------------------------------------------------------------------------

// sshDialPassword SSH 密码认证连接（一次性使用场景；后续平台访问走公钥认证）。
func sshDialPassword(host string, port int, username, password string, timeoutSec int) (*ssh.Client, error) {
	conf := &ssh.ClientConfig{
		User: username,
		// password + keyboard-interactive 双方法(OpenSSH 客户端/paramiko 隐式回退行为):
		// 很多 sshd 禁用 PasswordAuthentication 只留 KbdInteractive(x/crypto 不自动回退,
		// 缺此方法时密码正确也报 unable to authenticate)
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
			ssh.KeyboardInteractive(func(_ string, _ string, questions []string, _ []bool) ([]string, error) {
				// 用同一密码回答全部 challenge(PAM 单问题的标准场景)
				answers := make([]string, len(questions))
				for i := range answers {
					answers[i] = password
				}
				return answers, nil
			}),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 一次性安装流：目标机指纹未预置（跳过校验）
		Timeout:         time.Duration(timeoutSec) * time.Second,
	}
	return ssh.Dial("tcp", net.JoinHostPort(host, fmt.Sprintf("%d", port)), conf)
}

// sshDialPublicKey SSH 公钥认证连接（平台私钥；后续 SSH 访问/重装/用户公钥下发用）。
func sshDialPublicKey(host string, port int, username string, signer ssh.Signer, timeoutSec int) (*ssh.Client, error) {
	conf := &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Duration(timeoutSec) * time.Second,
	}
	return ssh.Dial("tcp", net.JoinHostPort(host, fmt.Sprintf("%d", port)), conf)
}

// appendAuthorizedKey 幂等追加公钥到目标用户 authorized_keys（不存在则建目录文件）。
// 写任意公钥的通用函数：平台公钥/P4 用户公钥同入口。
func appendAuthorizedKey(client *ssh.Client, pubKeyLine string) error {
	pubKeyLine = strings.TrimSpace(pubKeyLine)
	if pubKeyLine == "" {
		return errors.New("公钥内容为空")
	}
	// 读现内容 → 判重 → 追加（幂等）
	out, err := sshRun(client, "cat ~/.ssh/authorized_keys 2>/dev/null || true")
	if err != nil {
		return fmt.Errorf("读取 authorized_keys 失败: %w", err)
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == pubKeyLine {
			return nil // 已部署
		}
	}
	// 写入用 printf '%s\n' + 单引号包裹。不能用 %q——它给整串加双引号且把换行转义成
	// 字面 \n，写出的行非法(sshd 拒绝解析=公钥无效)。公钥行自产(类型+base64+固定注释)，
	// 字符集不含单引号，防御性剔除后单引号包裹即安全。
	line := strings.ReplaceAll(pubKeyLine, "'", "")
	script := fmt.Sprintf(
		"mkdir -p ~/.ssh && chmod 700 ~/.ssh && printf '%%s\\n' '%s' >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys",
		line)
	if _, err := sshRun(client, script); err != nil {
		return fmt.Errorf("追加公钥失败: %w", err)
	}
	return nil
}

// sshRun 执行远程命令，返回合并输出。
func sshRun(client *ssh.Client, cmd string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	var buf strings.Builder
	session.Stdout = &buf
	session.Stderr = &buf
	if err := session.Run(cmd); err != nil {
		return buf.String(), err
	}
	return buf.String(), nil
}

// sshRunWriteFile 远程写文件（内容经 stdin 传入，避免命令行转义）。
func sshRunWriteFile(client *ssh.Client, path, content string) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}
	if err := session.Start(fmt.Sprintf("tee %s >/dev/null", path)); err != nil {
		return err
	}
	if _, err := io.WriteString(stdin, content); err != nil {
		return err
	}
	stdin.Close()
	return session.Wait()
}

// sftpUpload 经 sftp 子系统上传本地文件到远程路径（保留执行权限位）。
func sftpUpload(client *ssh.Client, localPath, remotePath string) error {
	local, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer local.Close()
	sc, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("打开 sftp 通道失败: %w", err)
	}
	defer sc.Close()
	remote, err := sc.Create(remotePath)
	if err != nil {
		return fmt.Errorf("创建远程文件失败: %w", err)
	}
	defer remote.Close()
	if _, err := io.Copy(remote, local); err != nil {
		return fmt.Errorf("传输失败: %w", err)
	}
	if err := sc.Chmod(remotePath, 0o755); err != nil {
		return err
	}
	return nil
}

// agentVersion agent 当前版本（与平台版本一致；构建脚本按版本命名产物）。
func agentVersion() string {
	if v := global.Version; v != "" {
		return strings.TrimPrefix(v, "v")
	}
	return "dev"
}

// mapUnameToGoarch uname -m 输出 → Go arch。
func mapUnameToGoarch(m string) string {
	switch m {
	case "x86_64":
		return "amd64"
	case "aarch64", "arm64":
		return "arm64"
	default:
		return m
	}
}
