/**
 * Namespace Api
 *
 * backend api module: "server"（服务器模块：资产管理/凭据/agent/IPMI/Docker/DB/SNMP）
 */
declare namespace Api {
  /**
   * namespace Server
   *
   * backend api module: "server"
   */
  namespace Server {
    /** 资产类型(统一资产表类型域，与后端 model/server 常量严格对齐) */
    type AssetType = 'physical' | 'vm' | 'docker_host' | 'db_instance' | 'net_device';

    /** 监控在线状态(采集链路回写，管理端只读) */
    type MonitorStatus = 'online' | 'offline' | 'unknown';

    /** Agent 状态(安装流与心跳回写，管理端只读) */
    type AgentStatus = 'none' | 'installing' | 'running' | 'lost';

    /** 凭据类型(与后端 model/server 常量严格对齐) */
    type CredentialType = 'ssh' | 'bmc' | 'snmp' | 'db';

    /** 统一资产(物理机/虚机/docker主机/db实例/网络设备) */
    type Asset = Common.CommonRecord<{
      /** 资产ID */
      assetId: CommonType.IdType;
      /** 资产名称 */
      assetName: string;
      /** 资产类型 */
      assetType: AssetType;
      /** 管理IP */
      manageIp: string;
      /** SSH端口 */
      sshPort: number;
      /** SSH用户名(公钥认证用户) */
      sshUsername: string;
      /** SSH验证状态(录入即验证,公钥已部署可用) */
      sshVerified: boolean;
      /** 操作系统(linux/windows) */
      osType: string;
      /** 环境标签(prod/test/dev) */
      env: string;
      /** 机房/位置 */
      location: string;
      /** 是否启用 */
      isActive: boolean;
      /** 监控在线状态 */
      monitorStatus: MonitorStatus;
      /** Agent状态 */
      agentStatus: AgentStatus;
      /** Agent版本(注册/心跳回写) */
      agentVersion: string;
      /** Agent主机名(注册回写) */
      agentHostname: string;
      /** 最近心跳时间(UTC) */
      lastHeartbeatAt: string | null;
      /** SSH凭据ID(0=未关联) */
      credentialId: CommonType.IdType;
      /** 采集通道配置(BMC地址等，按类型) */
      channelConfig: Record<string, any> | null;
      /** 描述 */
      description: string;
    }>;

    /** 服务器概览统计(ByType/ByEnv 含停用;MonitorStatus/AgentStatus 仅启用中) */
    type AssetOverview = {
      /** 资产总数(未软删) */
      total: number;
      /** 启用中资产数 */
      activeTotal: number;
      /** 各类型数量 */
      byType: Record<string, number>;
      /** 监控状态数量(启用中) */
      monitorStatus: Record<string, number>;
      /** Agent状态数量(启用中) */
      agentStatus: Record<string, number>;
      /** 环境分布 */
      byEnv: Record<string, number>;
    };

    /** 资产搜索参数(assetName/manageIp 模糊;其余精确) */
    type AssetSearchParams = CommonType.RecordNullable<
      Pick<Api.Server.Asset, 'assetName' | 'manageIp' | 'assetType' | 'env' | 'monitorStatus' | 'isActive'> &
        Api.Common.CommonSearchParams
    >;

    /** 资产新增/修改参数(create 时 assetId 为空;update 必填。isActive 为 null 表示不改) */
    type AssetOperateParams = CommonType.RecordNullable<
      Pick<
        Api.Server.Asset,
        'assetId' | 'assetName' | 'assetType' | 'manageIp' | 'sshPort' | 'sshUsername' | 'osType' | 'env' | 'location' | 'isActive' | 'credentialId' | 'description'
      >
    > & {
      /** SSH密码(一次性:仅保存时验证+部署公钥,不落库;编辑留空=公钥复验) */
      sshPassword?: string;
    };

