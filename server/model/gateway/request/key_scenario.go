package request

import (
	commonReq "github.com/hllkk/devops-admin/server/model/common/request"
)

// KeyScenarioSearch 场景分页查询。
type KeyScenarioSearch struct {
	commonReq.PageInfo
	Name     string `json:"name" form:"name"`         // 名称(模糊)
	IsActive *bool  `json:"isActive" form:"isActive"` // 是否启用(精确,nil=不限)
}

// KeyScenarioOperateParams 场景新增/修改。
// create 时 scenarioId 为空；name 未软删行内唯一。
type KeyScenarioOperateParams struct {
	ScenarioId  int64  `json:"scenarioId,string" form:"scenarioId"` // 场景ID(新增为空)
	Name        string `json:"name" form:"name"`                   // 名称
	Description string `json:"description" form:"description"`     // 描述
	IsActive    *bool  `json:"isActive" form:"isActive"`           // 启停(nil=新增默认true/修改不改)
}
