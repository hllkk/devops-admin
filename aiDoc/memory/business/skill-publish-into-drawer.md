# Skill 发布能力合并进编辑抽屉（发布+领用审批双复选框）

- 日期：2026-08-30
- 状态：已实现

## 需求

用户要求：编辑 Skill 时提供两个复选框——「发布」与「领用前需要审批」（后者依赖前者勾选）；Skill 默认发布到全员市场，**不做针对部门/个人的可见性选择**。

## 实现口径

- **抽屉双复选框**（新建/编辑都显示）：`isPublish`/`requireApproval` 独立 ref（不进 `SkillOperateParams` 契约）；`watch(isPublish)` 取消发布时联动清掉审批勾选；未发布时审批复选框 `disabled`。
- **提交串联**（后端零改动）：
  - edit：`UpdateSkill` → `syncPublish`（发布配置与回显快照 `publishSnap` 对比，变化才调）
  - add：`CreateSkill` → zip 上传 → `syncPublish`
  - `syncPublish` 固定传 `visibilityType: 'all' + departmentIds/userIds 空数组`（后端空可见性归 all）；失败 warning 返回 false，调用方照样 `emit('submitted')` 反映已保存现状。
- **前置校验**：勾发布但无包（add 看 `pendingFile`/edit 看 `packageInfo`）→ 提交前拦截提示 `publish.needPackage`。
- **移除旧入口**：列表「发布」按钮 + `skill-publish-dialog.vue`（三档可见性弹窗）删除——用户明确不需要部门/个人档位。后端 `PublishSkill`/`GetSkillPublish` 端点与三档能力保留未动，未来要恢复只需重建前端入口。

## 关键决策

- 可见性简化为 all 是用户明确取舍；`gateway_skill_visibility`/`_user` 投影表与授权对齐逻辑后端原样保留（all 档天然无投影行，零冗余）。

相关：[[skill-aihelms-alignment]]
