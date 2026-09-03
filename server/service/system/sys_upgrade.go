package system

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hllkk/devops-admin/server/global"
	systemRes "github.com/hllkk/devops-admin/server/model/system/response"
)

type UpgradeService struct{}

// updater 在 prod-net 内网的服务名:端口(compose 服务 updater,不对宿主暴露)
const updaterBaseURL = "http://updater:8090"

// manifestHTTPClient 拉取 manifest/转发 updater 专用:15s 超时,检查失败不影响业务
var manifestHTTPClient = &http.Client{Timeout: 15 * time.Second}

// ManifestPackage 发布服务器 manifest.json 里的包描述(与 deploy/release/build-release.sh 产物对齐)
type ManifestPackage struct {
	Type      string `json:"type"` // incr 增量 / full 全量
	URL       string `json:"url"`  // 相对发布服务器根的路径(/packages/xxx.tar.gz)
	Sha256    string `json:"sha256"`
	SizeBytes int64  `json:"sizeBytes"`
}

// Manifest 发布服务器 manifest.json 结构(静态文件,publish.sh 推送后人工改名生效)
type Manifest struct {
	Version           string            `json:"version"`
	BuildTime         string            `json:"buildTime"`
	ReleaseTime       string            `json:"releaseTime"`
	ChangeLog         string            `json:"changeLog"`
	MinUpgradeVersion string            `json:"minUpgradeVersion"`
	ForceUpgrade      bool              `json:"forceUpgrade"`
	Packages          []ManifestPackage `json:"packages"`
}

// GetVersion 版本信息(「关于」弹窗;版本经 ldflags 注入 global,升级完成后新进程启动即为新值)
func (s *UpgradeService) GetVersion() systemRes.VersionInfo {
	return systemRes.VersionInfo{
		AppName:     global.AppName,
		Version:     global.Version,
		BuildTime:   global.BuildTime,
		Description: global.Description,
	}
}

// CheckUpdate 拉取 manifest 与当前版本比对。
// 返回 hasUpdate=false 的各分支都带用户可读 msg(发布服务器未配置/不可达/已是最新)。
func (s *UpgradeService) CheckUpdate(ctx context.Context) (systemRes.UpgradeCheckResult, error) {
	res := systemRes.UpgradeCheckResult{CurrentVersion: global.Version}
	cfg := global.OPS_CONFIG.Upgrade
	if cfg.UpdateServerUrl == "" {
		res.Message = "未配置发布服务器地址(.env 的 UPDATE_SERVER_URL)"
		return res, nil
	}
	manifest, err := s.fetchManifest(ctx, cfg.UpdateServerUrl)
	if err != nil {
		res.Message = "发布服务器不可达: " + err.Error()
		return res, nil
	}
	res.Version = manifest.Version
	res.ChangeLog = manifest.ChangeLog
	res.ReleaseTime = manifest.ReleaseTime
	res.MinUpgradeVersion = manifest.MinUpgradeVersion
	res.ForceUpgrade = manifest.ForceUpgrade
	// 包选择:优先增量(日常小版本体量小);无增量包(大版本)用全量
	for _, p := range manifest.Packages {
		if p.Type == "incr" {
			res.Package = &systemRes.UpgradePackageInfo{Type: p.Type, URL: p.URL, Sha256: p.Sha256, SizeBytes: p.SizeBytes}
			break
		}
	}
	if res.Package == nil {
		for _, p := range manifest.Packages {
			if p.Type == "full" {
				res.Package = &systemRes.UpgradePackageInfo{Type: p.Type, URL: p.URL, Sha256: p.Sha256, SizeBytes: p.SizeBytes}
				break
			}
		}
	}
	if res.Package == nil {
		res.Message = "manifest 未包含可用升级包"
		return res, nil
	}
	switch CompareVersion(manifest.Version, global.Version) {
	case 1:
		res.HasUpdate = true
		res.Message = "发现新版本 " + manifest.Version
	case 0:
		// 主版本号相同(如 v0.2.1-a 与 v0.2.1-b):字符串不等也视为有更新,同版本重发布可生效
		if manifest.Version != global.Version {
			res.HasUpdate = true
			res.Message = "发现新构建 " + manifest.Version
		} else {
			res.Message = "已是最新版本"
		}
	default:
		res.Message = "manifest 版本不高于当前版本,忽略(" + manifest.Version + " <= " + global.Version + ")"
	}
	return res, nil
}

