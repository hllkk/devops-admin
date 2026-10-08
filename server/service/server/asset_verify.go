package server

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/utils/sshkey"
)

// asset_verify.go SSH 录入即验证（借鉴 spug _do_host_verify：保存资产时一体完成
// 「密码验证 → 平台公钥注入 → 私钥闭环 ping」，验证不过不落库；密码仅本次调用内存态）。
//
// 错误分诊（对齐 spug 语义）：
//   E00 = 主机不支持密码认证（加固堡垒机常见）
//   E01 = 主机不支持公钥认证（罕见，协议级配置）
//   E02 = 公钥注入成功但私钥认证仍失败（~/.ssh 权限/sshd_config AuthorizedKeysFile 非标准）
//   密码错 / 网络超时单独分诊。

// verifyTimeout 验证连接超时（保存是低频操作但不应让用户等 2 分钟）。
const verifyTimeout = 10 * time.Second

// VerifyAssetSSH 资产保存前 SSH 验证（physical/vm 专用）。
//
// 有密码：密码连接 → 注入平台公钥（幂等）→ 私钥闭环 ping（全过才算验证成功）。
// 无密码：仅私钥 ping（公钥已部署的主机改字段无需重输密码；失败提示重建立信任）。
// username 持久化到资产行（spug 模式），password 用后即弃不落任何存储。
func VerifyAssetSSH(host string, port int, username, password string) error {
	if username == "" {
		return errors.New("SSH 用户名不能为空")
	}
	keyDir := global.OPS_CONFIG.ServerModule.Agent.SSHKeyDir
	signer, err := sshkey.Signer(keyDir)
	if err != nil {
		return fmt.Errorf("平台密钥对初始化失败: %w", err)
	}
	if password != "" {
		if err := verifyWithPassword(host, port, username, password, keyDir); err != nil {
			return err
		}
	}
	return verifyWithPublicKey(host, port, username, signer, password != "")
}

// verifyWithPassword 密码路径：连接 + 注入平台公钥。
func verifyWithPassword(host string, port int, username, password, keyDir string) error {
	client, err := sshDialPassword(host, port, username, password, int(verifyTimeout.Seconds()))
	if err != nil {
		return diagnoseDialError(err)
	}
	defer client.Close()
	pubKey, err := sshkey.PublicKey(keyDir)
	if err != nil {
		return fmt.Errorf("读取平台公钥失败: %w", err)
	}
	if err := appendAuthorizedKey(client, pubKey); err != nil {
		return err
	}
	return nil
}

// verifyWithPublicKey 私钥闭环 ping（injected=本次是否走过了密码注入路径,
// 影响 E02 语义提示——公钥刚注入仍失败 vs 公钥从未部署）。
func verifyWithPublicKey(host string, port int, username string, signer ssh.Signer, injected bool) error {
	client, err := sshDialPublicKey(host, port, username, signer, int(verifyTimeout.Seconds()))
	if err != nil {
		authErr := diagnoseDialError(err)
		if injected {
			// 公钥已注入但私钥认证失败：权限/sshd_config 层问题（spug E02）
			return fmt.Errorf("%w；错误代码 E02(公钥已注入但密钥认证失败,检查目标机 ~/.ssh 权限与 sshd_config AuthorizedKeysFile)", authErr)
		}
		return fmt.Errorf("%w；平台公钥未部署到该主机,请输入密码重新建立信任", authErr)
	}
	defer client.Close()
	return nil
}

// diagnoseDialError SSH 连接错误分诊（密码错/超时/认证方式,对齐 spug 错误码语义）。
// x/crypto/ssh 无 paramiko 的异常类型细分，按错误内容分诊：
//
//	"unable to authenticate" = 认证失败（密码错或主机禁用该认证方式=E00）；
//	"no supported authentication" = 主机禁用全部可用认证方式（E00/E01 语义）；
//	net.Error Timeout = 网络不通。
func diagnoseDialError(err error) error {
	msg := err.Error()
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return errors.New("连接主机超时,请检查网络与端口")
	}
	if strings.Contains(msg, "no supported authentication methods") || strings.Contains(msg, "no common algorithm") {
		return errors.New("该主机不支持的认证方式(错误代码 E00/E01,检查 sshd_config 的 PasswordAuthentication/PubkeyAuthentication)")
	}
	if strings.Contains(msg, "unable to authenticate") {
		return fmt.Errorf("SSH 认证失败:检查用户名/密码是否正确(已含键盘交互回退;若仍失败请核对账号状态/登录限制如 AllowUsers): %v", err)
	}
	return fmt.Errorf("连接主机失败: %v", err)
}
