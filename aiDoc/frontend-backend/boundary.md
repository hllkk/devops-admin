# 前后端边界与契约 (boundary)

> 前后端唯一交集。冲突时以 `AGENT.MD` 与本文件为准。

## 统一响应
`{ code, data, msg }`
- **成功码 `code = 0`**（后端 `response.Ok*` 返回 0）
- ⚠️ SoybeanAdmin 默认 mock 常用 `code: 200`；**前端 `isBackendSuccess` 必须改为判 `code === 0`** 以对齐后端

## 统一分页
`{ page, pageSize, total, list }`

## 时间
ISO 8601 / RFC3339，如 `2026-07-09T10:00:00+08:00`

## 鉴权
JWT；请求头 `Authorization: Bearer <token>`（由 `@sa/axios` 的 `onRequest` 注入）

## 类型一致性（关键）
| Go | TypeScript |
|---|---|
| `*T` 指针 nil | `null` |
| `int64`（大 ID） | `string`（防精度丢失） |
| `int` / `int64`（小 ID） | `number` |
| `bool` | `boolean` |
| `time.Time` | ISO 8601 `string` |
| `[]byte` | base64 `string` |

- 前后端字段名一致（Go json tag ↔ TS 字段名）
- 枚举值前后端统一（建议数值或全大写字符串，避免大小写漂移）

## 文件 / 上传相关（网盘模块特有）
- 分片上传：前端切片 + hash 秒传，后端 RustFS 组装；接口约定见 `modules/business-modules.md`
- 大文件 ID 一律 `string` 传输
- 上传/下载走 RustFS 预签名 URL 或流式，**不**经业务 API 中转大二进制

## 跨栈变更
接口/字段/枚举变更时，**同步更新本文件与 `aiDoc/modules/business-modules.md`**，并通知对端。
