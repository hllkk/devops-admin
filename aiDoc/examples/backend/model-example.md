# Model 示例

## 这个文件负责什么

Model 负责定义数据库实体与持久化字段，是 Service 和数据库交互的基础。

## 什么时候应该这样写

- 新增一张业务表
- 为现有表补字段
- 需要定义 GORM 结构与关联关系

## 推荐写法示例

两种姿势——按「是否需要追溯操作人」选基座（`OPS_MODEL` / `OPS_AUDIT_MODEL`）：

```go
package server

import "github.com/hllkk/devops-admin/server/global"

// 内部表（append-only 系统记录）：直接内嵌 OPS_MODEL（自带 ID 主键），无需再声明主键
type Host struct {
	global.OPS_MODEL
	HostName string `json:"hostName" gorm:"comment:主机名"`
	Port     int    `json:"port" gorm:"comment:端口"`
	Status   int    `json:"status" gorm:"default:1;comment:主机状态"`
}

// 对外业务实体（需追溯创建/更新人）：内嵌 OPS_AUDIT_MODEL + 自定义业务命名主键
type HostCredential struct {
	CredentialId int64 `json:"credentialId,string" gorm:"primarykey;comment:凭据ID"` // 雪花目标(autoIncrement:false+回调);当前 DB 自增
	global.OPS_AUDIT_MODEL
	Secret string `json:"secret" gorm:"comment:凭据密文"`
}
```

## 为什么这样写

- 基座三层在 `global/model.go`：`OPS_BASE`（CreateTime/UpdateTime/DeletedAt，无主键）/ `OPS_MODEL`（OPS_BASE + 自带 `ID` 主键，内部表直接复用）/ `OPS_AUDIT_MODEL`（OPS_BASE + CreateBy/UpdateBy，对外实体）
- 内部表内嵌 `OPS_MODEL`（自带主键，无需再声明）；对外业务实体内嵌 `OPS_AUDIT_MODEL` + 自定义业务命名主键（`credentialId` 等）
- 主键统一 `json:",string"`（防前端精度丢失）；雪花 `int64` + `ops:snowflake_id` 回调为**目标**，当前 `utils/snowflake` 待落地、走 DB 自增
- `json` 标签用于接口输出，`gorm` 标签约束字段类型、默认值和注释
- 字段命名清晰、稳定，便于前后端保持一致

## 常见错误

- 缺少 `json` 或 `gorm` 标签
- 把仅用于请求或展示的字段直接写入数据库 model
- 同一个字段在前后端使用不同类型
- 忽略 `Status`、`ID`、时间字段这类高风险类型一致性问题

## 真实参考文件

- 对外业务实体（`OPS_AUDIT_MODEL`）：`server/model/system/sys_user.go`（目前仅 `SysUser` 完成新基座改造；`SysRole` 等仍为旧形态，待改造）
- 内部表（`OPS_MODEL`）：`server/model/system/sys_error.go`、`sys_jwt_blacklist.go`
