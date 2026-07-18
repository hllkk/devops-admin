package system

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/response"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// PostApi 岗位管理(对齐前端 /system/post/* 资源)
type PostApi struct{}

// GetPostList
// @Tags      SysPost
// @Summary   分页获取岗位列表
// @Produce   application/json
// @Param     postCode      query  string  false  "岗位编码(模糊匹配)"
// @Param     postName      query  string  false  "岗位名称(模糊匹配)"
// @Param     status        query  string  false  "岗位状态(精确 0正常1停用)"
// @Param     belongDeptId  query  int     false  "归属部门ID(左侧部门树点击过滤)"
// @Param     pageNum       query  int     true   "页码"
// @Param     pageSize      query  int     true   "每页大小"
// @Success   200  {object}  response.Response{data=response.PageResult{rows=[]system.SysPost},msg=string}
// @Router    /system/post/list [get]
func (p *PostApi) GetPostList(c *gin.Context) {
	var q systemReq.PostSearch
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := postService.GetPostList(c.Request.Context(), q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("获取岗位列表失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		Rows:     list,
		Total:    total,
		PageNum:  q.PageNum,
		PageSize: q.PageSize,
	}, "获取成功", c)
}

// CreatePost
// @Tags      SysPost
// @Summary   新增岗位
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  systemReq.PostOperateParams  true  "岗位信息"
// @Success   200   {object}  response.Response{data=bool,msg=string}
// @Router    /system/post [post]
func (p *PostApi) CreatePost(c *gin.Context) {
	var req systemReq.PostOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := postService.CreatePost(c.Request.Context(), req, utils.GetUserID(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "新增成功", c)
}

// UpdatePost
// @Tags      SysPost
// @Summary   修改岗位
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  systemReq.PostOperateParams  true  "岗位信息(含 postId)"
// @Success   200   {object}  response.Response{data=bool,msg=string}
// @Router    /system/post [put]
func (p *PostApi) UpdatePost(c *gin.Context) {
	var req systemReq.PostOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := postService.UpdatePost(c.Request.Context(), req, utils.GetUserID(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "修改成功", c)
}

// BatchDeletePost
// @Tags      SysPost
// @Summary   批量删除岗位
// @Produce   application/json
// @Param     ids  path  string  true  "岗位ID列表(逗号分隔)"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /system/post/{ids} [delete]
func (p *PostApi) BatchDeletePost(c *gin.Context) {
	ids := make([]int64, 0, 4)
	for s := range strings.SplitSeq(c.Param("ids"), ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			response.FailWithMessage("无效的岗位ID: "+s, c)
			return
		}
		ids = append(ids, n)
	}
	if err := postService.DeletePost(c.Request.Context(), ids); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "删除成功", c)
}

// GetPostOption
// @Tags      SysPost
// @Summary   获取岗位选择框列表
// @Produce   application/json
// @Param     deptId  query  int  false  "部门ID(限定该部门下启用岗位)"
// @Success   200  {object}  response.Response{data=[]system.SysPost,msg=string}
// @Router    /system/post/optionselect [get]
func (p *PostApi) GetPostOption(c *gin.Context) {
	var q struct {
		DeptId int64 `form:"deptId"`
	}
	_ = c.ShouldBindQuery(&q)
	list, err := postService.GetPostOptionList(c.Request.Context(), q.DeptId)
	if err != nil {
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(list, "获取成功", c)
}

// GetPostDeptTree
// @Tags      SysPost
// @Summary   获取岗位页部门树
// @Produce   application/json
// @Success   200  {object}  response.Response{data=[]system.DeptTreeNode,msg=string}
// @Router    /system/post/deptTree [get]
func (p *PostApi) GetPostDeptTree(c *gin.Context) {
	tree, err := postService.GetDeptTree(c.Request.Context())
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("获取岗位部门树失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(tree, "获取成功", c)
}
