package system

import (
	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/middleware"
)

type UserRouter struct{}

func (s *UserRouter) InitUserRouter(Router *gin.RouterGroup) {
	userRouter := Router.Group("user").Use(middleware.OperationRecord())
	userRouterWithoutRecord := Router.Group("user")
	{
		userRouter.POST("admin_register", baseApi.Register) // 管理员注册账号
		// userRouter.POST("changePassword", baseApi.ChangePassword)         // 用户修改密码
		// userRouter.POST("setUserAuthority", baseApi.SetUserAuthority)     // 设置用户权限
		// userRouter.DELETE("deleteUser", baseApi.DeleteUser)               // 删除用户
		// userRouter.PUT("setUserInfo", baseApi.SetUserInfo)                // 设置用户信息
		// userRouter.PUT("setSelfInfo", baseApi.SetSelfInfo)                // 设置自身信息
		// userRouter.POST("setUserAuthorities", baseApi.SetUserAuthorities) // 设置用户权限组
		// userRouter.POST("setUserDepartments", baseApi.SetUserDepartments) // 设置用户归属部门
		// userRouter.POST("setUserPositions", baseApi.SetUserPositions)     // 设置用户岗位
		// userRouter.POST("resetPassword", baseApi.ResetPassword)           // 重置用户密码
		// userRouter.PUT("setSelfSetting", baseApi.SetSelfSetting)          // 用户界面配置
	}
	{
		userRouterWithoutRecord.GET("getUserInfo", baseApi.GetUserInfo) // 获取自身信息
	}

	// /system/user/* RESTful 接口(对齐前端 user.ts);鉴权与操作日志由 PrivateGroup 全局中间件统一处理。
	// GET ":userId" 放在 static 路由之后(param 兜底,gin static 优先匹配)。
	systemUserRouter := Router.Group("system/user")
	{
		systemUserRouter.GET("list", userApi.GetUserList)                   // 用户分页列表
		systemUserRouter.POST("export", userApi.ExportUser)                 // 导出用户(Excel)
		systemUserRouter.POST("importTemplate", userApi.ImportTemplate)     // 下载用户导入模板
		systemUserRouter.POST("importData", userApi.ImportUser)             // 导入用户(Excel)
		systemUserRouter.GET("list/dept/:deptId", userApi.GetDeptUserList)  // 部门下用户(负责人选择用)
		systemUserRouter.GET("deptTree", userApi.GetDeptTree)               // 部门树(复用部门模块)
		systemUserRouter.GET(":userId", userApi.GetUserDetail)              // 用户详情
		systemUserRouter.POST("", userApi.CreateUser)                       // 新增用户(含分配角色/岗位)
		systemUserRouter.PUT("", userApi.UpdateUser)                        // 修改用户(全量替换角色/岗位)
		systemUserRouter.PUT("changeStatus", userApi.UpdateUserStatus)      // 修改用户状态
		systemUserRouter.PUT("resetPwd", userApi.ResetUserPwd)              // 重置密码
		systemUserRouter.PUT("profile/updatePwd", userApi.ChangeMyPassword) // 当前用户自助改密(密码过期解锁入口)
		systemUserRouter.PUT("profile", userApi.UpdateMyProfile)            // 当前用户自助改基本资料(昵称/邮箱/手机号/性别)
		systemUserRouter.POST("profile/avatar", userApi.UpdateMyAvatar)     // 当前用户自助上传头像
		systemUserRouter.DELETE(":userIds", userApi.BatchDeleteUser)        // 批量删除用户
	}
}
