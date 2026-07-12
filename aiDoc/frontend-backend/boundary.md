# 前后端边界说明

## 归属边界

- 后端负责路由、参数校验、业务逻辑和响应结构
- 前端负责页面流程、交互体验、本地状态和展示层
- 共同行为通过明确的 API 契约协作，不通过隐式约定耦合

## 契约规则

- 保持统一响应结构：`{ code, data, msg }`
- 保持统一分页结构：`{ pageNum, pageSize, total, rows }`
- 字段名不要随意漂移
- 前后端字段类型必须保持一致
- 后端必须提供完整而准确的 Swagger 接口说明
- 前端接口调用应以实际 Swagger 与后端实现为准

## 主键 ID 契约

所有继承 `OPS_MODEL` 的表主键统一使用**雪花算法**生成的 `int64` ID（后端 `utils/snowflake` 自实现，GORM `BeforeCreate` 回调在主键为 0 时自动填充，三处 `OPS_DB` 赋值点均已注册）。

- **传输格式**：JSON 中 ID 以**字符串**传输（Go 端 `json:"ID,string"`），规避 JS `Number` 仅能安全表示到 2^53 的精度丢失（雪花 ID 通常 18~19 位十进制）。
- **前端约束**：所有 ID 字段一律按 `string` 收发，**禁止当 number 做数值运算或比较**；需要排序/比较时转 BigInt。
- **显式 ID 不被覆盖**：创建时若已指定主键，回调不会覆盖。

> 例外：GVA 基座认证链路（`BaseClaims.ID` / `Login.GetUserId` / `GetById.ID` 等）目前仍是独立的 `uint`/`int` 空壳，待用户表/登录实现时统一改为 int64，并解决 JWT token 内 ID 的精度问题（届时 claims 改用 string 存储）。

## 变更规则

- 涉及破坏性接口调整时，要先写清楚变更范围
- Swagger 或其他接口说明必须与真实实现一致
- 前端接口封装应继续放在 `web/src/service/api/` 或 `web/src/plugin/<name>/api/`
- 可复用逻辑优先复用 `web/src/utils/` 现有能力

## 完成前检查

跨前后端改动结束前，至少确认以下几点：

1. 后端响应结构仍然满足前端预期
2. 前端仍在使用正确的字段名和数据类型
3. 若契约发生了长期变化，对应说明已经补到 `aiDoc/`
