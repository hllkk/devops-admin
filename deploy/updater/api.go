package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// runDaemon 常驻模式：HTTP API + 升级包下载/校验/解压 + 发起安装 job。
//
// 环境变量（compose 注入）：
//
//	STACK_DIR       安装目录挂载点（默认 /opt/stack）
//	UPDATER_TOKEN   写接口鉴权 token（server 与 updater 同源 .env 注入；空=不鉴权，仅限可信内网）
//	UPDATER_IMAGE   当前 updater 镜像（安装 job 容器复用本镜像；compose 插值为版本化 tag）
//	UPDATER_NETWORK 安装 job 接入的网络（健康检查 web 用；compose 固定命名 devops-admin-prod_prod-net）
//	UPDATER_LISTEN  监听地址（默认 :8090，仅 prod-net 内网可达，不对宿主暴露）
func runDaemon() {
	addr := envOr("UPDATER_LISTEN", ":8090")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /api/status", handleStatus)
	mux.HandleFunc("POST /api/upgrade", requireToken(handleUpgrade))
	log.Printf("[daemon] devops-admin-updater %s 监听 %s（stack=%s）", Version, addr, stackDir())
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("[daemon] 启动失败: %v", err)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "version": Version, "buildTime": BuildTime})
}

func handleStatus(w http.ResponseWriter, _ *http.Request) {
	s, err := loadState()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, s)
}

// UpgradeRequest server 下发的升级指令（manifest 里已按环境选好包）
type UpgradeRequest struct {
	DownloadURL string `json:"downloadUrl"` // 升级包绝对 URL（发布服务器）
	Sha256      string `json:"sha256"`      // 包 sha256（安装前置校验）
	Version     string `json:"version"`     // 目标版本（亦是缓存目录名）
}

func handleUpgrade(w http.ResponseWriter, r *http.Request) {
	var req UpgradeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "请求体解析失败: " + err.Error()})
		return
	}
	if req.DownloadURL == "" || req.Sha256 == "" || req.Version == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "downloadUrl/sha256/version 均必填"})
		return
	}
	s, err := loadState()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if s.active() {
		writeJSON(w, http.StatusConflict, map[string]any{
			"error": fmt.Sprintf("升级进行中（%s %s），请等待完成后再试", s.State, s.Version),
			"state": s,
		})
		return
	}
	// 异步执行下载→校验→解压→发起 job；立即返回 202，进度经 /api/status 轮询
	go executeUpgrade(req)
	writeJSON(w, http.StatusAccepted, map[string]any{"accepted": true, "version": req.Version})
}

// executeUpgrade daemon 侧升级流水线（goroutine）：
// downloading → verifying → unpacking → 发起安装 job（installing 及终态由 job 写）
func executeUpgrade(req UpgradeRequest) {
	start := time.Now()
	pkgDir := filepath.Join(stackDir(), "upgrade-cache", req.Version)
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		saveState(&UpgradeState{State: StateFailed, Version: req.Version, Message: "建缓存目录失败: " + err.Error()})
		return
	}
	dest := filepath.Join(stackDir(), "upgrade-cache", req.Version+".download.tar.gz")

	// 1. 下载（断点续传，失败可直接重发请求续传）
	saveState(&UpgradeState{State: StateDownloading, Version: req.Version, Progress: 0, Message: "开始下载"})
	err := downloadWithResume(req.DownloadURL, dest, func(done, total int64) {
		pct := 0
		if total > 0 {
			pct = int(done * 100 / total)
		}
		saveState(&UpgradeState{State: StateDownloading, Version: req.Version, Progress: pct,
			Message: fmt.Sprintf("已下载 %dMiB / %dMiB", done>>20, total>>20)})
	})
	if err != nil {
		saveState(&UpgradeState{State: StateFailed, Version: req.Version, Message: "下载失败（重发请求可断点续传）: " + err.Error()})
		return
	}

	// 2. sha256 校验（不符即拒装，杜绝损坏/被篡改的包进入安装阶段）
	saveState(&UpgradeState{State: StateVerifying, Version: req.Version, Message: "校验 sha256"})
	got, err := sha256File(dest)
	if err != nil {
		saveState(&UpgradeState{State: StateFailed, Version: req.Version, Message: "读取包失败: " + err.Error()})
		return
	}
	if got != req.Sha256 {
		// 校验失败删除损坏文件，避免下次续传到坏数据
		os.Remove(dest)
		saveState(&UpgradeState{State: StateFailed, Version: req.Version,
			Message: fmt.Sprintf("sha256 不符（期望 %s 实得 %s），已删除损坏包", req.Sha256, got)})
		return
	}

	// 3. 解压到缓存目录（strip 顶层 devops-admin-/）
	saveState(&UpgradeState{State: StateUnpacking, Version: req.Version, Message: "解压升级包"})
	if err := extractTarGz(dest, pkgDir); err != nil {
		saveState(&UpgradeState{State: StateFailed, Version: req.Version, Message: "解压失败: " + err.Error()})
		return
	}
	// 解压完成即可删包腾磁盘（升级包数百 MiB~2G）
	os.Remove(dest)

	// 4. 发起安装 job 容器（daemon 直管，与 updater 容器生命周期解耦）
	saveState(&UpgradeState{State: StateInstalling, Version: req.Version, Message: "安装 job 已发起"})
	if err := launchInstallJob(req.Version); err != nil {
		saveState(&UpgradeState{State: StateFailed, Version: req.Version, Message: "发起安装 job 失败: " + err.Error()})
		return
	}
	log.Printf("[daemon] 升级 %s：下载+校验+解压完成（%s），安装 job 接管", req.Version, fmtDuration(time.Since(start)))
}

