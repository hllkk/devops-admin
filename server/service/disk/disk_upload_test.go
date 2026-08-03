package disk

import "testing"

// TestRelPathDirPart 验证从文件相对路径(webkitRelativePath,含文件名末段)取目录段。
// 回归保护:上传 merge 懒建目录时只建文件名之前的目录段,文件名段交给 Merge 建文件节点。
// 若此函数错误地保留文件名末段,EnsureFolderByRelPath 会把文件名建成目录节点,
// 出现"文件被存成文件夹、文件夹下又套同名文件"(用户报告的 123/123.txt → 123.txt/ 目录 + 123.txt 文件)。
func TestRelPathDirPart(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"顶层文件含目录前缀", "123/123.txt", "123"},
		{"两级目录+文件", "123/456/456.txt", "123/456"},
		{"三级目录+文件", "123/456/789/789.txt", "123/456/789"},
		{"单文件名无目录段", "789.txt", ""},
		{"空路径", "", ""},
		{"尾斜杠末段为空", "a/b/", "a/b"},     // 防御:webkitRelativePath 实际不带尾斜杠
		{"前导斜杠防御", "/a/b.txt", "/a"}, // 防御:webkitRelativePath 实际不带前导斜杠
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := relPathDirPart(c.in); got != c.want {
				t.Errorf("relPathDirPart(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
