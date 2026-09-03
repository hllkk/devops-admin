package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// runInstall 安装 job（由 daemon 经 docker run 发起的一次性容器内执行）：
//
//	load 镜像 → 替换编排资产 → 更新 .env 版本 → 镜像完整性校验 →
//	up -d --force-recreate web server updater → onlyoffice 网络归属按需迁移 →
//	健康检查 → 状态落盘
//
// 任一步失败写 failed 终态并退出非零；资产替换发生在 load 成功之后，
// load 失败时 .env/编排未动，旧栈继续运行，重发升级请求即可重试。
func runInstall(args []string) error {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	version := fs.String("version", "", "要安装的版本（升级缓存目录名）")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *version == "" {
		return fmt.Errorf("缺少 --version")
	}
	stack := stackDir()
	pkgDir := filepath.Join(stack, "upgrade-cache", *version)

	// ---- 1. 包完整性前置检查 ----
	for _, must := range []string{
		filepath.Join(pkgDir, "docker-compose.yml"),
		filepath.Join(pkgDir, "VERSION"),
		filepath.Join(pkgDir, ".env.example"),
	} {
		if _, err := os.Stat(must); err != nil {
			fail(*version, fmt.Sprintf("升级包不完整，缺 %s: %v", filepath.Base(must), err))
			return err
		}
	}
	tars, err := filepath.Glob(filepath.Join(pkgDir, "images", "*.tar"))
	if err != nil || len(tars) == 0 {
		fail(*version, "升级包缺 images/*.tar")
		return fmt.Errorf("升级包缺镜像 tar")
	}

	// ---- 2. docker load 镜像（版本化 tag，不覆盖旧镜像，可回滚）----
	for _, t := range tars {
		if out, err := runCmd(10*time.Minute, "docker", "load", "-i", t); err != nil {
			fail(*version, fmt.Sprintf("docker load %s 失败: %v: %s", filepath.Base(t), err, out))
			return err
		}
	}

	// ---- 3. 替换编排资产（compose/.env.example/config/nginx/VERSION/BUILD_TIME）----
	// .env 不在包内，机密与数据路径不受影响；
	// config/config.yaml 是生产机密资产（构建期注入的 credential-key 为 AI 网关凭证
	// AES 密钥，轮换会使历史加密凭证不可解）——替换前暂存、替换后原样还原，
	// 生产现值永不被升级包内的入库默认版覆盖
	assets := []string{"docker-compose.yml", ".env.example", "config", "nginx", "VERSION", "BUILD_TIME"}
	protected := []string{"config/config.yaml"}
	backupDir := filepath.Join(stack, "upgrade-cache", ".protected")
	for _, rel := range protected {
		// 不存在 = 首次部署无既有配置，静默跳过
		if err := moveIfPresent(filepath.Join(stack, rel), filepath.Join(backupDir, rel)); err != nil {
			fail(*version, fmt.Sprintf("暂存生产配置 %s 失败: %v", rel, err))
			return err
		}
	}
	for _, name := range assets {
		src := filepath.Join(pkgDir, name)
		if _, err := os.Stat(src); err != nil {
			continue // 非必须资产（如 BUILD_TIME）缺失不阻塞
		}
		dst := filepath.Join(stack, name)
		if err := os.RemoveAll(dst); err != nil {
			fail(*version, fmt.Sprintf("清理旧资产 %s 失败: %v", name, err))
			return err
		}
		if err := copyPath(src, dst); err != nil {
			fail(*version, fmt.Sprintf("拷贝资产 %s 失败: %v", name, err))
			return err
		}
	}
	for _, rel := range protected {
		// 原样还原（覆盖升级包带入的同名入库默认版）
		if err := moveIfPresent(filepath.Join(backupDir, rel), filepath.Join(stack, rel)); err != nil {
			fail(*version, fmt.Sprintf("还原生产配置 %s 失败: %v", rel, err))
			return err
		}
	}
	os.RemoveAll(backupDir)

	// ---- 4. .env 写入新版本（回滚 = 改回旧值 + up）----
	envFile := filepath.Join(stack, ".env")
	if err := replaceEnvValue(envFile, "APP_VERSION", *version); err != nil {
		fail(*version, "更新 .env 的 APP_VERSION 失败: "+err.Error())
		return err
	}

	// ---- 5. 镜像完整性校验（缺失立即失败，避免 up 卡在 registry 拉取超时）----
	compose, err := composeBaseArgs()
	if err != nil {
		fail(*version, err.Error())
		return err
	}
	cfgArgs := append(compose, "config", "--images")
	out, err := runCmd(time.Minute, cfgArgs[0], cfgArgs[1:]...)
	if err != nil {
		fail(*version, "compose config 失败: "+out)
		return err
	}
	for _, img := range splitLines(out) {
		if img == "" {
			continue
		}
		if o, err := runCmd(time.Minute, "docker", "image", "inspect", img); err != nil {
			fail(*version, fmt.Sprintf("镜像缺失: %s（第三方镜像缺失说明包类型选错，应改用全量包）: %s", img, o))
			return err
		}
	}

	// ---- 6. 重建自研服务（含 updater 自身；job 容器独立于此不受影响）----
	// 只动自研三件：pgsql/redis/rustfs/litellm 等基础设施容器不重建，数据面与
	// AI 网关转发面升级窗口内持续可用（litellm 不动也无 litellm 建表时序问题）
	upArgs := append(compose, "up", "-d", "--no-build", "--force-recreate", "web", "server", "updater")
	if out, err := runCmd(10*time.Minute, upArgs[0], upArgs[1:]...); err != nil {
		fail(*version, "compose up 失败（可手工排查后重试，或改 .env APP_VERSION 回滚）: "+out)
		return err
	}

	// ---- 7. 健康检查（job 与栈同网络，直接访问服务名 web）----
	if err := waitHealthy("http://web/", 120*time.Second); err != nil {
		fail(*version, "健康检查超时（容器可能仍在启动，docker compose logs -f server 排查）: "+err.Error())
		return err
	}

	// ---- 8. 终态 ----
	if err := saveState(&UpgradeState{State: StateSuccess, Progress: 100, Version: *version,
		Message: "升级完成并健康"}); err != nil {
		return err
	}
	fmt.Printf("[install] 升级完成: %s\n", *version)
	// 升级成功的缓存目录保留一个版本作重装/排查源，更早的清掉
	pruneOldCaches(*version)
	return nil
}

