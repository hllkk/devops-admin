package request

import (
	commonReq "github.com/hllkk/devops-admin/server/model/common/request"
)

// CredentialSearch 凭据分页查询(对齐前端 GET /server/credential/list，query 传输)。
// credentialName 模糊；credentialType/isActive 精确(指针区分未传与 false)。
type CredentialSearch struct {
	commonReq.PageInfo
	CredentialName string `json:"credentialName" form:"credentialName"` // 凭据名称(模糊)
	CredentialType string `json:"credentialType" form:"credentialType"` // 凭据类型(精确)
	IsActive       *bool  `json:"isActive" form:"isActive"`             // 是否启用(精确,nil=不限)
}

// CredentialOperateParams 凭据新增/修改(对齐前端 POST/PUT /server/credential)。
// credential_values 合并语义(对齐 gateway credential)：敏感键掩码回传=未修改保留旧明文，
// 新值覆盖，可新增键。credentialType 建后不可改(域内字段集合不同)。
type CredentialOperateParams struct {
	CredentialId     int64             `json:"credentialId,string" form:"credentialId"`  // 凭据ID(新增为空)
	CredentialName   string            `json:"credentialName" form:"credentialName"`     // 凭据名称
	CredentialType   string            `json:"credentialType" form:"credentialType"`     // 凭据类型(ssh/bmc/snmp/db)
	CredentialValues map[string]string `json:"credentialValues" form:"credentialValues"` // 凭据键值(敏感值可掩码回传=保留旧明文)
	Description      string            `json:"description" form:"description"`           // 描述
	IsActive         *bool             `json:"isActive" form:"isActive"`                 // 是否启用(nil=不改/默认true)
}
