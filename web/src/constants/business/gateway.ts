/**
 * AI 网关共享选项常量
 *
 * providerType 为自由文本(后端无枚举约束)，下列为常用项，NSelect filterable+tag 允许输入其他；
 * 其余枚举项与后端 model/gateway 的常量严格对齐。
 * as const 让 label 成为字面量联合类型，便于 $t(o.label) 通过 I18nKey 校验。
 */

/** 供应商类型选项(英文专有名词，label=value 风格，不走 i18n) */
export const PROVIDER_TYPE_OPTIONS = [
  { label: 'OpenAI', value: 'openai' },
  { label: 'Anthropic', value: 'anthropic' },
  { label: 'Azure', value: 'azure' },
  { label: 'DeepSeek', value: 'deepseek' },
  { label: 'Qwen', value: 'qwen' },
  { label: 'Doubao', value: 'doubao' },
  { label: 'Zhipu', value: 'zhipu' },
  { label: 'Volcengine', value: 'volcengine' },
  { label: 'xAI', value: 'xai' },
  { label: 'Mistral', value: 'mistral' },
  { label: 'Cohere', value: 'cohere' },
  { label: 'vLLM', value: 'vllm' },
  { label: 'Ollama', value: 'ollama' },
  { label: '百炼', value: 'bailian' }
];

/** 计费类型选项(label 为 i18n key) */
export const BILLING_TYPE_OPTIONS = [
  { label: 'page.gateway.common.billingTypeToken', value: 'token' },
  { label: 'page.gateway.common.billingTypePerCall', value: 'per_call' },
  { label: 'page.gateway.common.billingTypeMonthlyQuota', value: 'monthly_quota' }
] as const;

/** 凭证协议格式选项 */
export const CREDENTIAL_FORMAT_OPTIONS = [
  { label: 'page.gateway.common.formatOpenai', value: 'openai' },
  { label: 'page.gateway.common.formatAnthropic', value: 'anthropic' }
] as const;

/** 模型类别选项(label 为 i18n key) */
export const MODEL_CATEGORY_OPTIONS = [
  { label: 'page.gateway.common.categoryChat', value: 'chat' },
  { label: 'page.gateway.common.categoryEmbedding', value: 'embedding' },
  { label: 'page.gateway.common.categoryRerank', value: 'rerank' }
] as const;

/** 启停状态选项(value 用 1/0，NSelect 不支持 boolean，由搜索组件 computed 转 boolean) */
export const ACTIVE_OPTIONS = [
  { label: 'page.gateway.common.active', value: 1 },
  { label: 'page.gateway.common.inactive', value: 0 }
] as const;

/** AI 密钥类型选项 */
export const KEY_TYPE_OPTIONS = [
  { label: 'page.gateway.common.keyPersonalMain', value: 'personal_main' },
  { label: 'page.gateway.common.keyPersonalScene', value: 'personal_scene' },
  { label: 'page.gateway.common.keyDeptMain', value: 'dept_main' },
  { label: 'page.gateway.common.keyDeptScene', value: 'dept_scene' }
] as const;

/** 归属类型选项 */
export const OWNER_TYPE_OPTIONS = [
  { label: 'page.gateway.common.ownerUser', value: 'user' },
  { label: 'page.gateway.common.ownerDept', value: 'dept' }
] as const;

/** 限流模式选项 */
export const RATE_LIMIT_MODE_OPTIONS = [
  { label: 'page.gateway.common.rateLimitNone', value: 'none' },
  { label: 'page.gateway.common.rateLimitTotal', value: 'total' },
  { label: 'page.gateway.common.rateLimitPerModel', value: 'per_model' }
] as const;

/** 预算周期选项 */
export const BUDGET_DURATION_OPTIONS = [
  { label: 'page.gateway.common.duration1d', value: '1d' },
  { label: 'page.gateway.common.duration7d', value: '7d' },
  { label: 'page.gateway.common.duration30d', value: '30d' }
] as const;

/** 看板成本 Top 维度选项 */
export const TOP_DIMENSION_OPTIONS = [
  { label: 'page.gateway.common.dimUser', value: 'user' },
  { label: 'page.gateway.common.dimModel', value: 'model' },
  { label: 'page.gateway.common.dimAiKey', value: 'aiKey' }
] as const;

/** 看板时间快捷选项 */
export const DASHBOARD_RANGE_OPTIONS = [
  { label: 'page.gateway.common.rangeToday', value: 'today' },
  { label: 'page.gateway.common.range7d', value: '7d' },
  { label: 'page.gateway.common.rangeThisMonth', value: 'thisMonth' },
  { label: 'page.gateway.common.range30d', value: '30d' },
  { label: 'page.gateway.common.rangeLastMonth', value: 'lastMonth' }
] as const;
