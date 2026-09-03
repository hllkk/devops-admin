package config

// Upgrade 在线升级配置（发布服务器与升级执行器对接；生产由 .env 覆盖，config.yaml 留空值）
type Upgrade struct {
	// UpdateServerUrl 发布服务器根地址（其下 manifest.json 描述最新版本与升级包；空=不启用，检查更新仅提示未配置）
	UpdateServerUrl string `mapstructure:"update-server-url" json:"update-server-url" yaml:"update-server-url"`
	// UpdaterToken 升级执行器(updater sidecar)写接口鉴权 token（与 .env 的 UPDATER_TOKEN 同源；空=不鉴权，仅限可信内网）
	UpdaterToken string `mapstructure:"updater-token" json:"updater-token" yaml:"updater-token"`
}
