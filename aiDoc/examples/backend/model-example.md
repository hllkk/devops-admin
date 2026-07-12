# Model 示例

## 这个文件负责什么

Model 负责定义数据库实体与持久化字段，是 Service 和数据库交互的基础。

## 什么时候应该这样写

- 新增一张业务表
- 为现有表补字段
- 需要定义 GORM 结构与关联关系

## 推荐写法示例

```go
package disk

import "devops-admin/server/global"

type File struct {
	global.OPS_MODEL
	Name      string `json:"name" gorm:"comment:文件名"`
	Size      int64  `json:"size" gorm:"comment:文件大小(字节)"`
	Status    int    `json:"status" gorm:"default:1;comment:文件状态"`
	CreatorID uint   `json:"creatorId" gorm:"comment:创建人ID"`
}
```

## 为什么这样写

- 继承 `global.OPS_MODEL`，保持主键和时间字段风格一致
- `json` 标签用于接口输出
- `gorm` 标签用于约束字段类型、默认值和注释
- 字段命名尽量清晰、稳定，便于前后端保持一致

## 常见错误

- 缺少 `json` 或 `gorm` 标签
- 把仅用于请求或展示的字段直接写入数据库 model
- 同一个字段在前后端使用不同类型
- 忽略 `Status`、`ID`、时间字段这类高风险类型一致性问题

## 真实参考文件

> devops-admin 尚无业务代码；待网盘模块落地后在此补充。可对照 gin-vue-admin 的 `server/model/system/sys_user.go` 学习写法。
