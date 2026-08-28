-- =====================================================================
-- 已有库菜单补丁：模型广场(route.square) + 调用日志(route.usage)
-- 适用：在 sys_menu 种子(route.square 2026-08-27 / route.usage 2026-08-27)之前初始化的库。
-- 新库(初始化向导新建)不需要执行本补丁，种子已含。
-- 幂等：重复执行不产生重复行(NOT EXISTS / ON CONFLICT DO NOTHING)。
-- 生效方式：执行后重启 devops-admin 服务(菜单缓存 + casbin enforcer 启动时从表加载)。
-- 数据库：PostgreSQL
-- =====================================================================

-- 1. 插入菜单(menu_id 手造雪花风格值，与运行时生成的雪花 ID 区间不重叠；如冲突请改用界面「菜单管理」新增)
INSERT INTO sys_menu
  (menu_id, parent_id, menu_type, menu_name, order_num, path, component, module,
   is_frame, is_cache, visible, status, api_prefix, icon, create_time, update_time)
SELECT 2026082711000000001, 0, 'C', 'route.square', 10, 'square', '_gateway/square/index', 'gateway',
       '1', '0', '0', '0', '/gateway/model/active', 'lucide:store', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM sys_menu WHERE menu_name = 'route.square');

INSERT INTO sys_menu
  (menu_id, parent_id, menu_type, menu_name, order_num, path, component, module,
   is_frame, is_cache, visible, status, api_prefix, icon, create_time, update_time)
SELECT 2026082711000000002, 0, 'C', 'route.usage', 11, 'usage', '_gateway/usage/index', 'gateway',
       '1', '0', '0', '0', '/gateway/usage, /gateway/usage/*', 'lucide:scroll-text', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM sys_menu WHERE menu_name = 'route.usage');

-- 2. 角色授权：super/admin 两菜单全给；user 仅模型广场(调用日志是管理员视角)
INSERT INTO sys_role_menu (sys_role_id, sys_menu_id)
SELECT r.role_id, m.menu_id
FROM sys_role r
JOIN sys_menu m ON m.menu_name IN ('route.square', 'route.usage')
WHERE r.role_key IN ('super', 'admin')
  AND NOT EXISTS (
    SELECT 1 FROM sys_role_menu rm
    WHERE rm.sys_role_id = r.role_id AND rm.sys_menu_id = m.menu_id
  );

INSERT INTO sys_role_menu (sys_role_id, sys_menu_id)
SELECT r.role_id, m.menu_id
FROM sys_role r
JOIN sys_menu m ON m.menu_name = 'route.square'
WHERE r.role_key = 'user'
  AND NOT EXISTS (
    SELECT 1 FROM sys_role_menu rm
    WHERE rm.sys_role_id = r.role_id AND rm.sys_menu_id = m.menu_id
  );

-- 3. casbin 策略(与 saveRoleMenus→syncRoleCasbinPolicy 的产物一致: p, roleId, pattern, *)
--    调用日志接口前缀只给 super/admin；square 的 /gateway/model/active 在登录白名单，
--    策略照插保持与种子授权路径一致。
INSERT INTO casbin_rule (ptype, v0, v1, v2)
SELECT 'p', r.role_id::TEXT, p.pattern, '*'
FROM sys_role r
CROSS JOIN (VALUES
  ('/gateway/usage'),
  ('/gateway/usage/*'),
  ('/gateway/model/active')
) AS p(pattern)
WHERE r.role_key IN ('super', 'admin')
  AND NOT EXISTS (
    SELECT 1 FROM casbin_rule cr
    WHERE cr.ptype = 'p' AND cr.v0 = r.role_id::TEXT AND cr.v1 = p.pattern AND cr.v2 = '*'
  );
