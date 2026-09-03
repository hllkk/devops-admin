# AI 网关·纳管外部 stdio MCP server（SAP S/4HANA OData）

- 日期：2026-09-03
- 状态：**待实现**（方案已定稿，用户决定延后执行；本文件即落地配方，实现时照此操作）
- 关联：[[ai-gateway-mcp-stdio]]（stdio 型支持，本方案的技术底座）、[[ai-gateway-mcp-server]]（MCP 管理全链路）

## 需求

用户在网上发现第三方 MCP server `Nidhideep/sap-s4-mcp-server`（glama.ai 目录页），要求以最合理方式接入当前项目。结论：**平台零代码改动**，走 2026-09-03 刚落地的 MCP stdio 纳管，属于该能力的典型场景（Node 技术栈 stdio 子进程）。

## 该 server 事实（接入前必读）

- SAP S/4HANA OData 接入器，TypeScript，Node ≥18，**仅 stdio transport**；部署方式只有 git clone + `npm install` + `npm run build` → `node dist/src/index.js`（无 npm 包、无 HTTP 端点，`npx` 型注册不适用）
- 4 个工具：`discover_sap_services` / `get_entity_metadata` / `get_field_values` / `execute_odata_query`（**最后一个含写操作**）
- env 配置：`S4_ODATA_HOST`（完整 URL 含端口）、`S4_ODATA_CLIENT`（集团号）、`AUTH_METHOD`（basic/token）+ `S4_ODATA_USER`/`S4_ODATA_PASSWORD` 或 `S4_ODATA_TOKEN`；`DRY_RUN=true` 可在服务端阻断全部写操作
- 前置硬条件：可达的 SAP S/4HANA 系统（OData 服务开启）+ 有 OData 授权的 SAP 账号；没有则只能当广场演示条目

## 落地配方（dev）

1. **宿主机构建**（本机 node v24.18.0 已具备）：clone 走 gh-proxy 代理（HTTPS 直连 GitHub 不稳）→ build 出 `dist/`，产物落 `deploy/docker-dev/mcp/sap-s4-mcp-server/`
2. **compose 加挂载**：`deploy/docker-dev/docker-compose.yml` litellm 服务 volumes 追加 `./mcp/sap-s4-mcp-server:/opt/mcp/sap-s4:ro`（当前仅挂 config.yaml），`docker compose up -d litellm` 重建生效；**无需扩镜像**——dev litellm 容器内 node 已实测存在
3. **平台注册**（管理后台 MCP 管理，纯配置）：transport=stdio、command=`node`（白名单七项内）、args=`["/opt/mcp/sap-s4/dist/src/index.js"]`、url 留空、authType=none、env 凭据逐键录入（AES-256-GCM 落 Credentials 列）、**额外加 `DRY_RUN=true`**；serverName 禁 `-`（建议 `sap_s4`）
4. **闭环**：同步 LiteLLM → 拉取工具（应得 4 个）→ 健康检查 → 可见范围/计费（free 或 per_call）→ 发布广场；授权/接入配置/日志回流/计费全复用现有链路

## 生产环境差异（dev 配方之外的三点质变）

1. **dist 产物升级为部署资产**：上游仓库须 pin 到具体 commit、可重建、随 `deploy/docker-prod/` 目录体系纳入备份迁移；重新构建要留痕（谁/何时/哪版本）
2. **"容器内有 node"从事实变为隐式依赖契约**：litellm 官方镜像（prod 同 tag 1.99.0）不为托管 stdio 设计，node 属顺带产物、上游无承诺；**每次升 litellm 镜像必须重验 `docker exec devops-litellm which node`**，写进升级清单
3. **重启窗口与安全边界**：加挂载须重建 litellm 容器（全部 LLM 代理 + MCP 调用瞬断，挑窗口）；第三方代码以子进程进生产容器 + 持生产 SAP 凭据——**代码粗审（重点 config/auth.ts、config/policy.ts）+ DRY_RUN + SAP 只读账号三前置缺一不可**；prod 切换前先 exec 验证 node 同构

## 治理约束

- 第一期 `DRY_RUN=true` 只读上线，跑通 discover/metadata/field-values 链路后再评估放开写；SAP 侧账号同步给只读角色（双保险）
- stdio env 键后续不支持删除（浅 merge 限制），注册时一次配全键位
- 落地顺序：dev 全链路（构建→挂载→注册→拉工具→健康→一次真实调用）全通后，配方原样搬 prod，仅凭据换生产值
