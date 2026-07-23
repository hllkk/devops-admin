package system

type ServiceGroup struct {
	SysErrorService
	DataAccessLogService
	SysOperLogService
	DataScopeService
	InitDBService
	SecurityConfigService
	GeneralConfigService
	LdapConfigService
	NotifyConfigService
	DiskConfigService
	AuthConfigService
	CaptchaService
	UserService
	LoginLogService
	DictTypeService
	DictDataService
	PostService
	DepartmentService
	MenuService
	RoleService
	NoticeService
	SettingService
	AiService
	TimedTaskService
	SocialService
}
