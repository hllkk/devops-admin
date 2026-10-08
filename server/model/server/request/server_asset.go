package request

import (
	commonReq "github.com/hllkk/devops-admin/server/model/common/request"
	"gorm.io/datatypes"
)

// AssetSearch 资产分页查询(对齐前端 GET /server/asset/list，query 传输)。
// assetName 模糊；assetType/env/monitorStatus/isActive 精确(指针区分未传与 false)。
type AssetSearch struct {
	commonReq.PageInfo
	AssetName     string `json:"assetName" form:"assetName"`         // 资产名称(模糊)
	ManageIp      string `json:"manageIp" form:"manageIp"`           // 管理IP(模糊)
	AssetType     string `json:"assetType" form:"assetType"`         // 资产类型(精确)
	Env           string `json:"env" form:"env"`                     // 环境标签(精确)
	MonitorStatus string `json:"monitorStatus" form:"monitorStatus"` // 监控在线状态(精确)
	IsActive      *bool  `json:"isActive" form:"isActive"`           // 是否启用(精确,nil=不限)
}

// AssetOperateParams 资产新增/修改(对齐前端 POST/PUT /server/asset)。
// create 时 assetId 为空(雪花主键由回调填充)；update 时必填 assetId。
// monitorStatus/agentStatus 为采集回写字段,不在此接收。
type AssetOperateParams struct {
	AssetId       int64          `json:"assetId,string" form:"assetId"`                           // 资产ID(新增为空)
	AssetName     string         `json:"assetName" form:"assetName"`                              // 资产名称
	AssetType     string         `json:"assetType" form:"assetType"`                              // 资产类型
	ManageIp      string         `json:"manageIp" form:"manageIp"`                                // 管理IP
	SshPort       int            `json:"sshPort" form:"sshPort"`                                  // SSH端口
	SshUsername   string         `json:"sshUsername" form:"sshUsername"`                          // SSH用户名(physical/vm 必填,公钥认证用户)
	SshPassword   string         `json:"sshPassword" form:"sshPassword"`                          // SSH密码(一次性:仅保存时验证+部署公钥,不落库;编辑留空=公钥复验)
	OsType        string         `json:"osType" form:"osType"`                                    // 操作系统
	Env           string         `json:"env" form:"env"`                                          // 环境标签
	Location      string         `json:"location" form:"location"`                                // 机房/位置
	IsActive      *bool          `json:"isActive" form:"isActive"`                                // 是否启用(nil=不改/默认true)
	CredentialId  int64          `json:"credentialId,string" form:"credentialId"`                 // 采集凭据ID(0=未关联,BMC/DB凭据预留;SSH走公钥不关联)
	ChannelConfig datatypes.JSON `json:"channelConfig" form:"channelConfig" swaggertype:"object"` // 采集通道配置(BMC地址等,按类型)
	Description   string         `json:"description" form:"description"`                          // 描述
}
