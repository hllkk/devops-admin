package system

import "github.com/hllkk/devops-admin/server/global"

// SysSetting 系统设置存储模型：按分类（name）保存一段 JSON 配置（value）。
// 属于内部系统记录，主键用雪花 id；name 建唯一索引，供 upsert 按 name 合并。
type SysSetting struct {
	Id int64 `gorm:"column:id;primaryKey;autoIncrement:false" json:"id,string"`
	global.OPS_MODEL
	Name  string `gorm:"column:name;size:64;uniqueIndex:idx_setting_name;comment:配置分类" json:"name"`
	Value string `gorm:"column:value;type:text;comment:配置JSON" json:"value"`
	Desc  string `gorm:"column:desc;size:255;comment:描述" json:"desc"`
}

// TableName 表名采用蛇形单数（对齐项目规范）。
func (SysSetting) TableName() string { return "sys_setting" }

// SystemSettings 聚合配置 DTO：GET / PUT /system/setting 的请求与响应体。
// general/security 为核心配置（无 omitempty，确保零值准确返回）；
// 其余分类为指针，未配置时返回 nil，由前端按需渲染。
type SystemSettings struct {
	General        *GeneralSettings        `json:"general"`
	Security       *SecuritySettings       `json:"security"`
	Authentication *AuthenticationSettings `json:"authentication,omitempty"`
	Ldap           *LdapSettings           `json:"ldap,omitempty"`
	Notify         *NotifySettings         `json:"notify,omitempty"`
	Disk           *DiskSettings           `json:"disk,omitempty"`
}

// PublicSystemSettings 公开系统设置（登录页使用，仅暴露展示与开关字段，不含敏感信息）。
type PublicSystemSettings struct {
	SystemName         string `json:"systemName"`
	SystemDescription  string `json:"systemDescription"`
	LogoUrl            string `json:"logoUrl"`
	FaviconUrl         string `json:"faviconUrl"`
	EnableVerifyCode   bool   `json:"enableVerifyCode"`
	VerifyCodeType     string `json:"verifyCodeType,omitempty"`
	VerifyCodeLen      int    `json:"verifyCodeLen,omitempty"`
	VerifyCodeExp      int    `json:"verifyCodeExp,omitempty"`
	VerifyCodeTokenExp int    `json:"verifyCodeTokenExp,omitempty"`
	VerifyInaccuracy   int    `json:"verifyInaccuracy,omitempty"`
	EnableWecom        bool   `json:"enableWecom,omitempty"`
	EnableWechat       bool   `json:"enableWechat,omitempty"`
	EnableGitee        bool   `json:"enableGitee,omitempty"`
	EnableGithub       bool   `json:"enableGithub,omitempty"`
}

// GeneralSettings 通用配置：站点信息、默认用户、验证码、日志保留。
type GeneralSettings struct {
	SystemName                string  `json:"systemName"`
	SystemDescription         string  `json:"systemDescription"`
	LogoUrl                   string  `json:"logoUrl"`
	FaviconUrl                string  `json:"faviconUrl"`
	UserDefaultPassword       string  `json:"userDefaultPassword"`
	UserDefaultRole           *string `json:"userDefaultRole"`
	EnableVerifyCode          bool    `json:"enableVerifyCode"`
	VerifyCodeType            string  `json:"verifyCodeType"`            // click / slide / dragdrop / rotate
	VerifyCodeLen             int     `json:"verifyCodeLen"`             // 验证码长度
	VerifyCodeExp             int     `json:"verifyCodeExp"`             // 过期时间(分钟)
	VerifyCodeTokenExp        int     `json:"verifyCodeTokenExp"`        // Token 过期时间(分钟)
	VerifyInaccuracy          int     `json:"verifyInaccuracy"`          // 误差范围(像素)
	LoginLogRetentionDays     int     `json:"loginLogRetentionDays"`     // 登录日志保留天数
	OperationLogRetentionDays int     `json:"operationLogRetentionDays"` // 操作日志保留天数
	Watermark                 bool    `json:"watermark"`
	WatermarkContent          int     `json:"watermarkContent"`
	WatermarkSize             int     `json:"watermarkSize"`
	EnableWechat              bool    `json:"enableWechat"`
	EnableGitee               bool    `json:"enableGitee"`
}

// SecuritySettings 安全配置：密码策略、登录锁定、IP 黑白名单。
type SecuritySettings struct {
	PasswordMinLength        int    `json:"passwordMinLength"`
	PasswordRequireUppercase bool   `json:"passwordRequireUppercase"`
	PasswordRequireLowercase bool   `json:"passwordRequireLowercase"`
	PasswordRequireDigit     bool   `json:"passwordRequireDigit"`
	PasswordRequireSpecial   bool   `json:"passwordRequireSpecial"`
	LoginFailLockCount       int    `json:"loginFailLockCount"`
	LoginFailLockTime        int    `json:"loginFailLockTime"`
	IpValidationEnabled      bool   `json:"ipValidationEnabled"`
	IpValidationMode         string `json:"ipValidationMode"` // blacklist / whitelist
	IpBlacklist              string `json:"ipBlacklist"`
	IpWhitelist              string `json:"ipWhitelist"`
}

// AuthenticationSettings 第三方登录配置（阶段二启用，密钥类字段返回时需脱敏）。
type AuthenticationSettings struct {
	Wecom  *WecomSettings  `json:"wecom,omitempty"`
	Wechat *WechatSettings `json:"wechat,omitempty"`
	Gitee  *GiteeSettings  `json:"gitee,omitempty"`
	Github *GithubSettings `json:"github,omitempty"`
}

