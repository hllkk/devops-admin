# 角色默认路由 DefaultRouter + home tab 修复

> 日期：2026-07-24 ｜ 状态：已实现

## 需求
1. **home tab 刷新 bug**：刷新后 tab 第一恒为 admin。根因：`initHomeTab` 用全局 `routeStore.routeHome`(恒 admin),没跟当前模块。
2. **角色默认路由**：不同角色登录默认进不同模块/路由(对齐 GVA `SysAuthority.DefaultRouter`)。

## 改动

### home tab（`store/modules/tab/index.ts`）
- `initHomeTab` 用 `MODULE_CONFIG[currentModule].home`（跟随当前模块）+ `watch(currentModule)` 同步
- **原因**:切换走 `resetTabs`(目标模块),刷新走 `initHomeTab`。若 home tab=DefaultRouter,刷新会回退到角色默认页(与切换不一致)。改跟随 currentModule 后,登录/切换/刷新 home tab 都=当前模块首页,一致
- DefaultRouter 只管登录入口(`redirectFromLogin`),home tab 不绑定它
- (曾用"home tab=routeHome=DefaultRouter",导致刷新回 admin,已弃用)

### DefaultRouter（角色默认路由）
**后端**
- `SysRole` 加 `DefaultRouter string`（gorm `default:admin`）；seed 角色、`RoleOperateParams`、`CreateRole`/`UpdateRole` 全链路
- `GetUserDetail` 返回**主角色**（`user.RoleId`）的 defaultRouter；`UserInfoResponse` 加字段下发
- `resolveHome` 改：优先 defaultRouter（`containsRouteName` 校验用户有权访问），兜底 admin；`TestResolveHome` 覆盖新逻辑
- `MenuTreeSelectNode` 加 `Path`（供前端小房子推导 routeKey）；`buildMenuTreeSelect` 填充

**前端**
- `Api.Auth.UserInfo`/`Api.System.Role`/`RoleOperateParams` 加 `defaultRouter`
- `redirectFromLogin` 加 `homeRouteName` 参数；`login` 用 `userInfo.defaultRouter`（redirect query 优先）
- `MenuTree`：`enableDefaultRouter` prop + `defaultRouter` defineModel；C 菜单 `renderLabel` 加小房子（`path→routeKey`），点击设 defaultRouter（mdi:home 高亮当前）
- `role-operate-drawer`：model/submit 带 defaultRouter，MenuTree 绑 `v-model:default-router` + `:enable-default-router`
- i18n `setDefaultRouter` zh/en/app.d.ts 三处同步

## 关键约束
- **DefaultRouter 值 = routeKey**（路由名，如 `admin`/`disk`/`system_user`），非 path、非模块名；前端 `routerPushByKey` 跳转，后端 `resolveHome` 匹配 `MenuRoute.Name`
- **多角色取主角色**（`user.RoleId`）的 DefaultRouter
- **老库 `sys_roles.default_router` 空**（现有角色）→ getUserInfo 返回空 → resolveHome 兜底 admin（行为同改动前）；配默认模块需在角色管理编辑、菜单授权树点小房子
- **home tab = 当前模块首页**(currentModule):跟随 URL 所在模块,登录/切换/刷新一致。DefaultRouter 只管登录跳转入口,不绑定 home tab

## 相关
- [[module-isolation-backend-driven]] 模块隔离（module 字段决定 currentModule，home tab 跟随；DefaultRouter 决定登录落点模块）
