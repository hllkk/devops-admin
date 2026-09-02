package system

import (
	"github.com/gin-gonic/gin"

	"github.com/hllkk/devops-admin/server/model/common/response"
)

// WecomContactApi 企业微信通讯录同步接口(管理员,挂 PrivateGroup 走 Casbin 权限)。
type WecomContactApi struct{}

// SyncStructure 手动触发企业微信通讯录同步(异步):全量首同步含逐用户新建(bcrypt+事务),
// 耗时远超 HTTP 超时(前端默认 10s),故 goroutine 内执行、立即返回 started;结果经 syncStatus 轮询。
// 已有同步进行中则 started=false(不重复启动)。定时同步由定时任务 SyncWecomContact 直接调 Service。
//
// @Tags      Wecom
// @Summary   企业微信通讯录同步(异步启动)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=system.WecomSyncStatus,msg=string}  "启动状态(started/inProgress)"
// @Router    /system/wecom/syncStructure [post]
func (a *WecomContactApi) SyncStructure(c *gin.Context) {
	response.OkWithData(wecomContactService.StartSync(), c)
}

// SyncStatus 查询异步同步状态(进度/最近结果/错误)。
//
// @Tags      Wecom
// @Summary   企业微信通讯录同步状态查询
// @Produce   application/json
// @Success   200  {object}  response.Response{data=system.WecomSyncStatus,msg=string}  "同步状态"
// @Router    /system/wecom/syncStatus [get]
func (a *WecomContactApi) SyncStatus(c *gin.Context) {
	response.OkWithData(wecomContactService.SyncStatus(), c)
}
