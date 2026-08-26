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
      /** 支持的接入格式(凭证 format 从中选一) */
      supportedFormats: string[] | null;
      /** 凭证数(列表展示,后端填充) */
      credentialCount: number;
    }>;

    /** 供应商搜索参数(name 模糊;providerType/billingType/isActive 精确) */
    type ProviderSearchParams = CommonType.RecordNullable<
      Pick<Api.Gateway.Provider, 'name' | 'providerType' | 'billingType' | 'isActive'> & Api.Common.CommonSearchParams
    >;

    /** 供应商新增/修改参数(create 时 providerId 为空;update 必填。isActive/ monthlyBudget 为 null 表示不改) */
    type ProviderOperateParams = CommonType.RecordNullable<
      Pick<Api.Gateway.Provider, 'providerId' | 'name' | 'providerType' | 'billingType' | 'monthlyBudget' | 'isActive' | 'description' | 'supportedFormats'>
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

    /** 我的 AI 身份(home 契约：主 Key 明文 + 场景 Key 列表 + 可用模型；管理员创建制，未开通 opened=false) */
    type MyIdentity = {
      opened: boolean;
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

    // ───────────────── 凭证 Credential ─────────────────

    /** 凭证协议格式(credential_info.format 取值) */
    type CredentialFormat = 'openai' | 'anthropic';

    /** 凭证(管理面出网视图：credentialValues 为解密后掩码值，api_base 等非敏感值明文) */
    type Credential = Common.CommonRecord<{
      credentialId: CommonType.IdType;
      credentialName: string;
      providerId: CommonType.IdType;
      /** 凭证键值(敏感值已掩码,如 sk-ab****cdef;提交时掩码回传=不改,新值=覆盖) */
      credentialValues: Record<string, string>;
      /** 凭证元数据(format 等非敏感信息) */
      credentialInfo: Record<string, any>;
      litellmSynced: boolean;
      isActive: boolean;
      description: string;
    }>;

    /** 凭证搜索参数(credentialName 模糊;providerId/isActive/litellmSynced 精确) */
    type CredentialSearchParams = CommonType.RecordNullable<
      Pick<Credential, 'credentialName' | 'providerId' | 'isActive' | 'litellmSynced'> & Api.Common.CommonSearchParams
    >;

    /** 凭证新增/修改参数(credentialName 创建后不可改;credentialValues 明文或掩码回传) */
    type CredentialOperateParams = CommonType.RecordNullable<
      Pick<
        Credential,
        'credentialId' | 'credentialName' | 'providerId' | 'credentialValues' | 'credentialInfo' | 'isActive' | 'description'
      >
    >;

    type CredentialList = Api.Common.PaginatingQueryRecord<Credential>;

    /** 供应商凭证表单字段定义(透传 LiteLLM /public/providers/fields，结构宽松,动态表单按实返回渲染) */
    type ProviderField = Record<string, any>;

    /** 凭证重同步 LiteLLM 结果汇总 */
    type ResyncResult = {
      total: number;
      pushed: number;
      skipped: number;
      failed: string[];
    };

    // ───────────────── 模型 Model ─────────────────

    /** 模型类别 */
    type ModelCategory = 'chat' | 'embedding' | 'rerank';

    /** 模型(列表行：能力标签展开 + 部署计数) */
    type Model = Common.CommonRecord<{
      modelId: CommonType.IdType;
      modelKey: string;
      name: string;
      category: string;
      capabilities: string[];
      description: string;
      logoProviderType: string;
      isActive: boolean;
      isPublished: boolean;
      visibilityType: 'all' | 'selected';
      requiresApproval: boolean;
      deploymentCount: number;
      activeDeploymentCount: number;
    }>;

    /** 模型搜索参数(name/modelKey 模糊;category/isActive/isPublished 精确) */
    type ModelSearchParams = CommonType.RecordNullable<
      Pick<Model, 'name' | 'modelKey' | 'category' | 'isActive' | 'isPublished'> & Api.Common.CommonSearchParams
    >;

    /** 模型新增/修改参数(update 允许改 modelKey,触发关联部署路由名级联重建) */
    type ModelOperateParams = CommonType.RecordNullable<
      Pick<Model, 'modelId' | 'modelKey' | 'name' | 'category' | 'logoProviderType' | 'description' | 'capabilities'>
    >;

    type ModelList = Api.Common.PaginatingQueryRecord<Model>;

    // ───────────────── 部署 Deployment ─────────────────

    /** 部署(出网视图：含关联上下文 + 掩码后的路由参数) */
    type Deployment = Common.CommonRecord<{
      deploymentId: CommonType.IdType;
      modelId: CommonType.IdType;
      credentialId: CommonType.IdType;
      litellmModelId: string;
      litellmParams: Record<string, any>;
      modelInfo: Record<string, any>;
      deployName: string;
      billingType: BillingType;
      costPerCall: number | null;
      monthlyCallQuota: number | null;
      monthlyCallUsed: number;
      isActive: boolean;
      /** 关联模型路由名 */
      modelKey: string;
      /** 关联凭证名(无关联为空) */
      credentialName: string;
      /** 凭证协议格式(openai/anthropic) */
      format: string;
      /** 关联供应商类型 */
      providerType: string;
      /** 当前路由名(三态命名,routable 版) */
      routeName: string;
    }>;

    /** 部署搜索参数(modelId/credentialId 精确;keyword 模糊) */
    type DeploymentSearchParams = CommonType.RecordNullable<
      {
        modelId: CommonType.IdType;
        credentialId: CommonType.IdType;
        keyword: string;
        isActive: boolean;
      } & Api.Common.CommonSearchParams
    >;

    /** 部署新增/修改参数(litellmParams 必含 model 键;credentialId 非 0 绑定平台凭证) */
    type DeploymentOperateParams = CommonType.RecordNullable<
      Pick<
        Deployment,
        'deploymentId' | 'modelId' | 'credentialId' | 'deployName' | 'litellmParams' | 'modelInfo' | 'billingType' | 'costPerCall' | 'monthlyCallQuota' | 'isActive'
      >
    >;

    type DeploymentList = Api.Common.PaginatingQueryRecord<Deployment>;

    // ───────────────── AI 密钥 AiKey(搜索/操作参数) ─────────────────

    /** AI 密钥搜索参数(keyType/ownerType/ownerId/isActive 精确;name 模糊) */
    type AiKeySearchParams = CommonType.RecordNullable<
      Pick<AiKey, 'keyType' | 'ownerType' | 'ownerId' | 'name' | 'isActive'> & Api.Common.CommonSearchParams
    >;

    /** AI 密钥新增/修改参数(keyType/ownerType/ownerId 创建必填且不可改) */
    type AiKeyOperateParams = CommonType.RecordNullable<
      Pick<
        AiKey,
        | 'aiKeyId'
        | 'keyType'
        | 'ownerType'
        | 'ownerId'
        | 'name'
        | 'description'
        | 'models'
        | 'modelBudgets'
        | 'budgetLimit'
        | 'budgetHardLimit'
        | 'budgetDuration'
        | 'rateLimitMode'
        | 'tpmLimit'
        | 'rpmLimit'
        | 'modelLimits'
        | 'isActive'
      >
    >;
  }
}
