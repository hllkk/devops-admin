# AI 网关·发布可见范围第四档 mixed（部门+用户混合）

- 日期：2026-09-01
- 状态：已实现（go build/vet/test + vue-tsc + eslint 通过，未浏览器点触验证）
- 反向链接：[[ai-gateway-model-publish]]、[[ai-gateway-mcp-server]]、[[ai-gateway-skill]]

## 需求

发布设置可见范围此前三档单选（all/selected/user），无法"既指定部门又指定用户"（例：信息部某用户 + 技术部整部）。用户要求模型/MCP/Skill 三资源同批支持混合授权。

## 可行性关键（为什么成本低）

查询侧 `visibleModelScope`（及 MCP/Skill 同构）本来就是 `all 直通 OR EXISTS(部门投影) OR EXISTS(用户投影)` 的 **OR 语义，不看 visibility_type 具体档位**——两张投影表同时有行即并集可见。因此以下消费链零改动自动兼容：`GetActiveModels`（广场/home）、`visibleModelKeys`（主 Key 自愈）、资源申请可见性校验（resource_application.go）、`GetModelPublish` 回显（本就无条件查两张表）。**限制只在写入侧**（表单单选 + Publish 按档写一张投影表 + mainKeyScopeOf 单档 switch）。

## 已实现

### 后端（增量第四档，存量三档零迁移零语义变化）

- `VisibilityTypeMixed = "mixed"` 常量（model.go，三资源共用）；三表 `visibility_type` 列注释补 mixed。
- `visibilityUsesDept/visibilityUsesUser(visibility)` helper（ai_key.go，放 mainKeyScopeOf 旁）：selected=部门表、user=用户表、mixed=两张。
- `mainKeyScopeOf` 加 mixed 分支：selected 块 OR user 块按列表非空动态拼接；双空给 `1 = 0`（应持有空集=全量回收口径）。
- `PublishModel/PublishMCPServer/PublishSkill`：取值集合加 mixed；mixed+发布时两列表**合计至少一项**（宽容校验——只填一类行为等同单档，OR 语义下无害）；投影写入改 `visibilityUsesDept/User && len>0`（gorm Create 空 slice 报错，须空表跳过）。
- `alignMCPAuthorization/alignSkillAuthorization`（启停/更新路径重授）：读投影表条件从 `== Selected/== User` 改为 `visibilityUsesDept/User`，mixed 档两张都读。

### 前端

- `model-publish-dialog.vue` / `mcp-publish-dialog.vue`：radio 第四项「部门+用户」；mixed 档下部门树选与用户多选**同时显示**；`handleVisibilityChange` mixed 保留两侧已选、切出清对应侧；校验 mixed 时两列表合计至少一项（两个 validator 互查对方）。
- **Skill 前端无改动**：skill-operate-drawer 发布是简化版（固定 `visibilityType:'all'`，不暴露可见范围 UI，`fetchGetSkillPublish` 无调用方），后端 mixed 能力对它无感；`SkillPublishParams` 类型已同步四值。
- i18n：`visibilityMixed`/`mixedRequired` 两键 ×（model/mcp 两段）三处同步；`gateway.api.d.ts` visibilityType 联合类型全量加 mixed（顺手修正 Model 视图旧两值定义）。

## 设计决策

- **新增第四档而非合并 selected/user**：合并要动存量 user 档语义+数据迁移；第四档完全增量。
- **合计至少一项而非两项必填**：管理员从 selected 切 mixed 追加用户不必重填部门；单侧空在查询/授权 OR 语义下行为正确。
- **授权口径**：mixed = selected∪user 并集（部门成员个人主 Key+部门主 Key+指定用户个人主 Key），与可见口径对称；回收半边自动复用。

## 验证缺口

- 未做浏览器点触验证（发布 mixed → 广场可见 → 主 Key 自动授权 → 切回单档回收）。
