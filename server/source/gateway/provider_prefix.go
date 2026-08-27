package gateway

import (
	"context"

	gatewayModel "github.com/hllkk/devops-admin/server/model/gateway"
	"github.com/hllkk/devops-admin/server/service/system"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 供应商前缀差异表无跨初始化器依赖；排到 system 初始化链之后的独立区段
// （system 链内以 +1 递增，避开 +100/+200 段以防后续扩展撞号）。
const initOrderGatewayPrefix = system.InitOrderSystem + 200

type initProviderPrefix struct{}

// auto run
func init() {
	system.RegisterInit(initOrderGatewayPrefix, &initProviderPrefix{})
}

func (i *initProviderPrefix) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}
	return ctx, db.AutoMigrate(
		&gatewayModel.ProviderPrefix{},
		&gatewayModel.Model{},
		&gatewayModel.AiKey{},
		&gatewayModel.KeyScenario{},
		&gatewayModel.CostSummaryDaily{},
		&gatewayModel.Provider{},
		&gatewayModel.RouterSettings{},
		&gatewayModel.Credential{},
		&gatewayModel.ModelDeployment{},
	)
}

func (i *initProviderPrefix) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	return db.Migrator().HasTable(&gatewayModel.ProviderPrefix{})
}

func (i *initProviderPrefix) InitializerName() string {
	return gatewayModel.ProviderPrefix{}.TableName()
}

func (i *initProviderPrefix) InitializeData(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}
	if err := SeedProviderPrefix(db); err != nil {
		return ctx, err
	}
	if err := EnsureAiKeyUniqueIndex(db); err != nil {
		return ctx, err
	}
	if err := EnsureKeyScenarioUniqueIndex(db); err != nil {
		return ctx, err
	}
	return ctx, nil
}

func (i *initProviderPrefix) DataInserted(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	var count int64
	db.Model(&gatewayModel.ProviderPrefix{}).Count(&count)
	return count > 0
}

// providerPrefixSeeds 供应商路由前缀差异表种子（照抄 AIHelms docker/db/init.sql:281-330，42 行）。
// 消费方是下一 slice 的 ModelDeployment 同步：按 (provider_type, format, category) 查
// prefix（LiteLLM model 前缀化）与 needs_v1（api_base 自动补 /v1），不在代码里 switch。
var providerPrefixSeeds = []gatewayModel.ProviderPrefix{
	// 有专属前缀的供应商
	{ProviderType: "openai", Format: "openai", Category: "chat", Prefix: "openai"},
	{ProviderType: "openai", Format: "openai", Category: "embedding", Prefix: "openai"},
	{ProviderType: "openai", Format: "openai", Category: "image", Prefix: "openai"},
	{ProviderType: "openai", Format: "openai", Category: "audio", Prefix: "openai"},
	{ProviderType: "anthropic", Format: "anthropic", Category: "chat", Prefix: "anthropic"},
	{ProviderType: "azure", Format: "openai", Category: "chat", Prefix: "azure"},
	{ProviderType: "azure", Format: "openai", Category: "embedding", Prefix: "azure"},
	{ProviderType: "google", Format: "openai", Category: "chat", Prefix: "gemini"},
	{ProviderType: "google", Format: "openai", Category: "embedding", Prefix: "gemini"},
	{ProviderType: "deepseek", Format: "openai", Category: "chat", Prefix: "deepseek"},
	{ProviderType: "deepseek", Format: "anthropic", Category: "chat", Prefix: "anthropic"},
	{ProviderType: "bedrock", Format: "openai", Category: "chat", Prefix: "bedrock"},
	{ProviderType: "bedrock", Format: "openai", Category: "embedding", Prefix: "bedrock"},
	{ProviderType: "vertex_ai", Format: "openai", Category: "chat", Prefix: "vertex_ai"},
	{ProviderType: "vertex_ai", Format: "openai", Category: "embedding", Prefix: "vertex_ai"},
	// 兼容多格式的供应商
	{ProviderType: "volcengine", Format: "openai", Category: "chat", Prefix: "openai", NeedsV1: true},
	{ProviderType: "volcengine", Format: "openai", Category: "embedding", Prefix: "openai", NeedsV1: true},
	{ProviderType: "volcengine", Format: "anthropic", Category: "chat", Prefix: "anthropic"},
	{ProviderType: "dashscope", Format: "openai", Category: "chat", Prefix: "openai", NeedsV1: true},
	{ProviderType: "dashscope", Format: "openai", Category: "embedding", Prefix: "openai", NeedsV1: true},
	{ProviderType: "dashscope", Format: "anthropic", Category: "chat", Prefix: "anthropic"},
	{ProviderType: "zhipu", Format: "openai", Category: "chat", Prefix: "openai", NeedsV1: true},
	{ProviderType: "zhipu", Format: "anthropic", Category: "chat", Prefix: "anthropic"},
	{ProviderType: "moonshot", Format: "openai", Category: "chat", Prefix: "openai", NeedsV1: true},
	{ProviderType: "moonshot", Format: "anthropic", Category: "chat", Prefix: "anthropic"},
	{ProviderType: "minimax", Format: "openai", Category: "chat", Prefix: "openai", NeedsV1: true},
	{ProviderType: "minimax", Format: "anthropic", Category: "chat", Prefix: "anthropic"},
	// 自部署
	{ProviderType: "vllm", Format: "openai", Category: "chat", Prefix: "hosted_vllm", NeedsV1: true},
	{ProviderType: "vllm", Format: "openai", Category: "embedding", Prefix: "openai", NeedsV1: true},
	{ProviderType: "vllm", Format: "openai", Category: "rerank", Prefix: "hosted_vllm", NeedsV1: true},
	{ProviderType: "sglang", Format: "openai", Category: "chat", Prefix: "hosted_vllm", NeedsV1: true},
	{ProviderType: "sglang", Format: "openai", Category: "embedding", Prefix: "openai", NeedsV1: true},
	{ProviderType: "ollama", Format: "ollama", Category: "chat", Prefix: "ollama"},
	{ProviderType: "ollama", Format: "ollama", Category: "embedding", Prefix: "ollama"},
	{ProviderType: "lmstudio", Format: "openai", Category: "chat", Prefix: "openai", NeedsV1: true},
	// LM Studio 接入格式(OpenAI 兼容,默认端口1234):AIHelms 此格式查表 miss 靠 custom_llm_provider
	// 兜底且不补 /v1(用户须手填带 /v1 的 base,易 404)；此处直接落表行走统一管线,自动补 /v1
	{ProviderType: "lmstudio", Format: "lmstudio", Category: "chat", Prefix: "openai", NeedsV1: true},
	{ProviderType: "lmstudio", Format: "lmstudio", Category: "embedding", Prefix: "openai", NeedsV1: true},
	// 小米 MiMo / 腾讯混元 / xAI
	{ProviderType: "xiaomi_mimo", Format: "openai", Category: "chat", Prefix: "xiaomi_mimo"},
	{ProviderType: "xiaomi_mimo", Format: "anthropic", Category: "chat", Prefix: "anthropic"},
	{ProviderType: "tencent", Format: "openai", Category: "chat", Prefix: "tencent", NeedsV1: true},
	{ProviderType: "xai", Format: "openai", Category: "chat", Prefix: "xai"},
	// 其他
	{ProviderType: "other", Format: "openai", Category: "chat", Prefix: "openai", NeedsV1: true},
	{ProviderType: "other", Format: "openai", Category: "embedding", Prefix: "openai", NeedsV1: true},
	{ProviderType: "other", Format: "anthropic", Category: "chat", Prefix: "anthropic"},
}

