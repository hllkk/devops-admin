package gateway

import (
	"reflect"
	"testing"
)

// renameKeyReferences 纯函数单测：模型 modelKey 改名时密钥三处 JSONB 引用的改写。

func TestRenameKeyReferences_AllReferenced(t *testing.T) {
	models := []string{"qwen-max", "gpt-4o", "qwen-plus"}
	budgets := map[string]any{"qwen-max": 12.5, "gpt-4o": 30}
	limits := map[string]any{"qwen-max": map[string]any{"tpm": 100000, "rpm": 60}}

	newModels, nb, nl, changed := renameKeyReferences(models, budgets, limits, "qwen-max", "qwen3-max")

	if !changed {
		t.Fatal("引用了旧 modelKey,应报告变更")
	}
	if want := []string{"qwen3-max", "gpt-4o", "qwen-plus"}; !reflect.DeepEqual(newModels, want) {
		t.Errorf("models 改写错误: %v, 期望 %v", newModels, want)
	}
	if nb["qwen3-max"] != 12.5 || nb["gpt-4o"] != 30 || len(nb) != 2 {
		t.Errorf("model_budgets 键改名应保值: %v", nb)
	}
	lim := nl["qwen3-max"].(map[string]any)
	if lim["tpm"] != 100000 || lim["rpm"] != 60 || len(nl) != 1 {
		t.Errorf("model_limits 键改名应保值: %v", nl)
	}
}

func TestRenameKeyReferences_NotReferencedNoChange(t *testing.T) {
	models := []string{"gpt-4o", "claude-3"}
	budgets := map[string]any{"gpt-4o": 30}
	limits := map[string]any{"gpt-4o": map[string]any{"tpm": 1}}

	newModels, nb, nl, changed := renameKeyReferences(models, budgets, limits, "qwen-max", "qwen3-max")

	if changed {
		t.Error("未引用旧 modelKey,不应报告变更")
	}
	if want := []string{"gpt-4o", "claude-3"}; !reflect.DeepEqual(newModels, want) {
		t.Errorf("models 不应变动: %v, 期望 %v", newModels, want)
	}
	if _, ok := nb["qwen3-max"]; ok {
		t.Errorf("model_budgets 不应新增键: %v", nb)
	}
	if _, ok := nl["qwen3-max"]; ok {
		t.Errorf("model_limits 不应新增键: %v", nl)
	}
}

func TestRenameKeyReferences_PartialReference(t *testing.T) {
	// 仅 models 引用(主 Key 默认授权场景)：budgets/limits 无该键,不应产生新 map
	models := []string{"qwen-max"}
	newModels, nb, nl, changed := renameKeyReferences(models, nil, nil, "qwen-max", "qwen3-max")

	if !changed {
		t.Fatal("models 引用了旧 modelKey,应报告变更")
	}
	if want := []string{"qwen3-max"}; !reflect.DeepEqual(newModels, want) {
		t.Errorf("models 改写错误: %v, 期望 %v", newModels, want)
	}
	if nb != nil || nl != nil {
		t.Errorf("nil map 应原样返回: budgets=%v limits=%v", nb, nl)
	}
}

func TestRenameKeyReferences_MapOnlyReference(t *testing.T) {
	// 仅按模型预算/限流引用(models 未授权,管理员配过预算后移除授权的残留场景)
	models := []string{"gpt-4o"}
	budgets := map[string]any{"qwen-max": 5}
	limits := map[string]any{"qwen-max": map[string]any{"tpm": 2}}

	newModels, nb, nl, changed := renameKeyReferences(models, budgets, limits, "qwen-max", "qwen3-max")

	if !changed {
		t.Fatal("budgets/limits 引用了旧 modelKey,应报告变更")
	}
	if want := []string{"gpt-4o"}; !reflect.DeepEqual(newModels, want) {
		t.Errorf("models 不应变动: %v, 期望 %v", newModels, want)
	}
	if _, ok := nb["qwen3-max"]; !ok {
		t.Errorf("model_budgets 键应改名: %v", nb)
	}
	if _, ok := nl["qwen3-max"]; !ok {
		t.Errorf("model_limits 键应改名: %v", nl)
	}
}