    /** 采集凭据(credential_values 密文不出网，视图下发掩码键值) */
    type Credential = Common.CommonRecord<{
      /** 凭据ID */
      credentialId: CommonType.IdType;
      /** 凭据名称 */
      credentialName: string;
      /** 凭据类型 */
      credentialType: CredentialType;
      /** 描述 */
      description: string;
      /** 是否启用 */
      isActive: boolean;
    }> & {
      /** 凭据键值(敏感值掩码 ******，掩码原样回传=未修改保留旧明文) */
      credentialValues: Record<string, string>;
    };

    /** 凭据搜索参数(credentialName 模糊;其余精确) */
    type CredentialSearchParams = CommonType.RecordNullable<
      Pick<Api.Server.Credential, 'credentialName' | 'credentialType' | 'isActive'> & Api.Common.CommonSearchParams
    >;

    /** 凭据新增/修改参数(create 时 credentialId 为空;credentialType 建后不可改) */
    type CredentialOperateParams = CommonType.RecordNullable<
      Pick<Api.Server.Credential, 'credentialId' | 'credentialName' | 'credentialType' | 'description' | 'isActive'>
    > & {
      /** 凭据键值(敏感值掩码 ******，掩码原样回传=未修改保留旧明文) */
      credentialValues: Record<string, string>;
    };

    /** agent 安装任务状态(轮询) */
    type AgentInstallStatus = {
      /** 任务ID */
      taskId: string;
      /** 资产ID */
      assetId: CommonType.IdType;
      /** running/success/failed */
      status: 'running' | 'success' | 'failed';
      /** 当前步骤(连接目标机/部署公钥/上传二进制/注册服务/启动) */
      step: string;
      /** 失败原因/成功摘要 */
      message?: string;
      /** 开始时间(UTC) */
      startedAt: string;
      /** 结束时间(UTC，空=进行中) */
      finishedAt?: string | null;
    };

    /** agent 心跳上报的轻量指标快照(slice3 完整管道前的查看抽屉数据) */
    type AgentSnapshot = {
        /** CPU使用率% */
        cpuPercent: number;
        /** 1分钟负载 */
        loadavg1: number;
        /** 内存总量KB */
        memTotalKb: number;
        /** 内存可用KB */
        memAvailableKb: number;
        /** 根分区总量 */
        diskTotalBytes: number;
        /** 根分区已用 */
        diskUsedBytes: number;
        /** 网络入速率 B/s */
        netInBps: number;
        /** 网络出速率 B/s */
        netOutBps: number;
        /** 系统运行时长(秒) */
        uptimeSec: number;
        /** 上报时间(UTC) */
        reportedAt: string;
        /** 过期标记(agent 未上报时 true,数据为零值) */
        stale: boolean;
    };

    /** 指标趋势点(实时窗口帧与历史分钟/小时聚合行统一投影) */
    type MetricPoint = {
      /** 点时间(UTC, ISO) */
      ts: string;
      /** CPU使用率% */
      cpu: number;
      /** CPU峰值%(实时帧=cpu) */
      cpuMax: number;
      /** 内存使用率% */
      memPct: number;
      /** 内存峰值% */
      memPctMax: number;
      /** 磁盘使用率% */
      diskPct: number;
      /** 磁盘峰值% */
      diskPctMax: number;
      /** 网络入速率 B/s */
      netIn: number;
      /** 入峰值 */
      netInMax: number;
      /** 网络出速率 B/s */
      netOut: number;
      /** 出峰值 */
      netOutMax: number;
      /** 1分钟负载 */
      loadavg1: number;
    };

    /** 指标趋势响应 */
    type MetricsTrend = {
      /** 1h/1d/7d/30d */
      range: string;
      /** 时间正序点列 */
      points: MetricPoint[];
    };

    /** 凭据下拉选项(资产表单用，仅启用中) */
    type CredentialOption = {
      credentialId: CommonType.IdType;
      credentialName: string;
      credentialType: CredentialType;
    };
  }
}