// StartUpgrade 触发在线升级:实时拉 manifest 校验确有更新后,转发 updater 执行
// (下载/校验/安装进度经 GetStatus 轮询)。updater 活跃中会返回其 409 文案。
func (s *UpgradeService) StartUpgrade(ctx context.Context) (systemRes.UpgradeStartResult, error) {
	check, err := s.CheckUpdate(ctx)
	if err != nil {
		return systemRes.UpgradeStartResult{}, err
	}
	if !check.HasUpdate || check.Package == nil {
		return systemRes.UpgradeStartResult{Accepted: false, Message: check.Message}, nil
	}
	base := strings.TrimRight(global.OPS_CONFIG.Upgrade.UpdateServerUrl, "/")
	body, _ := json.Marshal(map[string]string{
		"downloadUrl": base + check.Package.URL,
		"sha256":      check.Package.Sha256,
		"version":     check.Version,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, updaterBaseURL+"/api/upgrade", bytes.NewReader(body))
	if err != nil {
		return systemRes.UpgradeStartResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token := global.OPS_CONFIG.Upgrade.UpdaterToken; token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := manifestHTTPClient.Do(req)
	if err != nil {
		return systemRes.UpgradeStartResult{Accepted: false, Message: "升级服务(updater)不可达: " + err.Error()}, nil
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
	var ur struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal(out, &ur)
	switch resp.StatusCode {
	case http.StatusAccepted:
		return systemRes.UpgradeStartResult{Accepted: true, Version: check.Version, Message: "升级已开始"}, nil
	case http.StatusConflict:
		return systemRes.UpgradeStartResult{Accepted: false, Message: "已有升级进行中: " + ur.Error}, nil
	default:
		return systemRes.UpgradeStartResult{Accepted: false, Message: fmt.Sprintf("updater 拒绝(HTTP %d): %s", resp.StatusCode, ur.Error)}, nil
	}
}

// GetStatus 代理 updater 升级状态机(updater 不可达时返回 unreachable 态,前端据此提示)
func (s *UpgradeService) GetStatus(ctx context.Context) systemRes.UpgradeStateInfo {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, updaterBaseURL+"/api/status", nil)
	if err != nil {
		return systemRes.UpgradeStateInfo{State: "unreachable", Message: err.Error()}
	}
	resp, err := manifestHTTPClient.Do(req)
	if err != nil {
		return systemRes.UpgradeStateInfo{State: "unreachable", Message: "升级服务(updater)不可达"}
	}
	defer resp.Body.Close()
	var st systemRes.UpgradeStateInfo
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		return systemRes.UpgradeStateInfo{State: "unreachable", Message: "状态解析失败: " + err.Error()}
	}
	return st
}

// fetchManifest 拉取并解析发布服务器 manifest.json
func (s *UpgradeService) fetchManifest(ctx context.Context, base string) (*Manifest, error) {
	url := strings.TrimRight(base, "/") + "/manifest.json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := manifestHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var m Manifest
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&m); err != nil {
		return nil, err
	}
	if m.Version == "" {
		return nil, fmt.Errorf("manifest 缺 version 字段")
	}
	return &m, nil
}

// CompareVersion 比较两个版本号(支持 v0.2.1 / v0.2.1-abc123 / 0.2 形态,按 . 分段数字比较,
// 忽略 v 前缀与 - 后缀):1=v1 大于 v2,0=主版本号相等,-1=v1 小于 v2。
// 后缀(构建 sha)不参与大小,同主版本号的重发布靠字符串不等识别。
func CompareVersion(v1, v2 string) int {
	p1, p2 := versionSegments(v1), versionSegments(v2)
	n := len(p1)
	if len(p2) > n {
		n = len(p2)
	}
	for i := 0; i < n; i++ {
		a, b := 0, 0
		if i < len(p1) {
			a = p1[i]
		}
		if i < len(p2) {
			b = p2[i]
		}
		if a != b {
			if a > b {
				return 1
			}
			return -1
		}
	}
	return 0
}

// versionSegments 取 - 前主段(去 v 前缀)按 . 切成数字(非法段按 0)
func versionSegments(v string) []int {
	main := v
	if i := strings.IndexByte(v, '-'); i >= 0 {
		main = v[:i]
	}
	main = strings.TrimPrefix(main, "v")
	parts := strings.Split(main, ".")
	out := make([]int, len(parts))
	for i, p := range parts {
		n, _ := strconv.Atoi(strings.TrimSpace(p))
		out[i] = n
	}
	return out
}
