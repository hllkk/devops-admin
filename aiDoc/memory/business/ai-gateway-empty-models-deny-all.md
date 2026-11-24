# AI网关·空models密钥LiteLLM全量放行修复(哨兵deny-all)

> 需求日期：2026-11-23。关联：[[ai-gateway-overview]]

## 背景

AIHelms 2026-09-10 `e1e5336` 修复了"空 models 的 Key 在 LiteLLM 侧全量放行"漏洞（对标巡查发现）。核对 devops-admin 确认同样存在：LiteLLM 对 key 缺 `models` 字段/空列表的语义是**不限模型**，而非零授权。我们 `syncKeyToLitellm` 的 `KeyCreateReq.Models`（omitempty）空切片缺字段下发 → 空 models 的 Key（新用户无可见模型时的个人主 Key、清空授权的 Key）创建即在 LiteLLM 侧全量放行；update 分支 `len>0` 判定还导致清空授权时旧授权残留。client.go:379 的 MCP 侧注释当时就点了"同 models 空 slice 坑"，MCP 修了 models 一直没修。

## 已实现（build/vet/test 通过，LiteLLM 1.108.5 实测验证）

- `server/service/gateway/ai_key.go`：
  - 常量 `litellmNoDefaultModels = "no-default-models"` + 纯函数 `toLitellmModels`（空→[哨兵]；非空→过滤空串与混入的哨兵值，过滤后空仍落哨兵；恒非空保证 create/update 两路判定都下发）。
  - `syncKeyToLitellm` 接入投影（唯一收口，CreateKey/UpdateKey 全部经此）。
  - `HealEmptyKeyModels(db)` 启动期存量自愈：查 litellm_key_id 非空且 models 空的行重推哨兵；DB models 保持空（可见模型自愈差集仍会回加），每次启动幂等重推兼一致性纠偏；LiteLLM 未配置直通；总超时 1min 防 LiteLLM 未就绪拖慢启动。
- `server/initialize/gorm.go`：`BackfillAiKeyHashes` 后挂 `HealEmptyKeyModels`（项目启动期幂等兜底惯例位）。
- `server/service/gateway/ai_key_litellm_models_test.go`：`TestToLitellmModels` 三场景（空→哨兵/正常透传/哨兵空串过滤）。

## 运行时实测（dev litellm 1.108.5，2026-11-23）

- 漏洞复现：`/key/generate` 不带 models 的 key，`/v1/models` 可见全部模型（与 master 一致），调用 glm-5.3 穿透网关授权层到达上游（报错来自供应商 AccessDenied，证明 LiteLLM 已放行）。
- 哨兵验证：`models=["no-default-models"]` 的 key，`/v1/models` 返回空列表，调用在 LiteLLM 网关层直接拒绝。
- dev 存量：2 个空 models 个人主 Key（LiteLLM 侧当前全量放行），待下次部署启动自愈纠正；prod 同机制。

## 设计决策

- **哨兵值 `no-default-models`**：LiteLLM 社区约定（AIHelms 同款），网关层拒绝、模型列表为空，比 max_budget=0 更精准（不动预算语义）。
- **投影收口在 syncKeyToLitellm**：单点修复覆盖创建/编辑/启停/级联/发布授权全部路径。
- **DB 不落哨兵**：哨兵只存在于 LiteLLM 侧，平台 models 字段语义不变，主 Key 可见模型自愈差集机制不受影响。

## 关联

- 蓝本：AIHelms `e1e5336`（`_to_litellm_models` + `LITELLM_NO_DEFAULT_MODELS`）
