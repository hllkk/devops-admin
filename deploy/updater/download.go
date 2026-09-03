package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// downloadWithResume 断点续传下载到 dest，onProgress 收到 (已下载字节, 总字节)。
//
// 已有部分文件时发 Range 续传（nginx 静态站天然支持 206）；服务器不支持 Range
// 返回 200 则从头重写。下载中断（网络断/容器重启）重试时从断点继续。
func downloadWithResume(url, dest string, onProgress func(done, total int64)) error {
	var existing int64
	if fi, err := os.Stat(dest); err == nil {
		existing = fi.Size()
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if existing > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", existing))
	}
	// 整体超时 30 分钟：内网 2G 全量包也远快于此；慢于它的环境应手工 scp 全量包。
	// 超时中断后重发升级请求即可从断点续传，不从头再来。
	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	flags := os.O_CREATE | os.O_WRONLY
	switch resp.StatusCode {
	case http.StatusPartialContent: // 206 续传：追加
		flags |= os.O_APPEND
	case http.StatusOK: // 200：服务器忽略 Range，从头重写（existing 作废）
		existing = 0
		flags |= os.O_TRUNC
	default:
		return fmt.Errorf("下载失败：HTTP %d（%s）", resp.StatusCode, url)
	}
	f, err := os.OpenFile(dest, flags, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	total := existing
	if resp.ContentLength > 0 {
		total = existing + resp.ContentLength
	}

	progressAt := existing
	var done int64 = existing
	buf := make([]byte, 256*1024)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := f.Write(buf[:n]); err != nil {
				return err
			}
			done += int64(n)
			// 进度节流：每 4MiB 或收尾时回调
			if onProgress != nil && (done-progressAt >= 4*1024*1024 || rerr != nil) {
				onProgress(done, total)
				progressAt = done
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return fmt.Errorf("下载中断（可重试续传）: %w", rerr)
		}
	}
	if onProgress != nil {
		onProgress(done, total)
	}
	return nil
}

// sha256File 流式计算文件 sha256（121M~2G 包全量读一遍，本地磁盘秒级）
func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// extractTarGz 解压升级包到 destDir，strip 掉顶层目录（打包结构 devops-admin-/...）。
//
// 安全校验：拒绝绝对路径与 .. 穿越（升级包来自网络，必须当不可信输入处理）。
func extractTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		// strip 首段路径（devops-admin-upgrade/ 或 devops-admin-release/）
		rel := stripFirstSegment(hdr.Name)
		if rel == "" || rel == "." {
			continue
		}
		target := filepath.Join(destDir, rel)
		// 穿越防护：解出的最终路径必须落在 destDir 内
		if !withinDir(destDir, target) {
			return fmt.Errorf("包内路径可疑，拒绝解压: %s", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode)&0o777)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		case tar.TypeSymlink:
			// 升级包不含软链；出现即拒绝，杜绝链接触发的逃逸
			return fmt.Errorf("包内含软链，拒绝解压: %s", hdr.Name)
		default:
			return fmt.Errorf("包内含非常规文件类型(%c)，拒绝: %s", hdr.Typeflag, hdr.Name)
		}
	}
}

// stripFirstSegment 去掉路径首段（a/b/c → b/c）；仅剩首段（a 或 a/）返回空
func stripFirstSegment(p string) string {
	p = filepath.ToSlash(p)
	// 归一化 ./ 前缀
	p = strings.TrimPrefix(p, "./")
	idx := strings.IndexByte(p, '/')
	if idx < 0 {
		return ""
	}
	return p[idx+1:]
}

// withinDir target 是否在 root 目录内（两者均已 Clean）
func withinDir(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// replaceEnvValue 更新 env 文件中 key 的值（保留其余行与顺序）；
// key 不存在时追加到末尾。保留原文件权限（.env 为 600）。
func replaceEnvValue(path, key, value string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	prefix := key + "="
	replaced := false
	for i, line := range lines {
		if strings.HasPrefix(line, prefix) {
			lines[i] = prefix + value
			replaced = true
		}
	}
	if !replaced {
		lines = append(lines, prefix+value)
	}
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), fi.Mode())
}

// fmtDuration 下载耗时展示用
func fmtDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}
