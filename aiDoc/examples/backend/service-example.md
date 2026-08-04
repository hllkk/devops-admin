# Service 示例

## 这个文件负责什么

Service 层负责业务逻辑、数据库查询、事务控制和数据拼装，不负责 HTTP 参数绑定和响应输出。

## 什么时候应该这样写

- 新增列表查询、创建、更新、删除逻辑
- 需要事务操作
- 需要对 model 和 request 做转换

## 推荐写法示例

```go
package server

import (
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/server"
	serverReq "github.com/hllkk/devops-admin/server/model/server/request"
)

type HostService struct{}

func (s *HostService) GetHostList(info serverReq.HostSearch) (list []server.Host, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)

	db := global.GVA_DB.Model(&server.Host{})
	if info.HostName != "" {
		db = db.Where("host_name LIKE ?", "%"+info.HostName+"%")
	}
	if info.Status != nil {
		db = db.Where("status = ?", *info.Status)
	}

	err = db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = db.Limit(limit).Offset(offset).Order("id desc").Find(&list).Error
	return list, total, err
}
```

## 为什么这样写

- 只返回业务结果和 `error`，便于 API 层统一处理
- 直接使用 `global.GVA_DB` 或事务对象，保持项目一致性
- 查询条件尽量在 Service 层收敛，不散落到 API 或 Router

## 常见错误

- 在 Service 层接触 `gin.Context`
- 在 Service 层直接拼 `response.OkWith...`
- 把参数绑定、权限判断、响应格式化和业务逻辑混在一起
- 涉及多表更新却不使用事务

## 真实参考文件

> devops-admin 尚无业务代码；待服务器模块落地后在此补充。可对照 gin-vue-admin 的 `server/service/system/sys_user.go` 学习写法。
