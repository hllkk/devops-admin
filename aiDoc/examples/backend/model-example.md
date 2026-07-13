# Model 示例

## 这个文件负责什么

Model 负责定义数据库实体与持久化字段，是 Service 和数据库交互的基础。

## 什么时候应该这样写

- 新增一张业务表
- 为现有表补字段
- 需要定义 GORM 结构与关联关系

## 推荐写法示例

两种姿势——按「是否需要追溯操作人」选基座：

```go
package disk

import "github.com/hllkk/devops-admin/server/global"

// 内部表（append-only 系统记录）：生命周期基座 + 自定义主键
type File struct {
	ID   int64  `json:"id,string" gorm:"primaryKey;autoIncrement:false;comment:雪花主键"`
	global.OPS_MODEL
	Name   string `json:"name" gorm:"comment:文件名"`
	Size   int64  `json:"size" gorm:"comment:文件大小(字节)"`
	Status int    `json:"status" gorm:"default:1;comment:文件状态"`
}

// 对外业务实体（需追溯创建/更新人）：审计基座 + 业务命名主键
type FileShare struct {
	ShareId int64 `json:"shareId,string" gorm:"primaryKey;autoIncrement:false"`
	global.OPS_AUDIT_MODEL
	Url string `json:"url" gorm:"comment:分享链接"`
}
```

## 为什么这样写

- 内部表内嵌 `global.OPS_MODEL`（时间戳）；对外业务实体内嵌 `global.OPS_AUDIT_MODEL`（时间戳 + `CreateBy`/`UpdateBy`）
- 主键不在基座，由模型自定义：内部 `id`，对外用业务命名（`shareId` 等）
- 主键统一雪花 `int64` + `json:",string"`（防前端精度丢失），回调 `ops:snowflake_id` 自动填充
- `json` 标签用于接口输出，`gorm` 标签约束字段类型、默认值和注释
- 字段命名清晰、稳定，便于前后端保持一致

## 常见错误

- 缺少 `json` 或 `gorm` 标签
- 把仅用于请求或展示的字段直接写入数据库 model
- 同一个字段在前后端使用不同类型
- 忽略 `Status`、`ID`、时间字段这类高风险类型一致性问题

## 真实参考文件

- 对外业务实体（`OPS_AUDIT_MODEL`）：`server/model/system/sys_user.go`、`sys_role.go`
- 内部表（`OPS_MODEL`）：`server/model/system/sys_error.go`、`sys_jwt_blacklist.go`
