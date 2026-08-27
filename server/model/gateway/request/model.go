package request

import (
	commonReq "github.com/hllkk/devops-admin/server/model/common/request"
)

// ModelSearch 模型分页查询(对齐前端 GET /gateway/model/list，query 传输)。
type ModelSearch struct {
	commonReq.PageInfo
	Name        string `json:"name" form:"name"`               // 展示名(模糊)
	ModelKey    string `json:"modelKey" form:"modelKey"`       // 路由名(模糊)
	Category    string `json:"category" form:"category"`       // 类别(精确)
	IsActive    *bool  `json:"isActive" form:"isActive"`       // 是否启用(精确,nil=不限)
	IsPublished *bool  `json:"isPublished" form:"isPublished"` // 是否发布(精确,nil=不限)
}

// ModelOperateParams 模型新增/修改(对齐前端 POST/PUT /gateway/model)。
// create 时 modelId 为空；update 允许改 modelKey(触发关联部署的路由名级联重建，尽力而为)。
type ModelOperateParams struct {
	ModelId          int64    `json:"modelId,string" form:"modelId"`   // 模型ID(新增为空)
	ModelKey         string   `json:"modelKey" form:"modelKey"`       // 路由名(全局唯一,可改,改名级联)
	Name             string   `json:"name" form:"name"`               // 展示名
	Category         string   `json:"category" form:"category"`       // 类别(空=默认 chat)
	Description      string   `json:"description" form:"description"` // 描述
	LogoProviderType string   `json:"logoProviderType" form:"logoProviderType"` // LOGO供应商类型
	Capabilities     []string `json:"capabilities" form:"capabilities"`        // 能力标签
}

// ModelPublishParams 模型发布设置(对齐前端 PUT /gateway/model/publish)。
// visibilityType=selected 且 isPublished=true 时 departmentIds 必填(重建部门可见行)；
// visibilityType=user 且 isPublished=true 时 userIds 必填(重建用户可见行)。
type ModelPublishParams struct {
	ModelId          int64   `json:"modelId,string" form:"modelId"`  // 模型ID
	IsPublished      bool    `json:"isPublished" form:"isPublished"` // 是否发布
	VisibilityType   string  `json:"visibilityType" form:"visibilityType"` // 可见范围(all/selected/user)
	RequiresApproval bool    `json:"requiresApproval" form:"requiresApproval"` // 订阅需审批
	DepartmentIds    []int64 `json:"departmentIds" form:"departmentIds"`      // 可见部门(selected 模式)
	UserIds          []int64 `json:"userIds" form:"userIds"`          // 可见用户(user 模式)
}

// DeploymentSearch 部署分页查询(对齐前端 GET /gateway/model/deployment/list)。
type DeploymentSearch struct {
	commonReq.PageInfo
	ModelId      int64  `json:"modelId,string" form:"modelId"`  // 关联模型(0=不限)
	CredentialId int64  `json:"credentialId,string" form:"credentialId"` // 关联凭证(0=不限)
	Keyword      string `json:"keyword" form:"keyword"`         // 部署名(模糊)
	IsActive     *bool  `json:"isActive" form:"isActive"`       // 是否启用(精确,nil=不限)
}

// DeploymentOperateParams 部署新增/修改(对齐前端 POST/PUT /gateway/model/deployment)。
// litellmParams 必含 model 键；credentialId 非 0 时绑定平台凭证(剔 inline key，api_base 归凭证)；
// 敏感键(api_key 等)支持掩码回传(与库内旧值掩码一致则保留旧明文)。
type DeploymentOperateParams struct {
	DeploymentId     int64          `json:"deploymentId,string" form:"deploymentId"` // 部署ID(新增为空)
	ModelId          int64          `json:"modelId,string" form:"modelId"`   // 关联模型
	CredentialId     int64          `json:"credentialId,string" form:"credentialId"` // 关联凭证(0=内联参数)
	DeployName       string         `json:"deployName" form:"deployName"`    // 部署名
	LitellmParams    map[string]any `json:"litellmParams" form:"litellmParams"`    // 路由参数(明文或掩码回传)
	ModelInfo        map[string]any `json:"modelInfo" form:"modelInfo"`     // 元数据(可选,定价镜像自动重算)
	BillingType      string         `json:"billingType" form:"billingType"`  // 计费类型(空=token)
	CostPerCall      *float64       `json:"costPerCall" form:"costPerCall"`  // 单次成本
	MonthlyCallQuota *int           `json:"monthlyCallQuota" form:"monthlyCallQuota"` // 月配额
	IsActive         *bool          `json:"isActive" form:"isActive"`       // 是否启用
}

// DeploymentTestParams 部署连通性测试(管理员视角,经 LiteLLM 数据面)。
type DeploymentTestParams struct {
	DeploymentId int64 `json:"deploymentId,string" form:"deploymentId"` // 部署ID
}
