package response

// VersionInfo 版本信息(GET /system/upgrade/version,「关于」弹窗展示)
type VersionInfo struct {
	AppName     string `json:"appName"`               // 应用名(global.AppName)
	Version     string `json:"version"`               // 当前版本号(global.Version,ldflags 注入)
	BuildTime   string `json:"buildTime"`             // 构建时间(RFC3339;裸构建为 unknown)
	Description string `json:"description"`           // 应用描述
}

// UpgradeCheckResult 检查更新结果(GET /system/upgrade/check)
type UpgradeCheckResult struct {
	CurrentVersion    string              `json:"currentVersion"`              // 当前运行版本(global.Version)
	HasUpdate         bool                `json:"hasUpdate"`                   // 是否有新版本
	Version           string              `json:"version,omitempty"`           // manifest 版本
	ChangeLog         string              `json:"changeLog,omitempty"`         // 更新内容(markdown)
	ReleaseTime       string              `json:"releaseTime,omitempty"`
	MinUpgradeVersion string              `json:"minUpgradeVersion,omitempty"` // 低于此版本需先升中间版(提示用)
	ForceUpgrade      bool                `json:"forceUpgrade"`
	Package           *UpgradePackageInfo `json:"package,omitempty"` // 选中的升级包(优先增量)
	Message           string              `json:"message"`           // 无更新/失败原因(用户可读)
}

// UpgradePackageInfo manifest 选中的升级包描述
type UpgradePackageInfo struct {
	Type      string `json:"type"`      // incr 增量 / full 全量
	URL       string `json:"url"`       // 相对发布服务器根路径
	Sha256    string `json:"sha256"`
	SizeBytes int64  `json:"sizeBytes"`
}

// UpgradeStartResult 触发升级结果(POST /system/upgrade/start)
type UpgradeStartResult struct {
	Accepted bool   `json:"accepted"`          // 是否已开始(进度轮询 /system/upgrade/status)
	Version  string `json:"version,omitempty"` // 目标版本
	Message  string `json:"message"`           // 未开始原因(已是最新/updater 拒绝等)
}

// UpgradeStateInfo 升级状态机(GET /system/upgrade/status,代理 updater 的 upgrade-state.json)
type UpgradeStateInfo struct {
	State      string `json:"state"`             // idle/downloading/verifying/unpacking/installing/success/failed/unreachable
	Progress   int    `json:"progress"`          // 0-100(downloading 阶段为下载百分比)
	Message    string `json:"message,omitempty"` // 进展/错误说明
	Version    string `json:"version,omitempty"` // 目标版本(终态时为已装版本)
	UpdateTime string `json:"updateTime,omitempty"`
}
