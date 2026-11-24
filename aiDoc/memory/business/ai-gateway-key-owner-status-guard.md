# AI网关·Key启用/创建校验归属方状态(对标AIHelms授权闭环)

> 需求日期：2026-11-23。关联：[[ai-gateway-user-key-cascade]]、[[ai-gateway-empty-models-deny-all]]

## 背景

AIHelms 2026-09-10 `e1e5336` 补了"停用用户的 Key 不能启用"闸门（`toggle_key` 校验 owner 状态）。对 devops-admin 逐项核对 AIHelms 近期授权一致性 5 项，4 项已规避、1 项真洞：

| 核对项 | 结论 |
|---|---|
| 取消发布/转需审批收回主 Key 授权 | 已规避(model.go 发布管线三态收放,比 AIHelms 早) |
| 资源申请失效(下架后旧审批) | 已规避(approvedApplicationModelKeys JOIN 过滤 is_active+is_published,查询侧等价 invalidate) |
| Key 启用时补齐停用期间积压授权 | 已规避(loadMainKey 幂等自愈差集,注释明确规避 AIHelms 同坑) |
| 部署编辑重算已发 Key 授权 | 已规避(UpdateDeployment 禁改模型归属,无授权重建逻辑) |
| **启用 Key 校验归属方状态** | **存在洞(本次修复)** |

洞的机理：用户停用级联(SyncUserKeysActive)只在状态变化瞬间触发；此后管理员仍可单独启用其 Key(is_active=true+清 disabled_by_cascade)或给停用用户单独建场景 Key(ensureOwnerExists 只查存在)——明文 Key 直连网关数据面照常可用(管理面登录挡不住数据面)。批量建主 Key/批量场景 Key 已有停用→失败明细(classifyBatchTargets/BatchCreateSceneKeys 内置 Status!="0" 判定),单建与启用两入口漏了。

## 已实现（build/vet/test 通过）

- `server/service/gateway/ai_key.go`：
  - `ensureOwnerExists` 升级改名 `ensureOwnerUsable`：存在 + status="0"(user/dept 对称,部门也有停用状态)；错误信息"归属用户/部门不存在或已停用"；软删计不存在。
  - `CreateSceneKey` 调用点换成 ensureOwnerUsable(批量两条路径经此也双保险)。
  - `UpdateAiKey` 启停分支：启用方向(req.IsActive=true)先 ensureOwnerUsable,停用归属的 Key 拒绝启用；停用方向不拦(停用恒合法)。

## 设计决策

- **闸门收口在 service 层两入口**(单建+启用)，批量路径已有分类判定不重复加。
- **启用方向才校验**：已启用 Key 改其他字段不进分支；轮换(RotateAiKey)不改启用态、停用 Key 轮换后仍停用(max_budget=0 随 !IsActive 下发)，不拦。
- **部门对称校验**：部门停用虽无 Key 级联机制,启用/建部门 Key 同样拦,防同型绕过。

## 关联

- 蓝本：AIHelms `e1e5336`（toggle_key 校验 + initialize_pending_model_access）