// fail 写 failed 终态（尽力而为，失败不影响错误返回）
func fail(version, msg string) {
	_ = saveState(&UpgradeState{State: StateFailed, Version: version, Message: msg})
}

// composeBaseArgs job 内执行 compose 的公共参数。
//
// 关键：job 在容器内跑 compose（经 docker.sock 指挥宿主 daemon），compose 文件里的
// 相对路径 bind mount（./、./config/...）会被解析成绝对路径传给 daemon——若以容器内
// 路径为基准，daemon 收到的就是宿主不存在的 /opt/stack/... 并自动创建空目录挂载
// （pg 配置挂空文件直接崩溃）。因此：
//
//	-f                用容器内路径读 compose 文件（job 挂了 /opt/stack）
//	--project-directory 用反查到的宿主真实路径（相对挂载的解析基准，daemon 侧正确落位）
//	--env-file        用容器内路径读 .env（project-directory 已指向宿主路径，默认查找会落空）
func composeBaseArgs() ([]string, error) {
	stack := stackDir()
	hostStack, err := stackHostPath()
	if err != nil {
		return nil, fmt.Errorf("反查安装目录宿主路径失败: %w", err)
	}
	return []string{"docker", "compose",
		"-f", filepath.Join(stack, "docker-compose.yml"),
		"--project-directory", hostStack,
		"--env-file", filepath.Join(stack, ".env"),
	}, nil
}

// runCmd 带超时执行命令，返回合并输出；超时/失败时错误带输出
func runCmd(timeout time.Duration, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	done := make(chan struct{})
	var out []byte
	var err error
	go func() {
		out, err = cmd.CombinedOutput()
		close(done)
	}()
	select {
	case <-done:
		return string(out), err
	case <-time.After(timeout):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		<-done
		return string(out), fmt.Errorf("命令超时（%s）", timeout)
	}
}

// waitHealthy 轮询 URL 直到 200 或超时
func waitHealthy(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("%s 在 %s 内未就绪", url, timeout)
}

// copyPath 递归拷贝文件或目录（资产替换用）
func copyPath(src, dst string) error {
	fi, err := os.Stat(src)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		if err := os.MkdirAll(dst, fi.Mode().Perm()); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if err := copyPath(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fi.Mode().Perm())
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// moveIfPresent src 存在则原子 mv 到 dst（不存在静默跳过，父目录自动创建）；
// 生产机密资产的「暂存→替换→还原」与升级缓存目录整理共用
func moveIfPresent(src, dst string) error {
	if _, err := os.Stat(src); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.Rename(src, dst)
}

// pruneOldCaches 只保留当前版本的升级缓存，更早版本目录删除（省磁盘）
func pruneOldCaches(keepVersion string) {
	cacheRoot := filepath.Join(stackDir(), "upgrade-cache")
	entries, err := os.ReadDir(cacheRoot)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() && name != keepVersion {
			_ = os.RemoveAll(filepath.Join(cacheRoot, name))
		}
	}
	// 半成品下载文件一并清理
	_ = os.RemoveAll(filepath.Join(cacheRoot, keepVersion+".download.tar.gz"))
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
