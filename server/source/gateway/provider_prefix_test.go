package gateway

import (
	"testing"

	gatewayModel "github.com/hllkk/devops-admin/server/model/gateway"
)

// 前端供应商预置选项与前缀差异表种子的口径一致性：每个预置类型在种子里必须有
// chat 行，否则用该类型建的部署推送裸名会被 LiteLLM 静默 Dropping
// (2026-10 生产 bailian 缺行事故的回归防线)。自由文本类型不在预置集合，由
// pushDeployment 的裸名拦截兜底。
func TestProviderOptionTypesHavePrefixSeed(t *testing.T) {
	seeded := map[string]bool{}
	for _, row := range providerPrefixSeeds {
		seeded[row.ProviderType+"/"+row.Format+"/"+row.Category] = true
	}
	for _, pt := range gatewayModel.ProviderOptionTypes {
		if !seeded[pt+"/openai/chat"] && !seeded[pt+"/anthropic/chat"] {
			t.Errorf("供应商类型 %q 缺前缀差异表种子行(openai/chat 或 anthropic/chat)，"+
				"该类型部署将推送裸模型名被 LiteLLM 拒绝；请补 providerPrefixSeeds 或前端选项", pt)
		}
	}
}

// 种子行 (provider_type, format, category) 唯一，重复行会被 OnConflict DoNothing 静默吞掉一调。
func TestProviderPrefixSeedsUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, row := range providerPrefixSeeds {
		key := row.ProviderType + "/" + row.Format + "/" + row.Category
		if seen[key] {
			t.Errorf("前缀差异表种子存在重复行: %s", key)
		}
		seen[key] = true
	}
}
