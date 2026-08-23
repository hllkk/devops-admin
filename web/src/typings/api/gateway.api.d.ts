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
  }
}
