# 服务器模块规划与可行性分析

> 日期：2026-10-07 ｜ 状态：slice1（资产+凭据+概览）与 slice2（agent 安装流+注册心跳）已落地（2026-10-08，含本机 Docker 内置资产/账号授权 P4 规划/SSH 凭据去密码三项追加决策）；安装流真实目标机验证待用户执行；后续 slice3（指标管道）/slice4（拨测告警）

## 需求

用户规划在 devops-admin（AIOps）新建服务器模块，规划五大功能：

1. 服务器资产管理
2. 物理服务器 IPMI 管理
3. Docker 纳管监控
4. 数据库监控
5. 网络监控

先做可行性分析：这些功能在当前项目技术栈下能否成功实现。

## 可行性结论（2026-10-07 分析）

**五项全部可实现，无技术硬伤**。风险集中在环境前提（管理网可达性）与运维接入成本，而非代码实现：

- 服务器资产管理：标准 GVA 四层 CRUD + ICMP/TCP 拨测在线状态，零新增依赖风险，建议首期打底
- IPMI：agentless，`stmcgann/gofish`（Redfish，现代 BMC）+ `vmware/goipmi`（IPMI 2.0 over LAN 老设备兜底）双协议；硬前提=BMC 管理网可达；国产 BMC Redfish 兼容度需实测
- Docker：`fsouza/go-dockerclient`/moby client；关键决策=接入方式（TCP+TLS / SSH 隧道 / 先纳管本机 sock）；多数生产 daemon 只听 unix socket，接入是运维成本非代码问题
- 数据库监控：MySQL/PG/Redis 驱动（go-sql-driver/mysql、pgx、go-redis）**已在 go.mod**，纯查询零侵入，风险最低
- 网络监控：拨测（ping/TCP/HTTP）+ SNMP（`gosnmp/gosnmp`）可行；netflow/sFlow 流量分析明确不建议第一期

## 统一架构方向（复用现有模式，不引入 Prometheus）

- 全 agentless 采集（IPMI/SSH/Docker API/DB 直连/SNMP/ICMP），不在目标机装 agent
- 资产为锚：统一资产表 + 类型字段，BMC/Docker主机/DB实例/网络设备为资产的采集通道
- 调度复用 task.Register + SysTimedTask 面板（同 AI 网关种子任务模式）
- 指标存储：PG 明细表 + 聚合表滚动重建 + 保留期清理（照搬 gateway_llm_log → cost_summary_daily → cleanup 模式），不引入时序库
- 告警：阈值规则表 + CheckAlerts 定时 + SysNotice/企微推送 + 去重（照搬 budget_alert 模式）
- 凭据：BMC/SSH/DB/SNMP 凭据复用 gateway credential 的 AES-256-GCM 加密 + 掩码回显模式
- 前端落 `_server/` 目录（占位页与 module=server 菜单 seed 已预留）

## 决策点（2026-10-07 用户确认）

1. **OS 层指标用 agent**：录入资产时可输入 root 账号密码——仅内存态一次性使用，用完即在服务器 authorized_keys 配置平台公钥（转密钥认证）并自动安装 agent，**密码不落库**；前端资产行提供「安装 agent」入口，点击后自动安装（scp 二进制 + systemd 注册）
2. **Docker 先只纳管本机**（挂 docker.sock 只读），远程纳管后续再做
3. **规模约 150 台**（服务器+虚机+网络设备），**默认采集频率 5 秒可配置**——影响存储设计：5s×150 台高频数据需热数据缓冲（Redis）+ PG 降采样聚合分层，不能纯 PG 明细表硬扛
4. **IPMI 厂商=浪潮/华为/联想**：三家 BMC 均支持 Redfish（联想 XCC 最标准，华为 iBMC 有私有扩展）+ IPMI 2.0 over LAN 兜底，gofish+goipmi 双协议覆盖成立，实施时逐家实测校准
5. **网络监控=拨测+SNMP 够用**，流量分析明确不做

## slice1 已有库补菜单 SQL（dev/prod 已初始化库执行后重启，启动期 RebuildRoleCasbinPolicies 自愈 casbin；全新库走 seed 无需执行）

> 注：dev 库已由 AI 直接执行。曾踩两坑：①表名是 `sys_roles`（复数）；②docker exec 传 SQL heredoc 必须带 `-i` 否则 stdin 不进容器。

### 结构修正 SQL（概览页改造后最终形态；已执行过旧版「M 目录+子菜单」补丁的库跑这段）

```sql
-- 1. route.server: M 目录 → C 概览页(模块顶层第一项,对齐 admin 首页/AI 看板)
UPDATE sys_menu SET menu_type = 'C', component = '_server/server/index',
       api_prefix = '/server/asset/overview', remark = ''
WHERE menu_name = 'route.server' AND menu_type = 'M';

-- 2. route.server_asset → route.asset 顶层单页(对齐 ai-key 模式)
UPDATE sys_menu SET menu_name = 'route.asset', parent_id = 0, path = 'asset',
       component = '_server/asset/index', order_num = 10
WHERE menu_name = 'route.server_asset';
```

### 旧版初装 SQL（slice1 首发时的「M 目录+子菜单」形态，已被上面结构修正取代，未打过的库可直接跑「初装+修正」合并或重建库）

```sql
-- 1. route.server 从顶层 C 占位改为 M 目录
UPDATE sys_menu SET menu_type = 'M', component = 'Layout', remark = '服务器管理目录'
WHERE menu_name = 'route.server' AND menu_type = 'C';

-- 2. 插入资产管理子菜单(幂等)
INSERT INTO sys_menu (parent_id, menu_name, menu_type, order_num, path, component, module, icon, api_prefix, is_frame, is_cache, visible)
SELECT m.menu_id, 'route.server_asset', 'C', 1, 'server/asset', '_server/server/asset/index', 'server', 'mdi:server',
       '/server/asset, /server/asset/*, /server/credential, /server/credential/*', '1', '0', '0'
FROM sys_menu m
WHERE m.menu_name = 'route.server'
  AND NOT EXISTS (SELECT 1 FROM sys_menu c WHERE c.menu_name = 'route.server_asset');

-- 3. super/admin 授权子菜单(user 角色不授,管理页面)
INSERT INTO sys_role_menu (sys_role_id, sys_menu_id)
SELECT r.role_id, m.menu_id
FROM sys_roles r, sys_menu m
WHERE r.role_key IN ('super', 'admin') AND m.menu_name = 'route.server_asset'
  AND NOT EXISTS (SELECT 1 FROM sys_role_menu rm WHERE rm.sys_role_id = r.role_id AND rm.sys_menu_id = m.menu_id);
```

## 相关

- [[module-isolation-backend-driven]]（server 模块菜单/路由隔离机制已就绪）
- [[ai-gateway-overview]]（AI 网关为大型业务模块落地的成熟参照）
