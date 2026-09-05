package gateway

import "github.com/gin-gonic/gin"

// SkillRouter Skill 管理路由(对齐前端 /gateway/skill/* 资源)
type SkillRouter struct{}

// InitSkillRouter 管理接口挂 PrivateGroup，鉴权/操作日志/数据权限由该组全局中间件统一处理。
// active/download/install-info 为用户侧接口(casbin 登录白名单，见 middleware/casbin_rbac.go)。
// agent/:id/zip 挂 PublicGroup(Agent 无登录态，AiKey Bearer/token 自鉴权，
// 对齐 InitRouteRouter 双组拆分先例)。
func (s *SkillRouter) InitSkillRouter(Router *gin.RouterGroup, PublicRouter *gin.RouterGroup) {
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
		r.GET("package/:id", skillApi.AdminDownloadSkill)  // 管理端下载zip包(不走用户侧发布/授权校验,经菜单授权)
		// 用户侧(广场)：登录白名单，不经菜单授权
		r.GET("available", skillApi.GetAvailableSkills)    // 管理端授权下拉
		r.GET("active", skillApi.GetActiveSkills)          // 用户侧可见列表(广场)
		r.GET("download/:id", skillApi.DownloadSkill)      // 下载zip包(登录态+授权校验;静态段在前,casbin 白名单前缀匹配)
		r.GET("install-info/:id", skillApi.GetSkillInstallInfo) // Agent接入信息(curl命令+安装提示词;登录白名单)
		// 管理端使用日志
		r.GET("usage/list", skillApi.GetSkillUsageList) // 使用日志分页
		r.GET("categories", skillApi.GetSkillCategories) // 分类去重列表(下拉受控;静态段与:id共存,gin静态优先)
	}
	pub := PublicRouter.Group("gateway/skill")
	{
		// Agent 直连下载(AI Key 自鉴权，无 JWT/casbin；静态段在前与 download 先例同构)
		pub.GET("agent/:id/zip", skillApi.AgentDownloadSkill)
	}
}
