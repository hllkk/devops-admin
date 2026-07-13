package system

type ServiceGroup struct {
	SysErrorService
	InitDBService
	CasbinService
	UserService
}
