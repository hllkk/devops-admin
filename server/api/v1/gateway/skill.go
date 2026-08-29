package gateway

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/hllkk/devops-admin/server/model/common/response"
	gatewayReq "github.com/hllkk/devops-admin/server/model/gateway/request"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/request"
)

// SkillApi Skill 管理(对齐前端 /gateway/skill/* 资源)
type SkillApi struct{}

// GetSkillList
// @Tags      GatewaySkill
// @Summary   分页获取Skill列表
// @Produce   application/json
// @Param     name         query  string  false  "名称/作者(模糊)"
// @Param     category     query  string  false  "分类(精确)"
// @Param     isActive     query  bool    false  "是否启用(精确)"
// @Param     isPublished  query  bool    false  "是否发布(精确)"
// @Param     pageNum      query  int     true   "页码"
// @Param     pageSize     query  int     true   "每页大小"
// @Success   200  {object}  response.Response{data=response.PageResult{rows=[]response.SkillView},msg=string}
// @Router    /gateway/skill/list [get]
func (a *SkillApi) GetSkillList(c *gin.Context) {
	var q gatewayReq.SkillSearch
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	request.NormalizeEmptyBoolQuery(c, &q)
	list, total, err := skillService.GetSkillList(c.Request.Context(), q)
	if err != nil {
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		Rows: list, Total: total, PageNum: q.PageNum, PageSize: q.PageSize,
	}, "获取成功", c)
}

// GetSkill
// @Tags      GatewaySkill
// @Summary   获取Skill详情
// @Produce   application/json
// @Param     id  path  int  true  "技能ID"
// @Success   200  {object}  response.Response{data=response.SkillView,msg=string}
// @Router    /gateway/skill/{id} [get]
func (a *SkillApi) GetSkill(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithMessage("无效的技能ID", c)
		return
	}
	view, err := skillService.GetSkill(c.Request.Context(), id)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(view, "获取成功", c)
}

