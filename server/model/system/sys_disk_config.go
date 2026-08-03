package system

import "github.com/hllkk/devops-admin/server/global"

// SysDiskConfig 网盘配置(单行表 id=1,启动加载入内存缓存,保存即热更新;对齐前端 DiskSettingConfig)
//
// 字段分四段:
//   - 基础存储:maxUploadSize/maxUploadSizeUnit/storageQuota/storageQuotaUnit/allowedExtensions/blockedExtensions/recycleBinRetentionDays
//   - 展示:diskName/diskLogo
//   - OnlyOffice:onlyOfficeEnabled/onlyOfficeServerUrl/onlyOfficeTokenSecret/onlyOfficeCallbackUrl
type SysDiskConfig struct {
	global.OPS_MODEL
	// 基础存储配置
	MaxUploadSize           float64 `json:"maxUploadSize" gorm:"default:5;comment:最大上传大小(数值)"`
	MaxUploadSizeUnit       string  `json:"maxUploadSizeUnit" gorm:"default:GB;comment:上传大小单位(MB/GB/TB)"`
	StorageQuota            float64 `json:"storageQuota" gorm:"default:1;comment:默认存储配额(数值)"`
	StorageQuotaUnit        string  `json:"storageQuotaUnit" gorm:"default:GB;comment:配额单位(MB/GB/TB)"`
	AllowedExtensions       string  `json:"allowedExtensions" gorm:"comment:允许上传扩展名(逗号分隔,空=允许全部)"`
	BlockedExtensions       string  `json:"blockedExtensions" gorm:"comment:禁止上传扩展名(逗号分隔,优先级高于允许)"`
	RecycleBinRetentionDays int     `json:"recycleBinRetentionDays" gorm:"default:30;comment:回收站自动清理天数"`
	MaxConcurrentUploads    int     `json:"maxConcurrentUploads" gorm:"default:3;comment:最大同时上传文件数(0=不限)"`
	MaxChunkConcurrency     int     `json:"maxChunkConcurrency" gorm:"default:0;comment:单文件分片并发数(0=按设备自适应)"`
	MaxChunkRetries         int     `json:"maxChunkRetries" gorm:"default:3;comment:单分片上传失败重试次数(0=不重试)"`
	// 展示配置
	DiskName string `json:"diskName" gorm:"default:DevOps Disk;comment:网盘名称"`
	DiskLogo string `json:"diskLogo" gorm:"comment:Logo URL"`
	// OnlyOffice 协同编辑
	OnlyOfficeEnabled     bool   `json:"onlyOfficeEnabled" gorm:"default:false;comment:启用OnlyOffice"`
	OnlyOfficeServerUrl   string `json:"onlyOfficeServerUrl" gorm:"comment:Document Server地址"`
	OnlyOfficeTokenSecret string `json:"onlyOfficeTokenSecret" gorm:"comment:JWT签名密钥"`
	OnlyOfficeCallbackUrl string `json:"onlyOfficeCallbackUrl" gorm:"comment:回调地址(OnlyOffice容器可访问的后端地址)"`
}

func (SysDiskConfig) TableName() string {
	return "sys_disk_config"
}

// DefaultDiskConfig 返回默认网盘配置,调用方负责设 id=1
func DefaultDiskConfig() SysDiskConfig {
	return SysDiskConfig{
		MaxUploadSize:           5,
		MaxUploadSizeUnit:       "GB",
		StorageQuota:            1,
		StorageQuotaUnit:        "GB",
		AllowedExtensions:       "",
		BlockedExtensions:       "",
		RecycleBinRetentionDays: 30,
		MaxConcurrentUploads:    3,
		MaxChunkConcurrency:     0,
		MaxChunkRetries:         3,
		DiskName:                "",
		DiskLogo:                "",
		OnlyOfficeEnabled:       false,
		OnlyOfficeServerUrl:     "",
		OnlyOfficeTokenSecret:   "",
		OnlyOfficeCallbackUrl:   "",
	}
}
