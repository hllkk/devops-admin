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

    /** 供应商（管理元数据，不同步 LiteLLM；其下凭证才同步。计费/预算口径在部署级，不在供应商级） */
    type Provider = Common.CommonRecord<{
      /** 供应商ID */
      providerId: CommonType.IdType;
      /** 供应商名称 */
      name: string;
      /** 供应商类型(openai/anthropic/vllm...) */
      providerType: string;
      /** 是否启用 */
      isActive: boolean;
      /** 描述 */
      description: string;
      /** 支持的接入格式(凭证 format 从中选一) */
      supportedFormats: string[] | null;
      /** 凭证数(列表展示,后端填充) */
      credentialCount: number;
    }>;

    /** 供应商搜索参数(name 模糊;providerType/isActive 精确) */
    type ProviderSearchParams = CommonType.RecordNullable<
      Pick<Api.Gateway.Provider, 'name' | 'providerType' | 'isActive'> & Api.Common.CommonSearchParams
    >;

    /** 供应商新增/修改参数(create 时 providerId 为空;update 必填。isActive 为 null 表示不改) */
    type ProviderOperateParams = CommonType.RecordNullable<
      Pick<Api.Gateway.Provider, 'providerId' | 'name' | 'providerType' | 'isActive' | 'description' | 'supportedFormats'>
    >;

    /** 供应商列表 */
    type ProviderList = Api.Common.PaginatingQueryRecord<Provider>;

    /** 余量条目类型 */
    type BalanceItemType = 'seat' | 'shared_package';

    /** 套餐余量条目(厂商侧快照行,旁路只读,与网关标价成本口径互不并) */
    type ProviderBalance = Common.CommonRecord<{
      /** 余量ID */
      balanceId: CommonType.IdType;
      /** 供应商ID */
      providerId: CommonType.IdType;
      /** 套餐类型(token_plan) */
      planType: string;
      /** 条目类型(seat=坐席/shared_package=共享包) */
      itemType: BalanceItemType;
      /** 条目键(坐席SeatId/共享包InstanceCode) */
      itemKey: string;
      /** 条目名称(坐席成员名/共享包说明) */
      itemName: string;
      /** 坐席档位(standard/pro/max) */
      specType: string;
      /** 条目状态(NORMAL/LIMIT/...) */
      status: string;
      /** 权益类型(CREDITS) */
      equityType: string;
      /** 当前计费周期开始 */
      cycleStart: string | null;
      /** 当前计费周期结束 */
      cycleEnd: string | null;
      /** 周期总额度(Credits) */
      totalValue: number;
      /** 周期剩余额度(Credits) */
      surplusValue: number;
      /** 周期已用额度(Credits,总-剩余) */
      usedValue: number;
      /** 同步时间(UTC) */
      syncedAt: string;
      /** 厂商原始返回(排障) */
      raw: Record<string, unknown> | null;
    }>;

    /** 套餐余量汇总(供应商面板头 + 看板汇总卡共用,厂商侧口径) */
    type ProviderBalanceSummary = {
      providerId: CommonType.IdType;
      providerName: string;
      /** 套餐标签(如"百炼 Token Plan") */
      planLabel: string;
      totalValue: number;
      usedValue: number;
      surplusValue: number;
      seatCount: number;
      packageCount: number;
      /** 最近同步时间(空=从未同步) */
      syncedAt: string | null;
    };

    /** 供应商余量明细(汇总+条目) */
    type ProviderBalanceDetail = {
      summary: ProviderBalanceSummary;
      items: ProviderBalance[];
    };

    /** 余量采集配置(AK/SK 掩码回显,保存时掩码占位保留旧明文) */
    type BalanceSyncConfig = {
      accessKeyId: string;
      accessKeySecret: string;
      /** 服务地域(默认 cn-beijing) */
      region: string;
    };

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
      ownerName: string;
      ownerUsername: string;
      scenarioId: CommonType.IdType;
      scenarioName: string;
      keyPrefix: string;
      litellmKeyId: string;
      litellmKeyAlias: string;
      models: string[];
      modelBudgets: Record<string, number>;
      /** 已授权 MCP(serverName 列表;P2 起生效) */
      mcps: string[];
      /** 已授权 Skill(skillId 字符串列表;P2 起生效,平台自有资源不经 LiteLLM) */
      skills: string[];
      budgetLimit: number | null;
      budgetUsed: number;
      budgetHardLimit: boolean;
      budgetDuration: BudgetDuration;
      rateLimitMode: RateLimitMode;
      tpmLimit: number | null;
      rpmLimit: number | null;
      modelLimits: Record<string, { tpm?: number; rpm?: number }>;
      isActive: boolean;
      /** 过期时间(RFC3339,null=永不过期;下发 LiteLLM expires_at 原生拦截) */
      expiresAt: string | null;
      /** 最近使用时间(用量回流回填,僵尸 Key 治理) */
      lastUsedAt: string | null;
    }>;

    /** 密钥完整明文(仅 value/:id 按需返回，管理员/超管复制给用户用) */
    type AiKeyReveal = {
      keyValue: string;
    };

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

    /** 我的 AI 身份(home 契约：主 Key 明文 + 场景 Key 列表 + 可见模型；管理员创建制，未开通 opened=false) */
    type MyIdentity = {
      opened: boolean;
      keyValue: string;
      isActive: boolean;
      expiresAt: string | null;
      budgetLimit: number | null;
      budgetHardLimit: boolean;
      budgetDuration: BudgetDuration;
      models: string[];
      modelBudgets: Record<string, number>;
      rateLimitMode: RateLimitMode;
      tpmLimit: number | null;
      rpmLimit: number | null;
      sceneKeys: AiKey[];
      /** 可见模型(按发布可见性过滤:全员/部门/指定用户) */
      availableModels: AvailableModel[];
      /** 已授权 MCP(serverName 列表) */
      mcps: string[];
      /** 可见 MCP(按发布可见性过滤,home MCP 区) */
      availableMcps: AvailableMcp[];
      /** 已授权 Skill(skillId 字符串列表) */
      skills: string[];
      /** 可见 Skill(按发布可见性过滤,home Skill 区) */
      availableSkills: AvailableSkill[];
      /** 网关接入点(litellm public-url,客户端 Base URL) */
      gatewayUrl: string;
    };

    /** 用户侧可见模型(模型广场契约:active+published+按发布可见性过滤,含 anthropic 变体标注) */
    type ActiveModel = {
      modelId: CommonType.IdType;
      modelKey: string;
      modelKeyAnthropic: string;
      name: string;
      category: string;
      description: string;
      logoProviderType: string;
      capabilities: string[];
      requiresApproval: boolean;
      hasAnthropicDeployment: boolean;
    };

    // ───────────────── MCP 服务器 MCPServer(AI 市场 P2) ─────────────────

    /** MCP 传输协议(sse/streamable_http,下发 LiteLLM 时后者映射 http) */
    type MCPTransport = 'sse' | 'streamable_http';

    /** MCP 鉴权方式(none 无凭据;api_key/bearer_token 凭据存 credentials.auth_value 密文) */
    type MCPAuthType = 'none' | 'api_key' | 'bearer_token';

    /** MCP 计费类型(per_call 按次/free 免费;工具级空=继承服务器) */
    type MCPBilling = 'per_call' | 'free' | '';

    /** MCP 健康状态 */
    type MCPHealthStatus = 'unknown' | 'healthy' | 'unhealthy';

    /** 可授权 MCP 服务器(精简版,Key 授权下拉与广场卡片共用) */
    type AvailableMcp = {
      mcpServerId: CommonType.IdType;
      /** 路由名(授权锚点,LiteLLM server_name) */
      serverName: string;
      name: string;
      description: string;
      category: string;
      author: string;
      iconUrl: string;
      documentationUrl: string;
      requiresApproval: boolean;
      toolCount: number;
    };

    /** MCP 服务器(管理面出网视图:credentials 为解密后掩码值,提交时掩码回传=保留旧明文) */
    type MCPServer = Common.CommonRecord<{
      mcpServerId: CommonType.IdType;
      /** 展示名称 */
      name: string;
      /** LiteLLM 路由名(唯一,禁 '-',不可修改) */
      serverName: string;
      /** MCP 端点 URL */
      url: string;
      transport: MCPTransport;
      authType: MCPAuthType;
      /** 鉴权凭据(敏感值已掩码;auth_value 键) */
      credentials: Record<string, string> | null;
      description: string;
      /** 使用说明(接入页展示) */
      instructions: string;
      category: string;
      author: string;
      iconUrl: string;
      documentationUrl: string;
      billingType: MCPBilling;
      /** 单次调用价(¥,null=免费) */
      externalCostPerCall: number | null;
      /** 单次调用内部结算价(¥,null=同外部价) */
      internalCostPerCall: number | null;
      /** 累计调用次数(回流任务按 server 增量维护) */
      callCount: number;
      isActive: boolean;
      isPublished: boolean;
      /** 可见范围(all/selected/user) */
      visibilityType: 'all' | 'selected' | 'user';
      /** 接入需审批 */
      requiresApproval: boolean;
      healthStatus: MCPHealthStatus;
      lastHealthCheck: string | null;
      healthCheckError: string | null;
      /** LiteLLM 侧 server_id(归因锚点) */
      litellmServerId: string;
      litellmSynced: boolean;
      litellmSyncError: string | null;
      /** 工具数(后端填充) */
      toolCount: number;
    }>;

    /** MCP 服务器搜索参数(name 模糊;其余精确) */
    type MCPServerSearchParams = CommonType.RecordNullable<
      Pick<Api.Gateway.MCPServer, 'name' | 'category' | 'isActive' | 'isPublished' | 'healthStatus'> & Api.Common.CommonSearchParams
    >;

    /** MCP 服务器新增/修改参数(create 时 mcpServerId 为空;serverName 创建后不可改) */
    type MCPServerOperateParams = CommonType.RecordNullable<
      Pick<
        Api.Gateway.MCPServer,
        | 'mcpServerId'
        | 'name'
        | 'serverName'
        | 'url'
        | 'transport'
        | 'authType'
        | 'credentials'
        | 'description'
        | 'instructions'
        | 'category'
        | 'author'
        | 'iconUrl'
        | 'documentationUrl'
        | 'billingType'
        | 'externalCostPerCall'
        | 'internalCostPerCall'
        | 'isActive'
      >
    >;

    /** MCP 服务器列表 */
    type MCPServerList = Api.Common.PaginatingQueryRecord<MCPServer>;

    /** MCP 工具(refresh-tools 远端全量重建;namespacedName=serverName_toolName) */
    type MCPTool = Common.CommonRecord<{
      mcpToolId: CommonType.IdType;
      mcpServerId: CommonType.IdType;
      /** 工具原始名 */
      toolName: string;
      /** 网关全名({serverName}_{toolName}) */
      namespacedName: string;
      displayName: string;
      description: string;
      /** 入参 Schema */
      inputSchema: Record<string, unknown> | null;
      /** 工具级计费类型(空=继承服务器) */
      billingType: MCPBilling;
      /** 工具级单次调用价(¥,null=继承) */
      externalCostPerCall: number | null;
      /** 工具级内部结算价(¥,null=继承/同外部价) */
      internalCostPerCall: number | null;
    }>;

    /** MCP 服务器详情(含工具列表) */
    type MCPServerDetail = MCPServer & {
      tools: MCPTool[];
    };

    /** MCP 发布设置(三档可见性+需审批,与模型发布同构) */
    type MCPPublishParams = {
      mcpServerId: CommonType.IdType;
      isPublished: boolean;
      visibilityType: 'all' | 'selected' | 'user';
      requiresApproval: boolean;
      /** 可见部门(selected 模式) */
      departmentIds: CommonType.IdType[];
      /** 可见用户(user 模式) */
      userIds: CommonType.IdType[];
    };

    /** MCP 发布设置回显(含 selected/user 模式的可见部门/用户) */
    type MCPPublishView = MCPPublishParams;

    // ───────────────── MCP 调用日志 McpLog(P3·从 LiteLLM SpendLogs 回流) ─────────────────

    /** MCP 调用日志(userName/aiKeyName 后端回填;serverName 落库已冗余) */
    type McpLog = Common.CommonRecord<{
      logId: CommonType.IdType;
      requestId: string;
      userId: CommonType.IdType;
      aiKeyId: CommonType.IdType;
      /** 归因MCP服务器(0=未匹配) */
      mcpServerId: CommonType.IdType;
      /** 服务器路由名(未匹配时为原始名) */
      serverName: string;
      /** 网关工具全名(LiteLLM 原始锚点) */
      namespacedName: string;
      /** 工具名(匹配 MCPTool 后回填) */
      toolName: string;
      externalCost: number;
      internalCost: number;
      durationMs: number;
      status: string;
      startedAt: string;
      endedAt: string;
      sessionId: string;
      userName: string;
      aiKeyName: string;
    }>;

    /** MCP 调用日志搜索参数(mcpServerId 0=不限;时间 ISO8601) */
    type McpLogSearchParams = CommonType.RecordNullable<
      Pick<McpLog, 'userId' | 'aiKeyId' | 'mcpServerId' | 'toolName' | 'status'> & {
        startTime?: string | null;
        endTime?: string | null;
      } & Api.Common.CommonSearchParams
    >;

    /** MCP 接入配置(用户侧:客户端配置 JSON 含主 Key 明文鉴权头) */
    type MCPConnectConfig = {
      name: string;
      serverName: string;
      /** 网关接入点 {publicUrl}/{serverName}/mcp */
      mcpUrl: string;
      description: string;
      instructions: string;
      documentationUrl: string;
      /** 工具清单 */
      tools: { name: string; description: string }[];
      /** 客户端接入配置 JSON */
      config: { mcpServers: Record<string, { url: string; name?: string; headers?: Record<string, string> }> } | null;
    };

    // ───────────────── Skill 技能包(AI 市场 P2 收尾) ─────────────────

    /** 可授权 Skill(精简版,Key 授权下拉与广场卡片共用;锚点=skillId 字符串) */
    type AvailableSkill = {
      skillId: CommonType.IdType;
      name: string;
      version: string;
      author: string;
      description: string;
      category: string;
      tags: string[];
      iconUrl: string;
      documentationUrl: string;
      requiresApproval: boolean;
      /** 是否已上传 zip 包(未上传时广场下载置灰) */
      hasPackage: boolean;
      installCount: number;
    };

    /** Skill 技能包(管理面出网视图;zip 存服务端 uploads/skills,经鉴权端点分发) */
    type Skill = Common.CommonRecord<{
      skillId: CommonType.IdType;
      name: string;
      version: string;
      author: string;
      description: string;
      category: string;
      tags: string[];
      iconUrl: string;
      documentationUrl: string;
      /** Agent 安装提示词(接入页展示) */
      agentInstallPrompt: string;
      /** 使用说明(接入页展示) */
      usageInstructions: string;
      /** zip 存储键(空=未上传) */
      zipFilename: string;
      zipOriginName: string;
      zipSize: number;
      installCount: number;
      isActive: boolean;
      isPublished: boolean;
      /** 可见范围(all/selected/user) */
      visibilityType: 'all' | 'selected' | 'user';
      requiresApproval: boolean;
    }>;

    /** Skill 搜索参数(name 模糊匹配名称/作者;其余精确) */
    type SkillSearchParams = CommonType.RecordNullable<
      Pick<Skill, 'name' | 'category' | 'isActive' | 'isPublished'> & Api.Common.CommonSearchParams
    >;

    /** Skill 新增/修改参数(zip 包另走上传端点) */
    type SkillOperateParams = CommonType.RecordNullable<
      Pick<
        Skill,
        | 'skillId'
        | 'name'
        | 'version'
        | 'author'
        | 'description'
        | 'category'
        | 'tags'
        | 'iconUrl'
        | 'documentationUrl'
        | 'agentInstallPrompt'
        | 'usageInstructions'
        | 'isActive'
      >
    >;

    /** Skill 列表 */
    type SkillList = Api.Common.PaginatingQueryRecord<Skill>;

    /** Skill 发布设置(三档可见性+需审批,与模型/MCP 发布同构) */
    type SkillPublishParams = {
      skillId: CommonType.IdType;
      isPublished: boolean;
      visibilityType: 'all' | 'selected' | 'user';
      requiresApproval: boolean;
      /** 可见部门(selected 模式) */
      departmentIds: CommonType.IdType[];
      /** 可见用户(user 模式) */
      userIds: CommonType.IdType[];
    };

    /** Skill 发布设置回显(含 selected/user 模式的可见部门/用户) */
    type SkillPublishView = SkillPublishParams;

    /** Skill 使用日志(回填用户名/技能名) */
    type SkillUsage = {
      id: number;
      userId: CommonType.IdType;
      userName: string;
      skillId: CommonType.IdType;
      skillName: string;
      action: string;
      createTime: string;
    };

    /** Skill 使用日志搜索参数(skillId/userId/action 精确) */
    type SkillUsageSearchParams = CommonType.RecordNullable<
      Pick<SkillUsage, 'skillId' | 'userId' | 'action'> & Api.Common.CommonSearchParams
    >;

    /** Skill 使用日志列表 */
    type SkillUsageList = Api.Common.PaginatingQueryRecord<SkillUsage>;

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
      tokens: number;
    };

    /** 排行项(按维度 user/model/aiKey,排序键 cost/requests/tokens) */
    type TopItem = {
      name: string;
      cost: number;
      requests: number;
      tokens: number;
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
    type CredentialFormat = 'openai' | 'anthropic' | 'lmstudio' | 'ollama';

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

    /** 模型新增/修改参数(update 允许改 modelKey,触发关联部署在 LiteLLM 侧级联重建) */
    type ModelOperateParams = CommonType.RecordNullable<
      Pick<Model, 'modelId' | 'modelKey' | 'name' | 'category' | 'logoProviderType' | 'description' | 'capabilities'>
    >;

    type ModelList = Api.Common.PaginatingQueryRecord<Model>;

    /** 模型发布设置视图(GET publish/:id 返回) */
    type ModelPublishView = {
      modelId: CommonType.IdType;
      isPublished: boolean;
      visibilityType: 'all' | 'selected' | 'user';
      requiresApproval: boolean;
      /** 可见部门 ID 列表(selected 模式) */
      departmentIds: CommonType.IdType[];
      /** 可见用户 ID 列表(user 模式) */
      userIds: CommonType.IdType[];
    };

    /** 模型发布设置提交参数(selected/user 档且 isPublished 时对应 ID 列表必填) */
    type ModelPublishParams = {
      modelId: CommonType.IdType;
      isPublished: boolean;
      visibilityType: 'all' | 'selected' | 'user';
      requiresApproval: boolean;
      departmentIds: CommonType.IdType[];
      userIds: CommonType.IdType[];
    };

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
      /** 关联模型 ID(用户请求标识) */
      modelKey: string;
      /** 关联凭证名(无关联为空) */
      credentialName: string;
      /** 凭证协议格式(openai/anthropic) */
      format: string;
      /** 关联供应商ID(编辑回填供应商下拉用,后端视图联表凭证带出) */
      providerId: CommonType.IdType;
      /** 关联供应商类型 */
      providerType: string;
      /** LiteLLM 侧完整调用名(三态命名,routable 版) */
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

    // ───────────────── 路由策略 RouterSettings ─────────────────

    /**
     * 路由策略类型(对齐 LiteLLM routing_strategy 取值)
     * - simple-shuffle 轮询 / latency-based-routing 最低延迟
     * - cost-based-routing 最低成本 / least-busy 最少使用 / usage-based-routing 按用量均衡
     */
    type RoutingStrategy =
      | 'simple-shuffle'
      | 'latency-based-routing'
      | 'cost-based-routing'
      | 'least-busy'
      | 'usage-based-routing';

    /** 降级链项(源模型 → 降级到的模型列表) */
    type FallbackItem = {
      /** 源模型 ID(model_key) */
      model: string;
      /** 降级到的模型 ID 列表 */
      fallbacks: string[];
    };

    /** 全局路由策略(单例配置,投影同步至 LiteLLM /router/settings) */
    type RouterSettings = {
      routingStrategy: RoutingStrategy;
      /** 降级链 */
      fallbacks: FallbackItem[];
      /** 允许连续失败次数(健康摘除阈值) */
      allowedFails: number;
      /** 冷却时间(秒) */
      cooldownTime: number;
      /** 全局重试次数 */
      numRetries: number;
      /** 全局超时(秒) */
      timeout: number;
      /** 扩展配置(预留,前端暂不配置 UI) */
      config: Record<string, any>;
    };

    /** 路由策略更新参数(可选字段,null=不改) */
    type RouterSettingsParams = CommonType.RecordNullable<
      Pick<RouterSettings, 'routingStrategy' | 'fallbacks' | 'allowedFails' | 'cooldownTime' | 'numRetries' | 'timeout' | 'config'>
    >;

    /** 部署连通性测试参数(管理员视角,经 LiteLLM 数据面) */
    type DeploymentTestParams = {
      deploymentId: CommonType.IdType;
    };

    /** 部署连通性测试结果(技术详情已脱敏+截断) */
    type DeploymentTestResult = {
      success: boolean;
      /** 耗时(毫秒) */
      latencyMs: number;
      /** 错误类别(auth_error/model_not_found/rate_limited/bad_request/provider_error/network_error/unknown) */
      errorCategory: string;
      /** 用户可读错误信息(后端已分类) */
      message: string;
      /** 技术详情(已脱敏+截断,用于排障) */
      technicalDetail: string;
    };

    // ───────────────── AI 密钥 AiKey(搜索/操作参数) ─────────────────

    /** AI 密钥搜索参数(keyType/ownerType/ownerId/scenarioId/isActive 精确;name 模糊) */
    type AiKeySearchParams = CommonType.RecordNullable<
      Pick<AiKey, 'keyType' | 'ownerType' | 'ownerId' | 'scenarioId' | 'name' | 'isActive'> & Api.Common.CommonSearchParams
    >;

    /** AI 密钥新增/修改参数(keyType/ownerType/ownerId 创建必填且不可改；scenarioId 仅场景 Key，null=清空) */
    type AiKeyOperateParams = CommonType.RecordNullable<
      Pick<
        AiKey,
        | 'aiKeyId'
        | 'keyType'
        | 'ownerType'
        | 'ownerId'
        | 'name'
        | 'description'
        | 'scenarioId'
        | 'models'
        | 'mcps'
        | 'skills'
        | 'modelBudgets'
        | 'budgetLimit'
        | 'budgetHardLimit'
        | 'budgetDuration'
        | 'rateLimitMode'
        | 'tpmLimit'
        | 'rpmLimit'
        | 'modelLimits'
        | 'isActive'
        | 'expiresAt'
      >
    >;

    /** 批量开通个人主 Key 参数(deptId 优先取部门下全部用户,userIds 补充,两者并集) */
    type AiKeyBatchCreateParams = {
      deptId?: CommonType.IdType | null;
      userIds?: CommonType.IdType[] | null;
    };

    /** 批量建个人场景 Key 参数(名称模板 {username}/{nickname} 逐用户渲染;资源配置整体作模板套到每个目标) */
    type AiKeyBatchSceneCreateParams = CommonType.RecordNullable<
      Pick<
        AiKey,
        | 'models'
        | 'mcps'
        | 'skills'
        | 'modelBudgets'
        | 'budgetLimit'
        | 'budgetHardLimit'
        | 'budgetDuration'
        | 'rateLimitMode'
        | 'tpmLimit'
        | 'rpmLimit'
        | 'modelLimits'
        | 'isActive'
        | 'expiresAt'
      > & {
        deptId?: CommonType.IdType | null;
        userIds?: CommonType.IdType[] | null;
        /** 名称模板(必填,{username}/{nickname} 占位) */
        nameTemplate: string;
        description?: string | null;
        scenarioId?: CommonType.IdType | null;
      }
    >;

    /** 批量开通个人主 Key 结果(部分成功语义:failed 空数组=全部成功) */
    type AiKeyBatchCreateResult = {
      total: number;
      created: number;
      skipped: number;
      failed: Array<{ userId: CommonType.IdType; name: string; reason: string }>;
    };

    // ───────────────── 使用场景 KeyScenario(场景 Key 的分类字典) ─────────────────

    /** 使用场景(极简字典：名称+描述+启停；建场景 Key 时下拉选择) */
    type KeyScenario = Common.CommonRecord<{
      scenarioId: CommonType.IdType;
      name: string;
      description: string;
      isActive: boolean;
    }>;

    /** 使用场景搜索参数(isActive 精确;name 模糊) */
    type KeyScenarioSearchParams = CommonType.RecordNullable<
      Pick<KeyScenario, 'name' | 'isActive'> & Api.Common.CommonSearchParams
    >;

    /** 使用场景新增/修改参数(name 未软删行内唯一) */
    type KeyScenarioOperateParams = CommonType.RecordNullable<
      Pick<KeyScenario, 'scenarioId' | 'name' | 'description' | 'isActive'>
    >;

    // ───────────────── 调用日志 UsageLog(管理员视角的用量明细) ─────────────────

    /** 用量日志(每笔调用的归因/token/成本；userName/aiKeyName/deploymentName 后端回填) */
    type UsageLog = {
      logId: CommonType.IdType;
      requestId: string;
      userId: CommonType.IdType;
      aiKeyId: CommonType.IdType;
      deploymentId: CommonType.IdType;
      userName: string;
      aiKeyName: string;
      deploymentName: string;
      model: string;
      provider: string;
      callType: string;
      promptTokens: number;
      completionTokens: number;
      totalTokens: number;
      cacheReadTokens: number;
      cacheCreationTokens: number;
      externalCost: number;
      internalCost: number;
      durationMs: number;
      startedAt: string;
      endedAt: string;
    };

    /** 用量日志搜索参数(时间范围 RFC3339;userId/aiKeyId/deploymentId 0=不限) */
    type UsageLogSearchParams = CommonType.RecordNullable<
      Pick<UsageLog, 'userId' | 'aiKeyId' | 'deploymentId' | 'model' | 'provider'> & {
        startTime?: string | null;
        endTime?: string | null;
      } & Api.Common.CommonSearchParams
    >;

    // ───────────────── 资源申请审批 ResourceApplication(P2·AI 市场) ─────────────────

    /** 资源申请(模型订阅审批;userName/resourceName/resourceKey/reviewerName 后端回填) */
    type ApplicationItem = {
      applicationId: CommonType.IdType;
      userId: CommonType.IdType;
      resourceType: 'model' | 'mcp' | 'skill';
      resourceId: CommonType.IdType;
      reason: string;
      status: 'pending' | 'approved' | 'rejected';
      reviewedBy: CommonType.IdType;
      reviewedAt: string | null;
      reviewNotes: string;
      userName: string;
      resourceName: string;
      resourceKey: string;
      reviewerName: string;
      createTime: string;
      updateTime: string;
    };

    /** 申请搜索参数(status/resourceType 空=全部;userId 0=不限,仅管理端生效) */
    type ApplicationSearchParams = CommonType.RecordNullable<
      Pick<ApplicationItem, 'status' | 'resourceType' | 'userId'> & Api.Common.CommonSearchParams
    >;

    /** 提交申请参数(P2 暂仅 resourceType=model) */
    type ApplicationCreateParams = {
      resourceType: string;
      resourceId: CommonType.IdType;
      reason: string;
    };

    /** 单条审批结果(warnings=LiteLLM 同步警告,由每日密钥重同步兜底) */
    type ApplicationReviewResult = {
      warnings: string[];
    };

    /** 批量审批结果(逐条独立事务,单条失败不中断) */
    type BatchReviewResult = {
      success: CommonType.IdType[];
      failed: { applicationId: CommonType.IdType; reason: string }[];
    };

    // ───────────────── 预算管控 BudgetRule(P3·部门/用户级预算规则+软限预警+硬限停用) ─────────────────

    /** 预算规则(含读时聚合已用+预警状态;scopeName 后端回填) */
    type BudgetRuleView = Common.CommonRecord<{
      ruleId: CommonType.IdType;
      scopeType: 'dept' | 'user';
      scopeId: CommonType.IdType;
      scopeName: string;
      budgetLimit: number;
      /** 已用(¥,读时聚合) */
      budgetUsed: number;
      /** 执行率(%) */
      budgetUsedPercent: number;
      budgetHardLimit: boolean;
      budgetDuration: '1d' | '7d' | '30d';
      /** 软限预警阈值(%) */
      softWarnPercent: number;
      isActive: boolean;
      /** 本周期是否已触发软限预警 */
      isSoftWarn: boolean;
      /** 本周期是否已触发硬限超限 */
      isHardLimited: boolean;
    }>;

    /** 预算规则搜索参数(scopeType 空=全部;isActive nil=全部) */
    type BudgetRuleSearchParams = CommonType.RecordNullable<
      { scopeType?: string; isActive?: boolean } & Api.Common.CommonSearchParams
    >;

    /** 预算规则新增/修改参数 */
    type BudgetRuleOperateParams = CommonType.RecordNullable<
      Pick<BudgetRuleView, 'ruleId' | 'scopeType' | 'scopeId' | 'budgetLimit' | 'budgetHardLimit' | 'budgetDuration' | 'softWarnPercent' | 'isActive'>
    >;

    // ───────────────── 成本分析 CostAnalysis(P3·多维成本聚合,管理员视角) ─────────────────

    /** 成本 KPI(含等长上一期环比;上期无消耗时环比为 0) */
    type CostKpi = {
      internalCost: number;
      externalCost: number;
      costDiff: number;
      dailyAvgInternal: number;
      internalChange: number;
      externalChange: number;
      totalRequests: number;
      totalTokens: number;
      days: number;
    };

    /** 成本趋势项(按业务日,内/外双线) */
    type CostTrendItem = {
      date: string;
      internalCost: number;
      externalCost: number;
      requests: number;
      tokens: number;
    };

    /** 成本总览(KPI+趋势) */
    type CostOverview = {
      kpi: CostKpi;
      trend: CostTrendItem[];
    };

    /** 成本明细行(六维共用;value=维度原始值,下钻/跳日志带参用) */
    type CostDetailRow = {
      label: string;
      value: string;
      requests: number;
      promptTokens: number;
      completionTokens: number;
      totalTokens: number;
      internalCost: number;
      externalCost: number;
      costDiff: number;
      activeUsers: number;
      perCapita: number;
    };

    /** 部门下钻成员行(userId=0 为该部门「部门Key消耗/未归因」合并行) */
    type CostScopeUserRow = {
      userId: CommonType.IdType;
      userName: string;
      requests: number;
      totalTokens: number;
      internalCost: number;
      externalCost: number;
    };

    /** 成本分析查询(时间为业务日 YYYY-MM-DD;departmentId 筛选含子树) */
    type CostSearchParams = CommonType.RecordNullable<
      {
        dimension?: 'department' | 'user' | 'model' | 'aiKey' | 'provider' | 'date' | 'mcp';
        sort?: 'internal' | 'external' | 'requests' | 'tokens';
        startDate?: string | null;
        endDate?: string | null;
        departmentId?: CommonType.IdType | null;
        userId?: CommonType.IdType | null;
        aiKeyId?: CommonType.IdType | null;
        model?: string | null;
        provider?: string | null;
      } & Api.Common.CommonSearchParams
    >;
  }
}
