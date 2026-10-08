package server

import (
	"time"

	"github.com/hllkk/devops-admin/server/global"
	"gorm.io/datatypes"
)

// Asset 服务器模块统一资产(锚点)：物理机/虚拟机/docker主机/数据库实例/网络设备
// 一张表(asset_type 区分)，各类型的采集通道差异收敛在 channel_config(JSONB)。
// monitor_status/agent_status 由采集链路回写，管理端只读。
// 详见 aiDoc/modules/business-modules.md「服务器模块」节。
type Asset struct {
	global.OPS_AUDIT_MODEL
	AssetId         int64          `json:"assetId,string" gorm:"primarykey;comment:资产ID(雪花)"`                                              // 资产ID
	AssetName       string         `json:"assetName" gorm:"index;comment:资产名称"`                                                            // 资产名称(未软删行内唯一,服务层查重)
	AssetType       string         `json:"assetType" gorm:"size:20;index;comment:资产类型(physical/vm/docker_host/db_instance/net_device)"`    // 资产类型
	ManageIp        string         `json:"manageIp" gorm:"size:45;comment:管理IP(v4/v6)"`                                                    // 管理IP
	SshPort         int            `json:"sshPort" gorm:"default:22;comment:SSH端口"`                                                        // SSH端口(验证/安装agent/SSH访问用)
	SshUsername     string         `json:"sshUsername" gorm:"size:64;comment:SSH用户名(公钥认证用户,验证/安装/SSH访问用)"`                       // SSH用户名(spug 模式:用户名是资产属性)
	SshVerified     bool           `json:"sshVerified" gorm:"default:false;comment:SSH验证状态(录入即验证,保存时强制验证通过)"`                  // SSH验证状态(公钥已部署可用)
	OsType          string         `json:"osType" gorm:"size:20;comment:操作系统(linux/windows)"`                                              // 操作系统
	Env             string         `json:"env" gorm:"size:20;index;comment:环境标签(prod/test/dev)"`                                           // 环境标签
	Location        string         `json:"location" gorm:"size:100;comment:机房/位置"`                                                         // 机房/位置
	IsActive        bool           `json:"isActive" gorm:"default:true;comment:是否启用"`                                                      // 是否启用(停用后采集/拨测跳过)
	MonitorStatus   string         `json:"monitorStatus" gorm:"size:20;default:unknown;comment:监控在线状态(online/offline/unknown,拨测与agent回写)"` // 监控在线状态
	AgentStatus     string         `json:"agentStatus" gorm:"size:20;default:none;comment:Agent状态(none/installing/running/lost,安装流与心跳回写)"` // Agent状态
	CredentialId    int64          `json:"credentialId,string" gorm:"index;comment:采集凭据ID(关联server_credential,BMC/DB凭据预留;SSH走公钥不关联)"` // 采集凭据ID(0=未关联)
	AgentTokenHash  string         `json:"-" gorm:"size:64;index;comment:Agent持久令牌sha256hex(注册写入,心跳比对;空=未注册)"`                             // Agent持久令牌哈希
	AgentVersion    string         `json:"agentVersion" gorm:"size:32;comment:Agent版本(注册/心跳回写)"`                                           // Agent版本
	AgentHostname   string         `json:"agentHostname" gorm:"size:128;comment:Agent主机名(注册回写,采集元信息)"`                                     // Agent主机名
	LastHeartbeatAt *time.Time     `json:"lastHeartbeatAt" gorm:"comment:最近心跳时间(UTC;lost判定依据)"`                                            // 最近心跳时间
	ChannelConfig   datatypes.JSON `json:"channelConfig" swaggertype:"object" gorm:"type:jsonb;comment:采集通道配置(BMC地址/SNMP参数/DB连接,按类型)"`     // 采集通道配置
	Description     string         `json:"description" gorm:"type:text;comment:描述"`                                                        // 描述
}

// 资产类型(统一资产表的类型域,前端常量严格对齐)
const (
	AssetTypePhysical   = "physical"    // 物理服务器(可带 BMC/IPMI 通道)
	AssetTypeVm         = "vm"          // 虚拟机
	AssetTypeDockerHost = "docker_host" // Docker 主机(P2 本机纳管,P3 扩远程)
	AssetTypeDbInstance = "db_instance" // 数据库实例(MySQL/PG/Redis 直连监控)
	AssetTypeNetDevice  = "net_device"  // 网络设备(SNMP 采集)
)

// 监控在线状态(拨测/agent 回写)
const (
	MonitorStatusOnline  = "online"  // 在线
	MonitorStatusOffline = "offline" // 离线
	MonitorStatusUnknown = "unknown" // 未知(未采集)
)

// Agent 状态(安装流与心跳回写)
const (
	AgentStatusNone       = "none"       // 未安装
	AgentStatusInstalling = "installing" // 安装中(异步安装任务进行中)
	AgentStatusRunning    = "running"    // 运行中(心跳正常)
	AgentStatusLost       = "lost"       // 失联(心跳超时)
)

func (Asset) TableName() string {
	return "server_asset"
}
