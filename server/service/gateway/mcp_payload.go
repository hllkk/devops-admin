package gateway

import (
	"fmt"
	"math"
	"regexp"

	"gorm.io/datatypes"

	"github.com/hllkk/devops-admin/server/model/gateway"
	"github.com/hllkk/devops-admin/server/utils/litellm"
)

// 本文件是 MCP 服务器的投影与校验纯函数层（零 DB/配置依赖，可单测）。
// 总原则与 credential_payload 一致：平台 DB 存原始值（¥ 口径定价/密文凭据），派生值只在
// 发往 LiteLLM 前的投影构建中产生，绝不回写平台 DB。规避 AIHelms 三坑：凭证明文落库回显
// （掩码+密文）、allow_all_keys 放开（投影恒 false）、授权只加不收（对齐在 service 层）。

// mcpServerNameRE LiteLLM 路由名合法字符：字母/数字/下划线。'-' 等字符会破坏
// /{server_name}/mcp 路由段与 key allowed_mcp_servers 匹配（AIHelms 同款限制）。
var mcpServerNameRE = regexp.MustCompile(`^[a-zA-Z0-9_]{1,128}$`)

// ValidMCPServerName 校验 LiteLLM 路由名（唯一性由服务层查重）。
func ValidMCPServerName(name string) bool {
	return mcpServerNameRE.MatchString(name)
}

// NormalizeMCPTransport 传输协议归一：空→默认 streamable_http；兼容 AIHelms 风格的
// streamableHttp 写法。返回值在 (sse|streamable_http|stdio) 三选一，非法报错。
func NormalizeMCPTransport(t string) (string, error) {
	switch t {
	case "":
		return gateway.MCPTransportStreamableHttp, nil
	case gateway.MCPTransportSse, gateway.MCPTransportStreamableHttp, gateway.MCPTransportStdio:
		return t, nil
	case "streamableHttp", "http":
		// AIHelms/旧版写法与 LiteLLM 侧传输名归一到平台枚举
		return gateway.MCPTransportStreamableHttp, nil
	}
	return "", fmt.Errorf("传输协议仅支持 sse、streamable_http 或 stdio")
}

// litellmMCPTransport 平台协议 → LiteLLM 传输名（sse/http/stdio，stdio 同名透传）。
func litellmMCPTransport(t string) string {
	switch t {
	case gateway.MCPTransportSse, gateway.MCPTransportStdio:
		return t
	}
	return "http"
}

// mcpStdioCommands LiteLLM stdio 命令白名单(与上游校验同清单)：仅放行标准运行时，
// 任意二进制拒绝——子进程跑在 LiteLLM 容器内，这是上游的安全边界，平台侧前置拦截，
// 错误提示不依赖 LiteLLM 报错。
var mcpStdioCommands = map[string]bool{
	"deno": true, "docker": true, "node": true, "npx": true,
	"python": true, "python3": true, "uvx": true,
}

// ValidMCPStdioCommand stdio 启动命令白名单校验(命令名精确比对、不带路径——
// LiteLLM 在容器内按 PATH 解析，带路径的写法上游同样拒绝)。
func ValidMCPStdioCommand(command string) bool {
	return mcpStdioCommands[command]
}

// ValidMCPAuthType 校验鉴权方式（LiteLLM MCPAuth 常用子集，none 无凭据）。
func ValidMCPAuthType(t string) bool {
	switch t {
	case gateway.MCPAuthNone, gateway.MCPAuthApiKey, gateway.MCPAuthBearerToken:
		return true
	}
	return false
}

// BuildMCPEndpointSpec 组装 LiteLLM MCP 探测端点描述(test/connection、test/tools/list
// 共用)：http/sse 型带 url+鉴权；stdio 型带 command/args/env——credentials 列在 stdio 时
// 存 env 键值对(env 即凭据)。纯函数，不落库。
func BuildMCPEndpointSpec(row *gateway.MCPServer, credentials map[string]any) litellm.MCPEndpointSpec {
	spec := litellm.MCPEndpointSpec{Transport: litellmMCPTransport(row.Transport)}
	if row.Transport == gateway.MCPTransportStdio {
		spec.Command = row.Command
		spec.Args = jsonToSlice(row.Args)
		spec.Env = credentials
		return spec
	}
	spec.Url = row.Url
	spec.AuthType = row.AuthType
	spec.Credentials = credentials
	return spec
}

