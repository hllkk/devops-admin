# 企微可信域名校验·公网 404 排查与修复

- 日期：2026-09-04
- 状态：已修复（build/vet/test 全过；**含路由 bug 修复，须重新构建部署生产才生效**）
- 反向链接：[[wecom-edge-callback-deploy]]（外层 DMZ 白名单形态）

## 需求

用户反馈：系统设置里可信域名校验已配置（生产库 `sys_auth_config` 文件名/内容均非空），公网访问 `/WW_verify_*.txt` 仍返回 not found。要求分析原因并优化。

## 根因（探针实证，gin v1.12.0）

**路由 bug**：`PublicRouter.GET("/WW_verify_:name.txt", ...)` 这种"段内参数+静态后缀"写法不成立——gin 的 param 名会吞掉段内 `.txt`（param key 变成 `name.txt`），`c.Param("name")` 恒返回空串，handler 拼出 `WW_verify_.txt` 永不等于配置 → **恒 404，功能从上线起就没工作过**。配置与外层/内层 Nginx 链路均无问题（"not found"裸串即 handler 返回，链路已通）。

探针结论（已验证后删除）：`/WW_verify_:name.txt` → 200 但 `c.Param("name")=""`、`c.Params=[{name.txt YEuHCfLumKsT6F1F.txt}]`；`/WW_verify_:name` → `c.Param("name")="YEuHCfLumKsT6F1F.txt"` ✓。

## 修复

- `router/system/sys_setting.go`：路由改 `/WW_verify_:name`（param 吃整段含 .txt），注释固化坑因防回退。
- `api/v1/system/sys_setting.go` WecomDomainVerify：`filename := "WW_verify_" + c.Param("name")`（不再补 .txt）；比对前规范化配置值（trim/前缀后缀补全，兼容存量脏数据）；两个 404 分支落 Warn 日志（记请求 filename 与 configured，区分"未配置"vs"文件名轮换"）。**刻意 Warn 不 Error**：公开端点防公网扫描灌爆 sys_error。
- `service/system/sys_auth_config.go` Set：落库前规范化（`NormalizeWecomDomainFileName` trim+中段自动补 `WW_verify_`/`.txt`；内容 trim）。
- `model/system/sys_auth_config.go`：`NormalizeWecomDomainFileName` 纯函数 + `sys_auth_config_test.go` 7 场景单测。

## 部署与自测

- 修复在后端二进制，生产须重建（build-release.sh 在线升级 或 docker compose build server && up -d），仅改配置/重启无效。
- 部署后自测：`curl http://172.21.96.171/proxy-default/WW_verify_<中段>.txt` 应返回配置内容；公网同路径同验；然后企微管理后台重新点「申请校验」。

## 排查口径（运维用）

- 生效配置查生产库：`select wecom_domain_file_name, length(wecom_domain_file_content) from sys_auth_config where id=1;`（表名单数 `sys_auth_config`）。
- 内网直连 `/proxy-default/WW_verify_*.txt` 与公网对照，一致即后端问题、非链路问题。
