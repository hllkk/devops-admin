package server

import servermod "github.com/hllkk/devops-admin/server/model/server"

// credential_payload.go 凭据键值的纯函数层(掩码/合并，无 IO 无全局依赖，单测覆盖)。
// 对齐 gateway credential_payload.go 的分层约定：派生/掩码只存在于此，service 只编排。

// MaskedValue 出网掩码占位值。前端原样回传该值=未修改，保留旧明文。
const MaskedValue = "******"

// credentialSensitiveKeys 敏感键集合(类型无关统一判定：SSH/BMC/数据库的 password、
// SNMP 的 community 与 v3 认证/加密密钥)。username/host/port 等非敏感键明文回显。
var credentialSensitiveKeys = map[string]struct{}{
	"password":     {},
	"community":    {},
	"authPassword": {},
	"privPassword": {},
}

// MaskCredentialValues 凭据键值出网掩码：敏感键统一打码，其余原样。
// nil 返回空 map(前端统一迭代，不判空)。
func MaskCredentialValues(values map[string]string) map[string]string {
	out := make(map[string]string, len(values))
	for k, v := range values {
		if _, ok := credentialSensitiveKeys[k]; ok {
			out[k] = MaskedValue
			continue
		}
		out[k] = v
	}
	return out
}

// MergeCredentialValues 凭据键值合并语义(编辑回写)：值等于掩码串=未修改保留旧明文，
// 其余覆盖/新增；旧键不在新值中即删除(前端全量键提交，删键=删配置)。
// 返回新 map，不改动入参。
func MergeCredentialValues(old, incoming map[string]string) map[string]string {
	merged := make(map[string]string, len(incoming))
	for k, v := range incoming {
		if v == MaskedValue {
			if oldV, ok := old[k]; ok {
				merged[k] = oldV
			}
			continue
		}
		merged[k] = v
	}
	return merged
}

// ValidateCredentialType 凭据类型域校验(空/未知名拒绝写入)。
func ValidateCredentialType(t string) bool {
	switch t {
	case servermod.CredentialTypeSSH, servermod.CredentialTypeBMC, servermod.CredentialTypeSNMP, servermod.CredentialTypeDB:
		return true
	}
	return false
}

// ValidateAssetType 资产类型域校验(空/未知名拒绝写入)。
func ValidateAssetType(t string) bool {
	switch t {
	case servermod.AssetTypePhysical, servermod.AssetTypeVm, servermod.AssetTypeDockerHost, servermod.AssetTypeDbInstance, servermod.AssetTypeNetDevice:
		return true
	}
	return false
}