type WecomSettings struct {
	EnableWecom               bool   `json:"enableWecom,omitempty"`
	CorpId                    string `json:"corpId,omitempty"`      // 企业 ID
	AgentId                   int    `json:"agentId,omitempty"`     // 应用 AgentId
	AgentSecret               string `json:"agentSecret,omitempty"` // 应用密钥（接口返回脱敏）
	RedirectUri               string `json:"redirectUri,omitempty"` // OAuth 回调地址
	ValidateDomainFileName    string `json:"validateDomainFileName,omitempty"`
	ValidateDomainFileContent string `json:"validateDomainFileContent,omitempty"`
}

type WechatSettings struct {
	EnableWechat bool `json:"enableWechat,omitempty"`
}

type GiteeSettings struct {
	EnableGitee bool `json:"enableGitee,omitempty"`
}

type GithubSettings struct {
	EnableGithub bool `json:"enableGithub,omitempty"`
}

// LdapSettings LDAP 配置：扁平 camelCase，与前端 type 对齐。
type LdapSettings struct {
	Enabled            bool   `json:"enabled,omitempty"`
	Server             string `json:"server,omitempty"`       // 如 ldap://host:389
	BindUser           string `json:"bindUser,omitempty"`     // 绑定用户 DN
	BindPassword       string `json:"bindPassword,omitempty"` // 绑定密码
	BaseOu             string `json:"baseOu,omitempty"`
	SearchPageSize     int    `json:"searchPageSize,omitempty"`
	FieldMapping       string `json:"fieldMapping,omitempty"` // 字段映射 JSON
	SyncEnabled        bool   `json:"syncEnabled,omitempty"`
	SyncDefaultEnabled bool   `json:"syncDefaultEnabled,omitempty"`
	SyncStrategy       string `json:"syncStrategy,omitempty"`     // incremental / full
	ConflictStrategy   string `json:"conflictStrategy,omitempty"` // skip / overwrite / merge
}

// NotifySettings 通知渠道配置（阶段二启用）。
type NotifySettings struct {
	Email *EmailSettings `json:"email,omitempty"`
}

// EmailSettings 邮件配置：字段统一 camelCase（修复 main 的 MAIL_* 全大写）。
type EmailSettings struct {
	SmtpHost       string `json:"smtpHost,omitempty"`
	SmtpPort       int    `json:"smtpPort,omitempty"`
	SmtpServer     string `json:"smtpServer,omitempty"`
	SmtpUser       string `json:"smtpUser,omitempty"`
	SmtpPassword   string `json:"smtpPassword,omitempty"`
	SmtpSslTls     bool   `json:"smtpSslTls,omitempty"`
	SmtpStarttls   bool   `json:"smtpStarttls,omitempty"`
	MailFrom       string `json:"mailFrom,omitempty"`
	MailFromName   string `json:"mailFromName,omitempty"`
	ValidateCerts  bool   `json:"validateCerts,omitempty"`
	UseCredentials bool   `json:"useCredentials,omitempty"`
}

// DiskSettings 网盘配置（网盘模块落地后消费）。
type DiskSettings struct {
	DiskName                    string                  `json:"diskName"`
	DiskLogo                    string                  `json:"diskLogo,omitempty"`
	MaxUploadSize               int                     `json:"maxUploadSize,omitempty"`
	AllowedExtensions           []string                `json:"allowedExtensions,omitempty"`
	BlockedExtensions           []string                `json:"blockedExtensions,omitempty"`
	TrashRetentionDays          int                     `json:"trashRetentionDays,omitempty"`
	StorageQuota                int                     `json:"storageQuota,omitempty"`
	SyncEnabled                 bool                    `json:"syncEnabled,omitempty"`
	ShareLinkPasswordRequired   bool                    `json:"shareLinkPasswordRequired,omitempty"`
	ShareLinkPasswordMinLength  int                     `json:"shareLinkPasswordMinLength,omitempty"`
	UploadLinkPasswordRequired  bool                    `json:"uploadLinkPasswordRequired,omitempty"`
	UploadLinkPasswordMinLength int                     `json:"uploadLinkPasswordMinLength,omitempty"`
	OnlyOffice                  *OnlyOfficeSettings     `json:"onlyOffice,omitempty"`
	VideoTranscode              *VideoTranscodeSettings `json:"videoTranscode,omitempty"`
	Archive                     *ArchiveSettings        `json:"archive,omitempty"`
}

type OnlyOfficeSettings struct {
	Enable      bool   `json:"enable,omitempty"`
	ServerUrl   string `json:"serverUrl,omitempty"`
	TokenSecret string `json:"tokenSecret,omitempty"`
	CallbackUrl string `json:"callbackUrl,omitempty"`
}

type VideoTranscodeSettings struct {
	Enable     bool   `json:"enable,omitempty"`
	FfmpegPath string `json:"ffmpegPath,omitempty"`
	Threads    int    `json:"threads,omitempty"`
	Preset     string `json:"preset,omitempty"`
}

// ArchiveSettings 在线解压缩配置。
type ArchiveSettings struct {
	MaxConcurrentExtract int `json:"maxConcurrentExtract,omitempty"`
	ArchiveCmdTimeout    int `json:"archiveCmdTimeout,omitempty"`
	MaxArchiveFileCount  int `json:"maxArchiveFileCount,omitempty"`
	MaxArchiveTotalSize  int `json:"maxArchiveTotalSize,omitempty"`
	ArchiveCacheTTL      int `json:"archiveCacheTtl,omitempty"`
}