// launchInstallJob docker run 一次性安装容器（--rm 跑完即清，日志落 stack/upgrade-install.log）
func launchInstallJob(version string) error {
	image := envOr("UPDATER_IMAGE", "devops-admin/updater:prod")
	network := envOr("UPDATER_NETWORK", "devops-admin-prod_prod-net")
	// 挂载必须用宿主路径：stackDir() 是容器内路径，直接传给 docker run -v 会挂到
	// 宿主同名空目录（docker 自动创建），job 将找不到升级包。
	// 经 docker inspect 本容器反查 /opt/stack 挂载的真实宿主源路径，零配置且兼容自定义安装目录
	hostStack, err := stackHostPath()
	if err != nil {
		return fmt.Errorf("反查安装目录宿主路径失败: %w", err)
	}
	// 幂等：残留同名 job（上次异常退出未 --rm）先清
	if err := exec.Command("docker", "rm", "-f", "devops-upgrade-job").Run(); err != nil {
		// 不存在属正常，其余错误也不阻塞（run 阶段会暴露）
		log.Printf("[daemon] 清理残留 job: %v（不存在则忽略）", err)
	}
	args := []string{"run", "-d", "--rm",
		"--name", "devops-upgrade-job",
		"--network", network,
		"-v", "/var/run/docker.sock:/var/run/docker.sock",
		"-v", hostStack + ":/opt/stack",
		"-e", "STACK_DIR=/opt/stack",
		// job 输出同时落安装目录，--rm 容器销毁后日志仍可查
		"--entrypoint", "sh", image,
		"-c", "/app/updater install --version " + version + " > /opt/stack/upgrade-install.log 2>&1",
	}
	out, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker run 失败: %v: %s", err, string(out))
	}
	log.Printf("[daemon] 安装 job 已启动: %s（stack=%s）", string(out), hostStack)
	return nil
}

// stackHostPath 反查 /opt/stack 挂载在宿主的真实路径（docker inspect 本容器）
func stackHostPath() (string, error) {
	cid, err := os.ReadFile("/etc/hostname")
	if err != nil {
		return "", fmt.Errorf("读容器 id 失败: %w", err)
	}
	inspectFmt := `{{range .Mounts}}{{if eq .Destination "/opt/stack"}}{{.Source}}{{end}}{{end}}`
	out, err := runCmd(time.Minute, "docker", "inspect", "--format", inspectFmt, strings.TrimSpace(string(cid)))
	if err != nil {
		return "", fmt.Errorf("docker inspect 失败: %s: %w", out, err)
	}
	hostPath := strings.TrimSpace(out)
	if hostPath == "" {
		return "", fmt.Errorf("本容器无 /opt/stack 挂载（compose 挂载缺失？）")
	}
	return hostPath, nil
}

// requireToken 写接口鉴权（UPDATER_TOKEN 非空时校验 Bearer）
func requireToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := os.Getenv("UPDATER_TOKEN")
		if token != "" && r.Header.Get("Authorization") != "Bearer "+token {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "token 无效"})
			return
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
