package system

import (
	"github.com/gin-gonic/gin"
)

// DictRouter 字典管理路由(对齐前端 /system/dict/type/* 与 /system/dict/data/* 资源)
type DictRouter struct{}

// InitDictRouter 字典相关路由挂在 PrivateGroup 下,鉴权与操作日志由该组全局中间件统一处理。
func (d *DictRouter) InitDictRouter(Router *gin.RouterGroup) {
	dictTypeRouter := Router.Group("system/dict/type")
	{
		dictTypeRouter.GET("list", dictApi.GetDictTypeList)           // 分页获取字典类型列表
		dictTypeRouter.GET("optionselect", dictApi.GetDictTypeOption) // 获取字典类型选择框列表
		dictTypeRouter.POST("", dictApi.CreateDictType)               // 新增字典类型
		dictTypeRouter.PUT("", dictApi.UpdateDictType)                // 修改字典类型
		dictTypeRouter.DELETE(":ids", dictApi.BatchDeleteDictType)    // 批量删除字典类型
	}
	dictDataRouter := Router.Group("system/dict/data")
	{
		dictDataRouter.GET("list", dictApi.GetDictDataList)              // 分页获取字典数据列表
		dictDataRouter.GET("type/:dictType", dictApi.GetDictDataByType)  // 按字典类型查字典数据(DictTag 渲染用)
		dictDataRouter.POST("", dictApi.CreateDictData)                  // 新增字典数据
		dictDataRouter.PUT("", dictApi.UpdateDictData)                   // 修改字典数据
		dictDataRouter.DELETE(":dictCodes", dictApi.BatchDeleteDictData) // 批量删除字典数据
	}
}
