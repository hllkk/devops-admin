package request

import (
	commonReq "github.com/hllkk/devops-admin/server/model/common/request"
)

// CredentialSearch 凭证分页查询(对齐前端 GET /gateway/credential/list，query 传输)。
// credentialName 模糊匹配；providerId 精确(0=不限)；isActive/litellmSynced 精确(指针区分未传与 false)。
type CredentialSearch struct {
	commonReq.PageInfo
	CredentialName string `json:"credentialName" form:"credentialName"` // 凭证名称(模糊)
	ProviderId     int64  `json:"providerId,string" form:"providerId"`  // 关联供应商ID(0=不限)
	IsActive       *bool  `json:"isActive" form:"isActive"`             // 是否启用(精确,nil=不限)
	LitellmSynced  *bool  `json:"litellmSynced" form:"litellmSynced"`   // 是否已同步(精确,nil=不限)
}

// CredentialOperateParams 凭证新增/修改(对齐前端 POST/PUT /gateway/credential)。
// create 时 credentialId 为空(雪花主键由回调填充)；update 时必填 credentialId 且不允许改 credentialName。
// credentialValues 为明文键值 map：更新时传掩码回传值(与旧值掩码一致)则保留旧明文，传新值则覆盖；
// credentialInfo 含 format(openai/anthropic)等非敏感元数据。
type CredentialOperateParams struct {
	CredentialId     int64          `json:"credentialId,string" form:"credentialId"`      // 凭证ID(新增为空)
	CredentialName   string         `json:"credentialName" form:"credentialName"`         // 凭证名称(全局唯一,创建后不可改)
	ProviderId       int64          `json:"providerId,string" form:"providerId"`          // 关联供应商ID(0=未关联)
	CredentialValues map[string]any `json:"credentialValues" form:"credentialValues"`    // 凭证键值(api_key/api_base等,明文或掩码回传)
	CredentialInfo   map[string]any `json:"credentialInfo" form:"credentialInfo"`         // 凭证元数据(format等)
	IsActive         *bool          `json:"isActive" form:"isActive"`                    // 是否启用
	Description      string         `json:"description" form:"description"`              // 描述
}
