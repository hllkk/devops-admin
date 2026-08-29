package request

import (
	commonReq "github.com/hllkk/devops-admin/server/model/common/request"

	"github.com/hllkk/devops-admin/server/model/common"
)

// SkillSearch Skill 分页查询(对齐前端 GET /gateway/skill/list，query 传输)。
// name 模糊匹配名称/作者；category 精确；isActive/isPublished 指针区分未传与 false
// (query 空串 *bool 绑 false 的坑由 NormalizeEmptyBoolQuery 归一)。
type SkillSearch struct {
	commonReq.PageInfo
	Name        string `json:"name" form:"name"`        // 名称/作者(模糊)
	Category    string `json:"category" form:"category"` // 分类(精确)
	IsActive    *bool  `json:"isActive" form:"isActive"` // 是否启用(nil=不限)
	IsPublished *bool  `json:"isPublished" form:"isPublished"` // 是否发布(nil=不限)
}

// SkillOperateParams Skill 新增/修改(对齐前端 POST/PUT /gateway/skill)。
// create 时 skillId 为空(雪花主键回调填充)；update 时必填。
// zip 包不在元数据表单里——单独走 POST /gateway/skill/:id/package 上传端点。
type SkillOperateParams struct {
	SkillId            int64    `json:"skillId,string" form:"skillId"` // 技能ID(新增为空)
	Name               string   `json:"name" form:"name"`             // 技能名称
	Version            string   `json:"version" form:"version"`       // 版本号(空=1.0.0)
	Author             string   `json:"author" form:"author"`         // 作者/提供方
	Description        string   `json:"description" form:"description"` // 描述
	Category           string   `json:"category" form:"category"`     // 分类(空=general)
	Tags               []string `json:"tags"`                         // 标签
	IconUrl            string   `json:"iconUrl" form:"iconUrl"`       // 图标URL
	DocumentationUrl   string   `json:"documentationUrl" form:"documentationUrl"` // 文档地址
	AgentInstallPrompt string   `json:"agentInstallPrompt" form:"agentInstallPrompt"` // Agent安装提示词
	UsageInstructions  string   `json:"usageInstructions" form:"usageInstructions"` // 使用说明
	IsActive           *bool    `json:"isActive"`                     // 是否启用(nil=不改)
}

// SkillPublishParams Skill 发布设置(对齐前端 PUT /gateway/skill/publish)。
// visibilityType=selected 且 isPublished=true 时 departmentIds 必填；=user 时 userIds 必填。
// ID 列表用 Int64StringSlice：前端 IdType 混 string/number，元素级兼容反序列化。
type SkillPublishParams struct {
	SkillId          int64                   `json:"skillId,string" form:"skillId"` // 技能ID
	IsPublished      bool                    `json:"isPublished" form:"isPublished"` // 是否发布
	VisibilityType   string                  `json:"visibilityType" form:"visibilityType"` // 可见范围(all/selected/user)
	RequiresApproval bool                    `json:"requiresApproval" form:"requiresApproval"` // 使用需审批
	DepartmentIds    common.Int64StringSlice `json:"departmentIds" form:"departmentIds"` // 可见部门(selected 模式)
	UserIds          common.Int64StringSlice `json:"userIds" form:"userIds"`       // 可见用户(user 模式)
}

// SkillUsageSearch Skill 使用日志分页查询(对齐前端 GET /gateway/skill/usage/list)。
type SkillUsageSearch struct {
	commonReq.PageInfo
	SkillId int64  `json:"skillId,string" form:"skillId"` // 技能ID(精确,0=不限)
	UserId  int64  `json:"userId,string" form:"userId"`   // 用户ID(精确,0=不限)
	Action  string `json:"action" form:"action"`          // 动作(精确,空=不限)
}
