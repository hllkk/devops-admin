package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSha256File(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.bin")
	content := []byte("hello devops-admin updater")
	if err := os.WriteFile(f, content, 0o600); err != nil {
		t.Fatal(err)
	}
	want := sha256.Sum256(content)
	got, err := sha256File(f)
	if err != nil {
		t.Fatal(err)
	}
	if got != hex.EncodeToString(want[:]) {
		t.Fatalf("sha256 不符: %s != %s", got, hex.EncodeToString(want[:]))
	}
}

func TestStripFirstSegment(t *testing.T) {
	cases := map[string]string{
		"devops-admin-upgrade/docker-compose.yml": "docker-compose.yml",
		"devops-admin-upgrade/config/config.yaml": "config/config.yaml",
		"devops-admin-upgrade/":                   "",
		"devops-admin-upgrade":                    "",
		"./devops-admin-upgrade/VERSION":          "VERSION",
	}
	for in, want := range cases {
		if got := stripFirstSegment(in); got != want {
			t.Errorf("stripFirstSegment(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestWithinDir(t *testing.T) {
	root := "/opt/stack/upgrade-cache"
	cases := map[string]bool{
		filepath.Join(root, "docker-compose.yml"):    true,
		filepath.Join(root, "config", "config.yaml"): true,
		filepath.Clean(root + "/../.env"):            false,
		filepath.Clean(root + "/../../etc/passwd"):   false,
	}
	for target, want := range cases {
		if got := withinDir(root, target); got != want {
			t.Errorf("withinDir(%q) = %v, want %v", target, got, want)
		}
	}
}

func TestReplaceEnvValue(t *testing.T) {
	dir := t.TempDir()
	env := filepath.Join(dir, ".env")
	if err := os.WriteFile(env, []byte("TZ=Asia/Shanghai\nAPP_VERSION=1.0.0-old\nWEB_PORT=8080\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := replaceEnvValue(env, "APP_VERSION", "1.0.1-new"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(env)
	if !strings.Contains(string(data), "APP_VERSION=1.0.1-new") || strings.Contains(string(data), "1.0.0-old") {
		t.Fatalf("替换失败: %s", data)
	}
	if !strings.Contains(string(data), "WEB_PORT=8080") {
		t.Fatalf("其余行被破坏: %s", data)
	}
	// key 不存在时追加
	if err := replaceEnvValue(env, "UPDATER_TOKEN", "abc"); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(env)
	if !strings.Contains(string(data), "UPDATER_TOKEN=abc") {
		t.Fatalf("追加失败: %s", data)
	}
	// 权限保留
	fi, _ := os.Stat(env)
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("权限被改: %v", fi.Mode().Perm())
	}
}

// buildTarGz 构造测试用升级包（顶层目录 top）
func buildTarGz(t *testing.T, top string, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		if err := tw.WriteHeader(&tar.Header{Name: top + "/" + name, Mode: 0o644, Size: int64(len(content))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestExtractTarGz(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "pkg.tar.gz")
	if err := os.WriteFile(archive, buildTarGz(t, "devops-admin-upgrade", map[string]string{
		"VERSION":            "1.0.2",
		"docker-compose.yml": "name: devops-admin-prod",
		"config/config.yaml": "db:",
	}), 0o600); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "out")
	if err := extractTarGz(archive, dest); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"VERSION": "1.0.2", "docker-compose.yml": "name: devops-admin-prod", "config/config.yaml": "db:"} {
		got, err := os.ReadFile(filepath.Join(dest, name))
		if err != nil {
			t.Fatalf("缺文件 %s: %v", name, err)
		}
		if string(got) != want {
			t.Fatalf("%s 内容不符: %s", name, got)
		}
	}
}

func TestExtractTarGzRejectPathTraversal(t *testing.T) {
	dir := t.TempDir()
	// 顶层目录为 .. 时 strip 后可越出 dest —— 必须被拒绝
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: "../../etc/crontab", Mode: 0o644, Size: 5}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte("evil\n")); err != nil {
		t.Fatal(err)
	}
	tw.Close()
	gz.Close()
	archive := filepath.Join(dir, "evil.tar.gz")
	os.WriteFile(archive, buf.Bytes(), 0o600)

	dest := filepath.Join(dir, "out")
	if err := extractTarGz(archive, dest); err == nil {
		t.Fatal("路径穿越包未被拒绝")
	}
}

func TestStateReadWrite(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("STACK_DIR", dir)
	s := &UpgradeState{State: StateDownloading, Version: "1.0.2", Progress: 42}
	if err := saveState(s); err != nil {
		t.Fatal(err)
	}
	got, err := loadState()
	if err != nil {
		t.Fatal(err)
	}
	if got.State != StateDownloading || got.Version != "1.0.2" || got.Progress != 42 || got.UpdateTime == "" {
		t.Fatalf("状态回读不符: %+v", got)
	}
	// 无状态文件 → idle
	os.Remove(filepath.Join(dir, "upgrade-state.json"))
	got, err = loadState()
	if err != nil || got.State != StateIdle {
		t.Fatalf("缺省态应为 idle: %+v %v", got, err)
	}
	// 损坏 JSON → failed（而非 idle，避免掩盖问题）
	os.WriteFile(filepath.Join(dir, "upgrade-state.json"), []byte("{broken"), 0o644)
	got, err = loadState()
	if err != nil || got.State != StateFailed {
		t.Fatalf("损坏态应为 failed: %+v %v", got, err)
	}
	// 活跃态判定
	active := &UpgradeState{State: StateInstalling}
	if !active.active() {
		t.Fatal("installing 应为活跃态")
	}
	idle := &UpgradeState{State: StateSuccess}
	if idle.active() {
		t.Fatal("success 不应为活跃态")
	}
}
