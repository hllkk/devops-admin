package upload

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// 网盘上传分片暂存(与 media 上传隔离:chunkRoot()/disk/{sessionID})。
// 复用 chunk.go 的 chunkRoot();sessionID 为网盘 disk_upload_sessions 主键(int64)。

// DiskChunkDir 网盘某次上传会话的分片暂存目录。
func DiskChunkDir(sessionID int64) string {
	return filepath.Join(chunkRoot(), "disk", fmt.Sprintf("%d", sessionID))
}

// SaveDiskChunkStream 流式写入分片到 <index>.part.tmp,边写边算 md5,
// 返回写入字节数与 md5。调用方校验 md5/大小后用 CommitDiskChunk 提交或丢弃。
// 替代旧 SaveDiskChunk 的「整片 io.ReadAll 入内存」写法——内存恒定,不随分片大小增长。
func SaveDiskChunkStream(sessionID int64, index int, reader io.Reader) (int64, string, error) {
	dir := DiskChunkDir(sessionID)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return 0, "", err
	}
	tmp := filepath.Join(dir, fmt.Sprintf("%d.part.tmp", index))
	out, err := os.Create(tmp)
	if err != nil {
		return 0, "", err
	}
	defer out.Close()
	h := md5.New()
	n, err := io.Copy(io.MultiWriter(out, h), reader)
	if err != nil {
		return 0, "", err
	}
	return n, hex.EncodeToString(h.Sum(nil)), nil
}

// CommitDiskChunk 提交或丢弃流式分片:keep=true 将 .tmp 重命名为 .part(覆盖旧值,幂等);
// keep=false 删除 .tmp。校验未通过时调 keep=false 回收临时文件。
func CommitDiskChunk(sessionID int64, index int, keep bool) error {
	dir := DiskChunkDir(sessionID)
	tmp := filepath.Join(dir, fmt.Sprintf("%d.part.tmp", index))
	if !keep {
		return os.Remove(tmp)
	}
	part := filepath.Join(dir, fmt.Sprintf("%d.part", index))
	_ = os.Remove(part) // 覆盖写幂等:先删旧 .part
	return os.Rename(tmp, part)
}

// MergeDiskChunks 按 0..total-1 顺序流式合并到 dstPath,边合并边算 md5,返回成品 md5。
func MergeDiskChunks(sessionID int64, total int, dstPath string) (string, error) {
	if err := os.MkdirAll(filepath.Dir(dstPath), os.ModePerm); err != nil {
		return "", err
	}
	out, err := os.Create(dstPath)
	if err != nil {
		return "", err
	}
	defer out.Close()
	h := md5.New()
	w := io.MultiWriter(out, h)
	for i := 0; i < total; i++ {
		in, err := os.Open(filepath.Join(DiskChunkDir(sessionID), fmt.Sprintf("%d.part", i)))
		if err != nil {
			return "", fmt.Errorf("缺少分片 %d: %w", i, err)
		}
		if _, err := io.Copy(w, in); err != nil {
			in.Close()
			return "", err
		}
		in.Close()
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// RemoveDiskUploadDir 删除网盘某次上传的分片暂存目录。
func RemoveDiskUploadDir(sessionID int64) error {
	return os.RemoveAll(DiskChunkDir(sessionID))
}
