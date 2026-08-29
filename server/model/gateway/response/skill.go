package response

import (
	"time"

	"github.com/hllkk/devops-admin/server/model/common"
	"github.com/hllkk/devops-admin/server/model/gateway"
)

// SkillView Skill 出网视图：模型本体(tags 密文 JSONB 列 json:"-" 不出网) +
// 转换后的 tags 字符串数组。
type SkillView struct {
	gateway.Skill
	Tags []string `json:"tags"` // 标签(JSONB 转出网)
}

// SkillPublishView Skill 发布设置视图(含 selected/user 模式的可见部门与可见用户回显)。
type SkillPublishView struct {
	SkillId          int64                   `json:"skillId,string"`     // 技能ID
	IsPublished      bool                    `json:"isPublished"`        // 是否发布
	VisibilityType   string                  `json:"visibilityType"`     // 可见范围(all/selected/user)
	RequiresApproval bool                    `json:"requiresApproval"`   // 使用需审批
	DepartmentIds    common.Int64StringSlice `json:"departmentIds"`      // 可见部门(selected 模式,string[] 雪花id)
	UserIds          common.Int64StringSlice `json:"userIds"`            // 可见用户(user 模式,string[] 雪花id)
}

// AvailableSkillView 可用 Skill(精简版，供 Key 授权选择与广场卡片)。
// hasPackage=false 表示尚未上传 zip，广场下载按钮置灰。
type AvailableSkillView struct {
	SkillId          int64    `json:"skillId,string"`     // 技能ID(授权锚点,AiKey.skills 存其字符串)
	Name             string   `json:"name"`               // 技能名称
	Version          string   `json:"version"`            // 版本号
	Author           string   `json:"author"`             // 作者/提供方
	Description      string   `json:"description"`        // 描述
	Category         string   `json:"category"`           // 分类
	Tags             []string `json:"tags"`               // 标签
	IconUrl          string   `json:"iconUrl"`            // 图标
	DocumentationUrl string   `json:"documentationUrl"`   // 文档地址
	RequiresApproval bool     `json:"requiresApproval"`   // 使用需审批
	HasPackage       bool     `json:"hasPackage"`         // 是否已上传zip包
	InstallCount     int64    `json:"installCount"`       // 下载次数
}

// SkillUsageView Skill 使用日志视图(回填用户名/技能名，防 N+1 一次 IN 批量)。
type SkillUsageView struct {
	Id         int64     `json:"id"`             // 日志ID
	UserId     int64     `json:"userId,string"`  // 用户ID
	UserName   string    `json:"userName"`       // 用户名(回填)
	SkillId    int64     `json:"skillId,string"` // 技能ID
	SkillName  string    `json:"skillName"`      // 技能名(回填)
	Action     string    `json:"action"`         // 动作
	CreateTime time.Time `json:"createTime"`     // 时间
}
