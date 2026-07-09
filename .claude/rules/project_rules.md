---
tool: claude
role: compatibility-adapter
canonical_source: /AGENT.MD
structured_context: /aiDoc
---

# Claude 规则适配层

本文件只用于兼容 Claude 的自动加载路径，**不承载项目级规则**。

## 真实规则入口（按顺序读取）

1. `/AGENT.MD`
2. `/aiDoc/README.md`
3. 按当前任务读取相关子目录：
   - `server/` 工作 → `aiDoc/modules/backend-layer-rules.md` + `aiDoc/modules/business-modules.md`
   - `web/` 工作 → `aiDoc/frontend-backend/frontend-rules.md` + `frontend-utils.md`
   - 跨端 / 接口 → `aiDoc/frontend-backend/boundary.md`
   - 新增某层文件前 → `aiDoc/examples/`
4. 用户提出业务需求 → 写 `aiDoc/memory/business/` + 更新 `demand-index.md`

## 适配层约束

- **不要在这里扩写项目级规则**
- 规则变更时，先改 `/AGENT.MD` 与 `/aiDoc/`，本文件保持薄适配层
- 代码读取约束以 `/AGENT.MD` 为准：**无论什么情况，都不要直接读取 `node_modules/`**
- 若本文件与 `AGENT.MD` 冲突，以 `AGENT.MD` 为准
