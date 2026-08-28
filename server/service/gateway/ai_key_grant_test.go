package gateway

import (
	"reflect"
	"testing"
)

// removeModelKey 纯函数单测：主 Key 授权回收时的 models 列表移除。

func TestRemoveModelKey_Present(t *testing.T) {
	models := []string{"qwen-max", "gpt-4o", "qwen-plus"}
	out, changed := removeModelKey(models, "gpt-4o")

	if !changed {
		t.Fatal("列表含目标 modelKey,应报告变更")
	}
	if want := []string{"qwen-max", "qwen-plus"}; !reflect.DeepEqual(out, want) {
		t.Errorf("移除错误: %v, 期望 %v", out, want)
	}
}

func TestRemoveModelKey_AbsentNoChange(t *testing.T) {
	models := []string{"qwen-max", "qwen-plus"}
	out, changed := removeModelKey(models, "gpt-4o")

	if changed {
		t.Error("列表不含目标 modelKey,不应报告变更")
	}
	if !reflect.DeepEqual(out, models) {
		t.Errorf("列表不应变动: %v", out)
	}
}

func TestRemoveModelKey_EmptyList(t *testing.T) {
	out, changed := removeModelKey([]string{}, "gpt-4o")

	if changed {
		t.Error("空列表不应报告变更")
	}
	if len(out) != 0 {
		t.Errorf("空列表应保持为空: %v", out)
	}
}
