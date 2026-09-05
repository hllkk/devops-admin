package gateway

import (
	"gorm.io/datatypes"

	"github.com/hllkk/devops-admin/server/global"
)

// Skill AI 技能包（AI 市场 P2：企业内 Skill 统一注册/发布/授权/分发）。
// 平台自有资源：不经 LiteLLM（无路由投影/无远端同步），授权锚点 = AiKey.skills
// JSONB（skill ID 字符串数组，ID 不可变故无改名级联），下载经平台端点鉴权分发。
// zip 包存 uploads/skills/{skillId}_{时间戳}.zip（独立于静态公开的 uploads/file，
// 防匿名直连绕过审批）；zip_filename 是存储键，原始名/大小随行供展示。
type Skill struct {
	global.OPS_AUDIT_MODEL
	SkillId            int64          `json:"skillId,string" gorm:"primarykey;comment:技能ID(雪花)"`                    // 技能ID(雪花)
	Name               string         `json:"name" gorm:"size:128;comment:技能名称"`                                // 技能名称
	Version            string         `json:"version" gorm:"size:20;default:1.0.0;comment:版本号"`                        // 版本号
	Author             string         `json:"author" gorm:"size:128;comment:作者/提供方"`                              // 作者/提供方
	Description        string         `json:"description" gorm:"type:text;comment:描述"`                                // 描述
	Category           string         `json:"category" gorm:"size:50;default:general;comment:分类"`                       // 分类(广场筛选)
	Tags               datatypes.JSON `json:"-" gorm:"type:jsonb;comment:标签" swaggertype:"object"`                  // 标签(string[]，service 转出网)
	IconUrl            string         `json:"iconUrl" gorm:"size:500;comment:图标URL"`                                  // 图标
	DocumentationUrl   string         `json:"documentationUrl" gorm:"size:500;comment:文档地址"`                             // 文档地址
	AgentInstallPrompt string         `json:"agentInstallPrompt" gorm:"type:text;comment:Agent安装提示词"`                    // Agent 安装提示词(接入页展示)
	UsageInstructions  string         `json:"usageInstructions" gorm:"type:text;comment:使用说明"`                             // 使用说明(接入页展示)
	ZipFilename        string         `json:"zipFilename" gorm:"size:200;comment:zip存储键(uploads/skills下,空=未上传)"`        // zip 存储键
	ZipOriginName      string         `json:"zipOriginName" gorm:"size:200;comment:zip原始文件名"`                            // zip 原始文件名
	ZipSize            int64          `json:"zipSize" gorm:"comment:zip大小(字节)"`                                    // zip 大小(字节)
	InstallCount       int64          `json:"installCount" gorm:"default:0;comment:下载次数"`                               // 下载次数
	IsActive           bool           `json:"isActive" gorm:"default:true;comment:是否启用"`                               // 是否启用
	IsPublished        bool           `json:"isPublished" gorm:"default:false;comment:是否发布到用户端"`                        // 是否发布
	VisibilityType     string         `json:"visibilityType" gorm:"size:20;default:all;comment:可见范围(all/selected/user/mixed)"` // 可见范围(与模型/MCP同口径,含部门+用户混合档)
	RequiresApproval   bool           `json:"requiresApproval" gorm:"default:false;comment:使用是否需审批"`                      // 使用需审批
}

func (Skill) TableName() string {
	return "gateway_skill"
}

// SkillVisibility Skill 部门可见性(发布投影表，visibility_type=selected 时使用)。
// 与 gateway_mcp_visibility 同口径：非业务实体，重建时物理删除(Unscoped)，
// 软删行会占住唯一索引挡住同组合重新发布。
type SkillVisibility struct {
	global.OPS_MODEL
	SkillId     int64 `json:"skillId,string" gorm:"uniqueIndex:idx_gateway_skill_visibility;comment:关联技能ID"`                          // 关联技能
	DepartmentId int64 `json:"departmentId,string" gorm:"uniqueIndex:idx_gateway_skill_visibility;comment:关联部门ID(sys_departments.dept_id)"` // 关联部门
}

func (SkillVisibility) TableName() string {
	return "gateway_skill_visibility"
}

// SkillVisibilityUser Skill 用户可见性(发布投影表，visibility_type=user 时使用)。
// 与 gateway_mcp_visibility_user 同口径。
type SkillVisibilityUser struct {
	global.OPS_MODEL
	SkillId int64 `json:"skillId,string" gorm:"uniqueIndex:idx_gateway_skill_visibility_user;comment:关联技能ID"`                 // 关联技能
	UserId  int64 `json:"userId,string" gorm:"uniqueIndex:idx_gateway_skill_visibility_user;comment:关联用户ID(sys_users.id)"` // 关联用户
}

func (SkillVisibilityUser) TableName() string {
	return "gateway_skill_visibility_user"
}

// SkillUsageLog Skill 使用日志（下载动作留痕，usage 计量/审计用；时间用基座 createTime）。
type SkillUsageLog struct {
	global.OPS_BASE
	Id      int64  `json:"id" gorm:"primarykey;autoIncrement;comment:日志ID(自增)"`     // 日志ID(自增)
	UserId  int64  `json:"userId,string" gorm:"index;comment:用户ID(sys_users.id;Agent经Key下载取Key owner,部门Key为0)"` // 用户
	SkillId int64  `json:"skillId,string" gorm:"index;comment:技能ID"`              // 技能
	AiKeyId int64  `json:"aiKeyId,string" gorm:"index;comment:密钥ID(Agent直连下载归因;0=登录态下载)"` // 密钥归因(0=登录态)
	Action  string `json:"action" gorm:"size:20;comment:动作(download/agent_download)"`         // 动作(登录态/Agent直连)
}

func (SkillUsageLog) TableName() string {
	return "gateway_skill_usage_log"
}

// Skill 使用日志动作
const (
	SkillActionDownload      = "download"       // 下载技能包(登录态)
	SkillActionAgentDownload = "agent_download" // Agent 经 AiKey 直连下载技能包
)
