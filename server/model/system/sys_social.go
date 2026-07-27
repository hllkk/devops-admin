package system

import "github.com/hllkk/devops-admin/server/global"

// SysSocial 第三方账号关联(本地用户 ↔ 第三方 OAuth 账号)。
//
// 设计对齐 docs/superpowers/specs/2026-07-23-social-binding-design.md:
//   - 嵌入 OPS_AUDIT_MODEL:createTime/updateTime/createBy/updateBy + 软删除(DeletedAt)
//   - 主键 ID 雪花 int64(json "id,string" 对齐前端 IdType)
//   - (source, open_id) 业务唯一:一个第三方账号只能绑一个本地用户(代码层查重)
//   - (user_id, source) 业务唯一:一个本地用户同一 provider 只能绑一个(代码层查重)
//   - AccessToken/RefreshToken AES-256-GCM 加密落库,json:"-" 绝不返回前端
//   - AuthId = source + "_" + openId,对齐 JustAuth 风格与前端类型
//   - UnionId 仅微信开放平台有意义;绑定时 unionId 非空则查重优先用 unionId
type SysSocial struct {
	global.OPS_AUDIT_MODEL
	ID           int64  `gorm:"primarykey;column:id;comment:主键" json:"id,string"`
	UserId       int64  `gorm:"index;comment:本地用户ID" json:"userId,string"`
	Source       string `gorm:"size:32;index;comment:来源标识" json:"source"` // wechat_open/gitee/github
	OpenId       string `gorm:"size:128;index;comment:第三方用户唯一标识" json:"openId"`
	UnionId      string `gorm:"size:128;comment:微信unionid" json:"unionId"` // 仅微信有意义
	AuthId       string `gorm:"size:160;comment:认证唯一ID(source_openId)" json:"authId"`
	NickName     string `gorm:"size:64;comment:第三方昵称(快照)" json:"nickName"`
	Avatar       string `gorm:"size:512;comment:第三方头像URL(快照)" json:"avatar"`
	Email        string `gorm:"size:128;comment:第三方邮箱(快照)" json:"email"`
	Mobile       string `gorm:"size:32;comment:第三方手机号(快照,企微等)" json:"mobile"`
	AccessToken  string `gorm:"size:2048;comment:访问令牌(AES加密)" json:"-"` // 加密存储,不返回前端
	RefreshToken string `gorm:"size:2048;comment:刷新令牌(AES加密)" json:"-"` // 加密存储,不返回前端
	ExpireIn     int64  `gorm:"comment:令牌有效期(秒,0=无信息)" json:"expireIn"`
	// 以下 gorm:"-" 瞬态字段:对齐前端 Api.System.Social 的 JustAuth 全平台风格字段,
	// 本项目只用 wechat_open/gitee/github 三平台,这些字段不落库,序列化输出空串(见设计 9.3)
	UserName         string `gorm:"-" json:"userName"`         // 与 nickName 重复,不落库
	AccessCode       string `gorm:"-" json:"accessCode"`       // 小米等平台专用
	Scope            string `gorm:"-" json:"scope"`            // 授权范围
	TokenType        string `gorm:"-" json:"tokenType"`        // token 类型
	IdToken          string `gorm:"-" json:"idToken"`          // id token
	MacAlgorithm     string `gorm:"-" json:"macAlgorithm"`     // 小米平台
	MacKey           string `gorm:"-" json:"macKey"`           // 小米平台
	Code             string `gorm:"-" json:"code"`             // 授权 code
	OauthToken       string `gorm:"-" json:"oauthToken"`       // Twitter 平台
	OauthTokenSecret string `gorm:"-" json:"oauthTokenSecret"` // Twitter 平台
}

func (SysSocial) TableName() string { return "sys_social" }
