/**
 * Namespace Api
 *
 * backend api module: "gateway"（AI 网关）
 */
declare namespace Api {
  /**
   * namespace Gateway
   *
   * backend api module: "gateway"
   */
  namespace Gateway {
    /** 供应商计费类型 */
    type BillingType = 'token' | 'per_call' | 'monthly_quota';

    /** 供应商（管理元数据，不同步 LiteLLM；其下凭证才同步） */
    type Provider = Common.CommonRecord<{
      /** 供应商ID */
      providerId: CommonType.IdType;
      /** 供应商名称 */
      name: string;
      /** 供应商类型(openai/anthropic/vllm...) */
      providerType: string;
      /** 计费类型 */
      billingType: BillingType;
      /** 月预算(USD,null=不限) */
      monthlyBudget: number | null;
      /** 月已用(USD,用量统计定时回填) */
      monthlyUsed: number;
      /** 是否启用 */
      isActive: boolean;
      /** 描述 */
      description: string;
    }>;

    /** 供应商搜索参数(name 模糊;providerType/billingType/isActive 精确) */
    type ProviderSearchParams = CommonType.RecordNullable<
      Pick<Api.Gateway.Provider, 'name' | 'providerType' | 'billingType' | 'isActive'> & Api.Common.CommonSearchParams
    >;

    /** 供应商新增/修改参数(create 时 providerId 为空;update 必填。isActive/ monthlyBudget 为 null 表示不改) */
    type ProviderOperateParams = CommonType.RecordNullable<
      Pick<Api.Gateway.Provider, 'providerId' | 'name' | 'providerType' | 'billingType' | 'monthlyBudget' | 'isActive' | 'description'>
    >;

    /** 供应商列表 */
    type ProviderList = Api.Common.PaginatingQueryRecord<Provider>;

    /** 密钥类型 */
    type KeyType = 'personal_main' | 'personal_scene' | 'dept_main' | 'dept_scene';
    /** 归属类型 */
    type OwnerType = 'user' | 'dept';
    /** 限流模式 */
    type RateLimitMode = 'none' | 'total' | 'per_model';
    /** 预算周期 */
    type BudgetDuration = '1d' | '7d' | '30d';

    /** AI 密钥(LiteLLM 虚拟 Key 的管理面投影；keyValue 不出网，仅 identity/my 返回主 Key 明文) */
    type AiKey = Common.CommonRecord<{
      aiKeyId: CommonType.IdType;
      name: string;
      description: string;
      keyType: KeyType;
      ownerType: OwnerType;
      ownerId: CommonType.IdType;
      keyPrefix: string;
      litellmKeyId: string;
      litellmKeyAlias: string;
      models: string[];
      modelBudgets: Record<string, number>;
      mcps: number[];
      skills: number[];
      budgetLimit: number | null;
      budgetUsed: number;
      budgetHardLimit: boolean;
      budgetDuration: BudgetDuration;
      rateLimitMode: RateLimitMode;
      tpmLimit: number | null;
      rpmLimit: number | null;
      modelLimits: Record<string, { tpm?: number; rpm?: number }>;
      isActive: boolean;
    }>;

    /** 可授权模型(发布+激活，含 anthropic 变体标注) */
    type AvailableModel = {
      modelId: CommonType.IdType;
      modelKey: string;
      modelKeyAnthropic: string;
      name: string;
      category: string;
      requiresApproval: boolean;
      hasAnthropicDeployment: boolean;
    };

    /** 我的 AI 身份(home 契约：主 Key 明文 + 场景 Key 列表 + 可用模型) */
    type MyIdentity = {
      keyValue: string;
      isActive: boolean;
      budgetLimit: number | null;
      budgetHardLimit: boolean;
      budgetDuration: BudgetDuration;
      models: string[];
      modelBudgets: Record<string, number>;
      rateLimitMode: RateLimitMode;
      tpmLimit: number | null;
      rpmLimit: number | null;
      sceneKeys: AiKey[];
      availableModels: AvailableModel[];
    };

    /** 看板总览(按时间范围汇总) */
    type DashboardOverview = {
      totalRequests: number;
      totalCost: number;
      internalCost: number;
      totalTokens: number;
      inputTokens: number;
      outputTokens: number;
      cacheReadTokens: number;
      budgetUsedTotal: number;
      budgetLimitTotal: number;
    };

    /** 成本趋势(按日) */
    type TrendItem = {
      date: string;
      cost: number;
      requests: number;
    };

    /** 成本排行项(按维度 user/model/aiKey) */
    type TopItem = {
      name: string;
      cost: number;
      requests: number;
    };

    /** 预算执行项(按 Key) */
    type BudgetItem = {
      aiKeyId: CommonType.IdType;
      name: string;
      ownerName: string;
      budgetLimit: number;
      budgetUsed: number;
      usageRate: number;
      hardLimit: boolean;
      isActive: boolean;
    };
  }
}