// mcpCostToUsd ¥→USD 换算（LiteLLM mcp_server_cost_info 记 USD；rate<=0 兜底 7.0，
// 与 ConvertCostsForLitellm/budgetLimitToUsd 同口径；保留 6 位小数规避浮点尾巴）。
func mcpCostToUsd(cny, usdToCnyRate float64) float64 {
	if usdToCnyRate <= 0 {
		usdToCnyRate = 7.0
	}
	return math.Round(cny/usdToCnyRate*1e6) / 1e6
}

// MCPCostInfo 组装 LiteLLM mcp_info 投影（description 展示提示 + mcp_server_cost_info）。
// billingType=free 或无定价 → cost_info 为 nil（LiteLLM 不计费）；工具级定价覆盖默认价。
// 换算 ¥→USD；返回 nil 表示整个 mcp_info 无内容（create/update 时不下发该键）。
func MCPCostInfo(billingType string, serverCost *float64, toolCosts map[string]*float64, description string, usdToCnyRate float64) map[string]any {
	info := map[string]any{}
	if description != "" {
		info["description"] = description
	}
	if billingType != gateway.MCPBillingPerCall || (serverCost == nil && len(toolCosts) == 0) {
		if len(info) == 0 {
			return nil
		}
		return info
	}
	costInfo := map[string]any{}
	if serverCost != nil {
		costInfo["default_cost_per_query"] = mcpCostToUsd(*serverCost, usdToCnyRate)
	}
	if len(toolCosts) > 0 {
		perTool := map[string]any{}
		for name, cost := range toolCosts {
			if cost != nil {
				perTool[name] = mcpCostToUsd(*cost, usdToCnyRate)
			}
		}
		if len(perTool) > 0 {
			costInfo["tool_name_to_cost_per_query"] = perTool
		}
	}
	if len(costInfo) > 0 {
		info["mcp_server_cost_info"] = costInfo
	}
	if len(info) == 0 {
		return nil
	}
	return info
}

// MCPCostConfig 工具级计费配置（refresh-tools 重建工具表时按 tool_name 保留）。
type MCPCostConfig struct {
	BillingType         string
	ExternalCostPerCall *float64
}

// CollectMCPToolBilling 从既有工具行收集计费配置（key=tool_name，仅收录有配置的行）。
func CollectMCPToolBilling(tools []gateway.MCPTool) map[string]MCPCostConfig {
	out := map[string]MCPCostConfig{}
	for i := range tools {
		t := tools[i]
		if t.BillingType == "" && t.ExternalCostPerCall == nil {
			continue
		}
		out[t.ToolName] = MCPCostConfig{BillingType: t.BillingType, ExternalCostPerCall: t.ExternalCostPerCall}
	}
	return out
}

// BuildMCPTools 从远端工具列表构建本地工具行（全量重建语义，不含主键——由 GORM 回调填充）：
// namespaced = {serverName}_{tool_name}；按 tool_name 套用 billing 保留的计费配置。
// name 为空的远端项跳过（残缺数据不入库）。
func BuildMCPTools(serverId int64, serverName string, remote []map[string]any, billing map[string]MCPCostConfig) []gateway.MCPTool {
	out := make([]gateway.MCPTool, 0, len(remote))
	for _, item := range remote {
		name, _ := item["name"].(string)
		if name == "" {
			continue
		}
		display, _ := item["display_name"].(string)
		if display == "" {
			display = name
		}
		desc, _ := item["description"].(string)
		tool := gateway.MCPTool{
			McpServerId:    serverId,
			ToolName:       name,
			NamespacedName: serverName + "_" + name,
			DisplayName:    display,
			Description:    desc,
			InputSchema:    mcpToolSchema(item),
		}
		if cfg, ok := billing[name]; ok {
			tool.BillingType = cfg.BillingType
			tool.ExternalCostPerCall = cfg.ExternalCostPerCall
		}
		out = append(out, tool)
	}
	return out
}

// mcpToolSchema 提取远端工具入参 Schema（兼容 inputSchema/input_schema 两种键）。
func mcpToolSchema(item map[string]any) datatypes.JSON {
	for _, key := range []string{"inputSchema", "input_schema"} {
		if v, ok := item[key]; ok {
			if m, ok := v.(map[string]any); ok {
				return marshalJSON(m)
			}
		}
	}
	return datatypes.JSON("{}")
}

// MergeMCPCredentials 合并编辑传入的凭据（掩码回传=保留旧明文），复用凭证同款语义。
func MergeMCPCredentials(oldValues, incoming map[string]any) map[string]any {
	return MergeCredentialValues(oldValues, incoming)
}
