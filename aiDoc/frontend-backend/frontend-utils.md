# 前端工具复用 (frontend-utils)

> TODO：待 `web/` scaffold 后据实补充。原则先行。

## 强制复用（禁止重复造轮子）
| 场景 | 用什么 |
|---|---|
| HTTP 请求 | `@sa/axios`（`createFlatRequest`），见 `frontend-rules.md` |
| 颜色 / 主题 token | `@sa/color` |
| 通用 hooks | `@sa/hooks`、`src/hooks/{common,business}` |
| 通用工具 | `@sa/utils`、`src/utils/` |
| 本地图标枚举 | `src/utils/icon.ts` 的 `getLocalIcons()` |
| 请求/响应类型 | `src/typings/api/` |

## 禁止
- 裸用 axios / 手写 fetch 封装
- 硬编码颜色、手写日期格式化、手写命名转换
