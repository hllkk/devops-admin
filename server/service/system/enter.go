package system

type ServiceGroup struct {
	SysErrorService
	InitDBService
	CasbinService
	UserService
	CaptchaService
	LoginLogService
	SettingService
}