// CreateSkill
// @Tags      GatewaySkill
// @Summary   注册Skill(仅元数据,zip包另走上传端点)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  gatewayReq.SkillOperateParams  true  "Skill参数"
// @Success   200  {object}  response.Response{data=response.SkillView,msg=string}
// @Router    /gateway/skill [post]
func (a *SkillApi) CreateSkill(c *gin.Context) {
	var req gatewayReq.SkillOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	view, err := skillService.CreateSkill(c.Request.Context(), req, utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(view, "创建成功", c)
}

// UpdateSkill
// @Tags      GatewaySkill
// @Summary   修改Skill元数据(发布配置走 publish；启停翻转联动授权对齐)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  gatewayReq.SkillOperateParams  true  "Skill参数(含 skillId)"
// @Success   200  {object}  response.Response{data=response.SkillView,msg=string}
// @Router    /gateway/skill [put]
func (a *SkillApi) UpdateSkill(c *gin.Context) {
	var req gatewayReq.SkillOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	view, err := skillService.UpdateSkill(c.Request.Context(), req, utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(view, "修改成功", c)
}

// DeleteSkills
// @Tags      GatewaySkill
// @Summary   批量删除Skill(回收主Key授权,删zip包;使用日志保留)
// @Produce   application/json
// @Param     ids  path  string  true  "技能ID(逗号分隔)"
// @Success   200  {object}  response.Response{msg=string}
// @Router    /gateway/skill/{ids} [delete]
func (a *SkillApi) DeleteSkills(c *gin.Context) {
	ids := make([]int64, 0, 4)
	for s := range strings.SplitSeq(c.Param("ids"), ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			response.FailWithMessage("无效的技能ID: "+s, c)
			return
		}
		ids = append(ids, id)
	}
	if err := skillService.DeleteSkills(c.Request.Context(), ids); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

// GetSkillPublish
// @Tags      GatewaySkill
// @Summary   获取Skill发布设置(含可见部门/用户回显)
// @Produce   application/json
// @Param     id  path  int  true  "技能ID"
// @Success   200  {object}  response.Response{data=response.SkillPublishView,msg=string}
// @Router    /gateway/skill/publish/{id} [get]
func (a *SkillApi) GetSkillPublish(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithMessage("无效的技能ID", c)
		return
	}
	view, err := skillService.GetSkillPublish(c.Request.Context(), id)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(view, "获取成功", c)
}

// PublishSkill
// @Tags      GatewaySkill
// @Summary   Skill发布设置(三档可见性+需审批;须先上传zip包)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  gatewayReq.SkillPublishParams  true  "发布参数"
// @Success   200  {object}  response.Response{msg=string}
// @Router    /gateway/skill/publish [put]
func (a *SkillApi) PublishSkill(c *gin.Context) {
	var req gatewayReq.SkillPublishParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := skillService.PublishSkill(c.Request.Context(), req, utils.GetUserID(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("发布设置已保存", c)
}

// UploadSkillPackage
// @Tags      GatewaySkill
// @Summary   上传/替换Skill zip包(multipart,≤100MB)
// @Accept    multipart/form-data
// @Produce   application/json
// @Param     id    path  int    true  "技能ID"
// @Param     file  formData file true  "zip文件"
// @Success   200  {object}  response.Response{data=response.SkillView,msg=string}
// @Router    /gateway/skill/{id}/package [post]
func (a *SkillApi) UploadSkillPackage(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithMessage("无效的技能ID", c)
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.FailWithMessage("请选择要上传的 zip 文件", c)
		return
	}
	view, err := skillService.UploadSkillPackage(c.Request.Context(), id, fileHeader, utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(view, "上传成功", c)
}

// DownloadSkill
// @Tags      GatewaySkill
// @Summary   下载Skill zip包(用户侧;需审批Skill须已授权;casbin 登录白名单)
// @Produce   application/octet-stream
// @Param     id  path  int  true  "技能ID"
// @Success   200  {file}  binary
// @Router    /gateway/skill/download/{id} [get]
func (a *SkillApi) DownloadSkill(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithMessage("无效的技能ID", c)
		return
	}
	filePath, originName, err := skillService.DownloadSkill(c.Request.Context(), id, utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	c.FileAttachment(filePath, originName)
}

// GetAvailableSkills
// @Tags      GatewaySkill
// @Summary   可授权Skill列表(管理端授权下拉,全量启用)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=[]response.AvailableSkillView,msg=string}
// @Router    /gateway/skill/available [get]
func (a *SkillApi) GetAvailableSkills(c *gin.Context) {
	list, err := skillService.GetAvailableSkills(c.Request.Context())
	if err != nil {
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(list, "获取成功", c)
}

// GetActiveSkills
// @Tags      GatewaySkill
// @Summary   用户侧可见Skill列表(广场,按发布可见性过滤;casbin 登录白名单)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=[]response.AvailableSkillView,msg=string}
// @Router    /gateway/skill/active [get]
func (a *SkillApi) GetActiveSkills(c *gin.Context) {
	list, err := skillService.GetActiveSkills(c.Request.Context(), utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(list, "获取成功", c)
}

// GetSkillUsageList
// @Tags      GatewaySkill
// @Summary   Skill使用日志分页(管理端,回填用户名/技能名)
// @Produce   application/json
// @Param     skillId    query  int     false  "技能ID(精确)"
// @Param     userId     query  int     false  "用户ID(精确)"
// @Param     action     query  string  false  "动作(精确)"
// @Param     pageNum    query  int     true   "页码"
// @Param     pageSize   query  int     true   "每页大小"
// @Success   200  {object}  response.Response{data=response.PageResult{rows=[]response.SkillUsageView},msg=string}
// @Router    /gateway/skill/usage/list [get]
func (a *SkillApi) GetSkillUsageList(c *gin.Context) {
	var q gatewayReq.SkillUsageSearch
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := skillService.GetSkillUsageList(c.Request.Context(), q)
	if err != nil {
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		Rows: list, Total: total, PageNum: q.PageNum, PageSize: q.PageSize,
	}, "获取成功", c)
}
