package gateway

import "github.com/gin-gonic/gin"

// SkillRouter Skill 管理路由(对齐前端 /gateway/skill/* 资源)
type SkillRouter struct{}

// InitSkillRouter 挂在 PrivateGroup，鉴权/操作日志/数据权限由该组全局中间件统一处理。
// active 与 download 为用户侧接口(casbin 登录白名单，见 middleware/casbin_rbac.go)。
func (s *SkillRouter) InitSkillRouter(Router *gin.RouterGroup) {
	r := Router.Group("gateway/skill")
	{
		r.GET("list", skillApi.GetSkillList)               // 分页获取Skill列表
		r.GET(":id", skillApi.GetSkill)                    // Skill详情
		r.POST("", skillApi.CreateSkill)                   // 注册Skill(元数据)
		r.PUT("", skillApi.UpdateSkill)                    // 修改Skill元数据
		r.DELETE(":ids", skillApi.DeleteSkills)            // 批量删除
		r.PUT("publish", skillApi.PublishSkill)            // 发布设置(三档可见性+审批)
		r.GET("publish/:id", skillApi.GetSkillPublish)     // 发布设置回显(含可见部门/用户)
		r.POST(":id/package", skillApi.UploadSkillPackage) // 上传/替换zip包(multipart)
		// 用户侧(广场)：登录白名单，不经菜单授权
		r.GET("available", skillApi.GetAvailableSkills)  // 管理端授权下拉
		r.GET("active", skillApi.GetActiveSkills)        // 用户侧可见列表(广场)
		r.GET("download/:id", skillApi.DownloadSkill)    // 下载zip包(登录态+授权校验;静态段在前,casbin 白名单前缀匹配)
		// 管理端使用日志
		r.GET("usage/list", skillApi.GetSkillUsageList) // 使用日志分页
	}
}
