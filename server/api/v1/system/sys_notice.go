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

// NoticeApi 通知公告管理(对齐前端 /system/notice/* 资源)
type NoticeApi struct{}

// GetNoticeList
// @Tags      SysNotice
// @Summary   分页获取通知公告列表
// @Produce   application/json
// @Param     noticeTitle  query  string  false  "公告标题(模糊匹配)"
// @Param     noticeType   query  string  false  "公告类型(1通知 2公告)"
// @Param     pageNum      query  int     true   "页码"
// @Param     pageSize     query  int     true   "每页大小"
// @Success   200  {object}  response.Response{data=response.PageResult{rows=[]system.SysNotice},msg=string}
// @Router    /system/notice/list [get]
func (n *NoticeApi) GetNoticeList(c *gin.Context) {
	var q systemReq.NoticeSearch
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := noticeService.GetNoticeList(c.Request.Context(), q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("获取通知公告列表失败")
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

// CreateNotice
// @Tags      SysNotice
// @Summary   新增通知公告
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  systemReq.NoticeOperateParams  true  "通知公告信息"
// @Success   200   {object}  response.Response{data=bool,msg=string}
// @Router    /system/notice [post]
func (n *NoticeApi) CreateNotice(c *gin.Context) {
	var req systemReq.NoticeOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := noticeService.CreateNotice(c.Request.Context(), req, utils.GetUserID(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "新增成功", c)
}

// UpdateNotice
// @Tags      SysNotice
// @Summary   修改通知公告
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  systemReq.NoticeOperateParams  true  "通知公告信息(含 noticeId)"
// @Success   200   {object}  response.Response{data=bool,msg=string}
// @Router    /system/notice [put]
func (n *NoticeApi) UpdateNotice(c *gin.Context) {
	var req systemReq.NoticeOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := noticeService.UpdateNotice(c.Request.Context(), req, utils.GetUserID(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "修改成功", c)
}

// BatchDeleteNotice
// @Tags      SysNotice
// @Summary   批量删除通知公告
// @Produce   application/json
// @Param     ids  path  string  true  "公告ID列表(逗号分隔)"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /system/notice/{ids} [delete]
func (n *NoticeApi) BatchDeleteNotice(c *gin.Context) {
	ids := make([]int64, 0, 4)
	for s := range strings.SplitSeq(c.Param("ids"), ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			response.FailWithMessage("无效的公告ID: "+s, c)
			return
		}
		ids = append(ids, id)
	}
	if err := noticeService.DeleteNotice(c.Request.Context(), ids); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "删除成功", c)
}
