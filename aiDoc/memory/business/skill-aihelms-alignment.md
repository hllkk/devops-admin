# Skill 管理对标 AIHelms 借鉴三项（新建带 zip / SKILL.md 校验 / 分类受控下拉）

- 日期：2026-08-30
- 状态：已实现

## 需求

对比本地 AIHelms 的 Skill 管理后借鉴三点（用户多选确认）：新建 Skill 一步带 zip 包、zip 内容校验 SKILL.md、分类受控下拉。

## 实现口径

1. **新建带 zip（前端串联方案，用户确认）**：新建抽屉 NUpload `:default-upload="false"` 暂存 File → `CreateSkill` 成功拿 skillId → 自动串联调现有 `UploadSkillPackage`。后端零改动。传包失败 warning"已创建请在编辑中重传"并照样 `emit('submitted')`（记录已建，列表刷新可见）。
2. **SKILL.md 校验**：`skill_payload.go` 新增 `ValidateSkillZipStructure`（`zip.OpenReader` 校验任意层级含 SKILL.md，`strings.EqualFold` 大小写不敏感，非 zip 伪装文件报"无法解析"）；`UploadSkillPackage` 落盘后校验、失败删临时文件拒绝入库。单测覆盖 4 组用例。
3. **分类受控（轻量版，不建分类表）**：后端 `GET /gateway/skill/categories`（distinct 非空升序，软删由基座 `gorm.DeletedAt` 自动过滤）；前端 NSelect `filterable + tag`——现有值可选、新值可输。casbin 由菜单 api_prefix `/gateway/skill/*` 通配覆盖，无需改菜单。

## 明确不借鉴

安全审查集成（skillspector）、install-info/Agent token 下载端点 → P3 单独立项；AIHelms 上传全量读内存/无大小校验是坑，保持本项目流式落盘+100MB 上限。

相关：[[skill-publish-into-drawer]]、aiDoc 的 [[ai-gateway-resource-application]]