// SeedProviderPrefix 幂等写入供应商前缀种子（复合 uniqueIndex + OnConflict DoNothing 自愈）。
// 导出供两路调用：/initdb 初始化器链（InitializeData）与重启路径 RegisterTables 末尾兜底
// （已有环境自愈拿到种子——这是部署路由的功能依赖数据而非展示数据，缺失则 deployment slice 不可用）。
func SeedProviderPrefix(db *gorm.DB) error {
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&providerPrefixSeeds).Error; err != nil {
		return errors.Wrap(err, gatewayModel.ProviderPrefix{}.TableName()+"供应商前缀种子初始化失败!")
	}
	return nil
}

// EnsureAiKeyUniqueIndex 创建 AiKey 同类归属下名称唯一的部分唯一索引
// (owner_type, owner_id, key_type, name) WHERE deleted_at IS NULL。
// 防同 owner 同 key_type 下 name 重复导致 LiteLLM key_alias 撞车(alias={ownerType}:{ownerId}/{name})；
// 软删记录不占约束(删后可重建同名)。gorm tag 建不出部分索引(含 deleted_at 的普通唯一索引在
// Postgres 下 NULL 不参与唯一比较、防不住未删重复)，用手写 SQL。
// 导出供两路调用：/initdb 初始化器链(InitializeData)与重启路径 RegisterTables 末尾兜底(同 SeedProviderPrefix 模式)。
func EnsureAiKeyUniqueIndex(db *gorm.DB) error {
	return db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_aikey_owner_name ON gateway_ai_key (owner_type, owner_id, key_type, name) WHERE deleted_at IS NULL`).Error
}

// EnsureKeyScenarioUniqueIndex 创建 KeyScenario 名称唯一的部分唯一索引(name) WHERE deleted_at IS NULL。
// 停用行占名(is_active=false 仍在列表可见，防同名二义)，软删行不占名(删后可重建同名)。
// 规避 AIHelms 的坑：其 key_scenarios 的 DB unique 与"删除=置 is_active=false"并存，软删后同名永远建不回。
// 导出供两路调用：/initdb 初始化器链(InitializeData)与重启路径 RegisterTables 末尾兜底(同上模式)。
func EnsureKeyScenarioUniqueIndex(db *gorm.DB) error {
	return db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_keyscenario_name ON gateway_key_scenario (name) WHERE deleted_at IS NULL`).Error
}
