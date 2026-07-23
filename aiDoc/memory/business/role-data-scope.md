# 角色数据权限(datascope)前后端集成

> 2026-07-22。前端先行做了角色 datascope 分配抽屉(role-data-scope-drawer.vue)+列表展示,后端按「前端对齐后端 5 档」路线集成。

## 背景:两套档位体系冲突

- 前端 RuoYi 6 档:1全部/2自定义/3本部门/4本部门及以下/5仅本人/6部门及以下或本人,**字符串**类型
- 后端引擎 datascope.go 自实现 5 档:1全部/2本部门及以下/3本部门/4仅本人/5自定义,**int**类型
- 编号语义冲突:前端「自定义」=2,后端「自定义」=5;BuildIdentity 只在 Scope==5 取 CustomDeptIDs,前端传 2 当自定义→引擎按「本部门及以下」处理且不取部门集,完全失效
- 前端调的 /system/role/dataScope、GET /system/role/deptTree/:roleId 后端未注册→404

## 决策:前端对齐后端 5 档(不动引擎)

datascope 引擎 BuildIdentity 强依赖编号,改它影响数据权限全链路;前端刚加无存量依赖,改前端语义成本最低、风险最低。

## 前端改动

- constants/business.ts: dataScopeRecord 改 5 档对齐后端语义,去掉 '6'
- system.api.d.ts: DataScope 类型 '1'..'5'(去 '6')+注释更新
- role-data-scope-drawer.vue: 自定义档触发条件 '2'→'5'(handleSubmit 提交 deptIds + template v-if 两处)
- 未新增 i18n key(dataScopeRecord 是硬编码中文常量,drawer label 同模式),不涉三处同步

## 后端改动

- SysRole(model/system/sys_role.go): DataScope json 加 `,string`(JSON 边界字符串传输,引擎内部仍 int);新增 DeptCheckStrictly 字段(与 MenuCheckStrictly 对称,部门树回显用)
- RoleOperateParams(request/sys_role.go): 加 DataScope(int `,string`)/DeptIds([]Int64String)/DeptCheckStrictly(共用参数体,create/update 不消费,仅 dataScope 接口用)
- RoleService.UpdateRoleDataScope: 超管角色(SuperAdmin)禁改防降级;事务更新 data_scope/dept_check_strictly;档位5(自定义)全量替换 sys_role_departments,非5清空(saveRoleDepartments)
- RoleService.GetRoleDeptTreeSelect: 复用 DepartmentService.GetDeptTree 取启用部门树 + Pluck sys_role_departments 全部 deptId(**忠实往返全部**,不学菜单只取叶子——自定义档要完整可见集合)
- 新增 model RoleDeptTreeSelect{CheckedKeys,Depts}(model/system/sys_department.go)
- api/router: PUT /system/role/dataScope、GET /system/role/deptTree/:roleId

## 契约要点

- DataScope 传输:后端 int + json `,string` ↔ 前端 string,两边字符串往返
- 自定义档(5)部门集存 sys_role_departments;BuildIdentity 在 ScopeCustom 时 Pluck 为 CustomDeptIDs(datascope.go:63 早已就绪,**无需改引擎**)
- 超管角色 dataScope 固定「全部」,前端禁改

## 验证

go build + go vet 通过;前端 typecheck 通过。顺带修了 components.d.ts:182 过期生成残留(pwd-login.vue 钉钉图标早期冒号写法 `<icon-ant-design:dingtalk-circle-filled/>` 导致 unplugin-vue-components 生成非法 `const 'IconAntDesign:dingtalkCircleFilled'` 声明;源码早已改连字符写法,仅生成文件未刷新,手改对齐)。

## 关联

- 数据权限引擎全貌见 [[未实现功能分析]] 中 datascope 一节
- 角色管理基座见 business/role-management.md(八接口)
