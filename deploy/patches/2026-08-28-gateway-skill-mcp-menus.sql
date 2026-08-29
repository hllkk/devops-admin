-- =====================================================================
-- 已有库菜单补丁：MCP 服务器管理(route.mcp 2026-08-28) + Skill 管理(route.skill 2026-08-28)
-- 适用：在 sys_menu 种子含 route.mcp / route.skill 之前初始化的库。
-- 新库(初始化向导新建)不需要执行本补丁，种子已含。
-- 幂等：重复执行不产生重复行(NOT EXISTS)。
-- 生效方式：执行后重启 devops-admin 服务(菜单缓存 + casbin enforcer 启动时从表加载)。
-- 数据库：PostgreSQL
-- =====================================================================

-- 1. 插入菜单(menu_id 手造雪花风格值，与运行时生成的雪花 ID 区间不重叠；如冲突请改用界面「菜单管理」新增)
--    两页均为管理页：用户侧 active/download 走 casbin 登录白名单，不经菜单授权。
INSERT INTO sys_menu
  (menu_id, parent_id, menu_type, menu_name, order_num, path, component, module,
   is_frame, is_cache, visible, status, api_prefix, icon, create_time, update_time)
SELECT 2026082811000000001, 0, 'C', 'route.mcp', 13, 'mcp', '_gateway/mcp/index', 'gateway',
       '1', '0', '0', '0', '/gateway/mcp, /gateway/mcp/*', 'lucide:plug', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM sys_menu WHERE menu_name = 'route.mcp');

INSERT INTO sys_menu
  (menu_id, parent_id, menu_type, menu_name, order_num, path, component, module,
   is_frame, is_cache, visible, status, api_prefix, icon, create_time, update_time)
SELECT 2026082811000000002, 0, 'C', 'route.skill', 14, 'skill', '_gateway/skill/index', 'gateway',
       '1', '0', '0', '0', '/gateway/skill, /gateway/skill/*', 'lucide:package', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM sys_menu WHERE menu_name = 'route.skill');

-- 2. 角色授权：super/admin 给两管理页；user 不给(用户侧走登录白名单)
INSERT INTO sys_role_menu (sys_role_id, sys_menu_id)
SELECT r.role_id, m.menu_id
FROM sys_role r
JOIN sys_menu m ON m.menu_name IN ('route.mcp', 'route.skill')
WHERE r.role_key IN ('super', 'admin')
  AND NOT EXISTS (
    SELECT 1 FROM sys_role_menu rm
    WHERE rm.sys_role_id = r.role_id AND rm.sys_menu_id = m.menu_id
  );

-- 3. casbin 策略(与 saveRoleMenus→syncRoleCasbinPolicy 的产物一致: p, roleId, pattern, *)
INSERT INTO casbin_rule (ptype, v0, v1, v2)
SELECT 'p', r.role_id::TEXT, p.pattern, '*'
FROM sys_role r
CROSS JOIN (VALUES
  ('/gateway/mcp'),
  ('/gateway/mcp/*'),
  ('/gateway/skill'),
  ('/gateway/skill/*')
) AS p(pattern)
WHERE r.role_key IN ('super', 'admin')
  AND NOT EXISTS (
    SELECT 1 FROM casbin_rule cr
    WHERE cr.ptype = 'p' AND cr.v0 = r.role_id::TEXT AND cr.v1 = p.pattern AND cr.v2 = '*'
  );
