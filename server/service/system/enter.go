package system

type ServiceGroup struct {
	SysErrorService
	DataAccessLogService
	SysOperLogService
	DataScopeService
	InitDBService
	SecurityConfigService
	CaptchaService
	UserService
	LoginLogService
	DictTypeService
	DictDataService
	PostService
}
