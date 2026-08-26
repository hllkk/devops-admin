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
  { label: 'Google', value: 'google' },
  { label: 'DeepSeek', value: 'deepseek' },
  { label: 'AWS Bedrock', value: 'bedrock' },
  { label: 'Vertex AI', value: 'vertex_ai' },
  { label: '火山引擎', value: 'volcengine' },
  { label: '百炼', value: 'bailian' },
  { label: '智谱', value: 'zhipu' },
  { label: 'Moonshot', value: 'moonshot' },
  { label: 'Minimax', value: 'minimax' },
  { label: '小米MiMo', value: 'xiaomi_mimo' },
  { label: '腾讯混元', value: 'tencent' },
  { label: 'xAI', value: 'xai' },
  { label: 'vLLM', value: 'vllm' },
  { label: 'SGLang', value: 'sglang' },
  { label: 'Ollama', value: 'ollama' },
  { label: '其他', value: 'other' }
];

/**
 * 供应商类型 → 本地图标名(src/assets/svg-icon/provider/ 下的文件，经 sprite 以 localIcon 引用)。
 * 仅覆盖有对应品牌图的类型；qwen 与 bailian 同属阿里云百炼(DashScope)平台故共用图；
 * 自由文本类型未命中时兜底 custom。
 */
const PROVIDER_ICON_MAP: Record<string, string> = {
  openai: 'provider-openai',
  anthropic: 'provider-anthropic',
  azure: 'provider-azure',
  deepseek: 'provider-deepseek',
  google: 'provider-google',
  bedrock: 'provider-bedrock',
  vertex_ai: 'provider-vertex_ai',
  dashscope: 'provider-dashscope',
  qwen: 'provider-dashscope',
  bailian: 'provider-dashscope',
  zhipu: 'provider-zhipu',
  moonshot: 'provider-moonshot',
  minimax: 'provider-minimax',
  xiaomi_mimo: 'provider-xiaomi_mimo',
  tencent: 'provider-tencent',
  volcengine: 'provider-volcengine',
  xai: 'provider-xai',
  vllm: 'provider-vllm',
  sglang: 'provider-sglang',
  ollama: 'provider-ollama',
  lmstudio: 'provider-lmstudio',
  openai_compatible: 'provider-openai_compatible'
};

/** 取供应商类型对应的本地图标名，未收录类型兜底 custom */
export function getProviderIcon(providerType?: string): string {
  return (providerType && PROVIDER_ICON_MAP[providerType]) || 'provider-custom';
}

/** 计费类型选项(label 为 i18n key) */
export const BILLING_TYPE_OPTIONS = [
  { label: 'page.gateway.common.billingTypeToken', value: 'token' },
  { label: 'page.gateway.common.billingTypePerCall', value: 'per_call' },
  { label: 'page.gateway.common.billingTypeMonthlyQuota', value: 'monthly_quota' }
] as const;

/** 凭证协议格式选项 */
export const CREDENTIAL_FORMAT_OPTIONS = [
  { label: 'page.gateway.common.formatOpenai', value: 'openai' },
  { label: 'page.gateway.common.formatAnthropic', value: 'anthropic' },
  { label: 'page.gateway.common.formatLmstudio', value: 'lmstudio' },
  { label: 'page.gateway.common.formatOllama', value: 'ollama' }
] as const;

/** 各接入格式的默认 API Base 提示(本地推理服务地址，用于表单 placeholder) */
export const FORMAT_API_BASE_PLACEHOLDER: Record<string, string> = {
  lmstudio: 'http://127.0.0.1:1234',
  ollama: 'http://127.0.0.1:11434'
};

/** Ollama 本地推理无鉴权，这些格式不需要 API Key(前端隐藏输入框) */
export const FORMAT_NEEDS_NO_KEY = new Set(['ollama']);

/** 模型类别选项(label 为 i18n key) */
export const MODEL_CATEGORY_OPTIONS = [
  { label: 'page.gateway.common.categoryChat', value: 'chat' },
  { label: 'page.gateway.common.categoryEmbedding', value: 'embedding' },
  { label: 'page.gateway.common.categoryRerank', value: 'rerank' }
] as const;

/**
 * 各类别的能力标签预设(值为展示文本即存储值，与供应商类型选项同风格不走 i18n)。
 * 表单中作为下拉打底选项，仍可 tag 输入自定义；切换类别时清空重选。
 */
export const MODEL_CAPABILITY_PRESETS: Record<string, string[]> = {
  chat: ['图像', '推理', '工具调用', '长上下文'],
  embedding: ['多语言', '多模态', '代码', '长文本'],
  rerank: ['多语言', '多模态']
};

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
