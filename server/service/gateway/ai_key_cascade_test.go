package gateway

import (
	"reflect"
	"testing"

	"github.com/hllkk/devops-admin/server/model/gateway"
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

// filterCascadeKeys 纯函数单测：被动停用标记区分级联停与手动停。

func TestFilterCascadeKeys_DisableOnlyActive(t *testing.T) {
	// 级联停用：只动启用中的 Key，管理员手动停的(无标记)不动
	keys := []gateway.AiKey{
		{AiKeyId: 1, IsActive: true},                          // 启用中 → 停用+打标
		{AiKeyId: 2, IsActive: false},                         // 手动停 → 不动
		{AiKeyId: 3, IsActive: true, DisabledByCascade: true}, // 已被动停(此前一轮) → 不重复动
	}
	got := filterCascadeKeys(keys, false)
	if len(got) != 2 || got[0].AiKeyId != 1 || got[1].AiKeyId != 3 {
		t.Errorf("停用应选中启用中的 Key 1/3, got %v", idsOf(got))
	}
}

func TestFilterCascadeKeys_EnableOnlyCascaded(t *testing.T) {
	// 级联恢复：只恢复带被动标记的 Key，手动停/超限停的(无标记)不动
	keys := []gateway.AiKey{
		{AiKeyId: 1, IsActive: false, DisabledByCascade: true},  // 被动停 → 恢复+清标
		{AiKeyId: 2, IsActive: false},                           // 管理员手动停 → 不恢复
		{AiKeyId: 3, IsActive: false, DisabledByCascade: false}, // 超限停(不打标) → 不恢复
		{AiKeyId: 4, IsActive: true},                            // 本就启用 → 不动
	}
	got := filterCascadeKeys(keys, true)
	if len(got) != 1 || got[0].AiKeyId != 1 {
		t.Errorf("恢复应仅选中被动停的 Key 1, got %v", idsOf(got))
	}
}

func TestFilterCascadeKeys_Empty(t *testing.T) {
	if got := filterCascadeKeys(nil, true); len(got) != 0 {
		t.Errorf("空列表应返回空, got %v", got)
	}
	if got := filterCascadeKeys(nil, false); len(got) != 0 {
		t.Errorf("空列表应返回空, got %v", got)
	}
}

func idsOf(keys []gateway.AiKey) []int64 {
	ids := make([]int64, 0, len(keys))
	for i := range keys {
		ids = append(ids, keys[i].AiKeyId)
	}
	return ids
}
