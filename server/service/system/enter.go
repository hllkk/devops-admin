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
}
