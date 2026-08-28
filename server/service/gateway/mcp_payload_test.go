package gateway

import (
	"strings"
	"testing"

	"github.com/hllkk/devops-admin/server/model/gateway"
)

func floatPtr(v float64) *float64 { return &v }

func TestValidMCPServerName(t *testing.T) {
	cases := []struct {
		name string
		ok   bool
	}{
		{"github", true},
		{"github_mcp", true},
		{"Mcp9", true},
		{"", false},
		{"git-hub", false}, // '-' 破坏 /{server_name}/mcp 路由段
		{"git hub", false},
		{"服务器", false},
		{strings.Repeat("a", 129), false},
	}
	for _, c := range cases {
		if got := ValidMCPServerName(c.name); got != c.ok {
			t.Errorf("ValidMCPServerName(%q) = %v, want %v", c.name, got, c.ok)
		}
	}
}

func TestNormalizeMCPTransport(t *testing.T) {
	cases := []struct {
		in   string
		want string
		err  bool
	}{
		{"", gateway.MCPTransportStreamableHttp, false},
		{"sse", "sse", false},
		{"streamable_http", "streamable_http", false},
		{"streamableHttp", "streamable_http", false}, // AIHelms 风格归一
		{"http", "streamable_http", false},
		{"websocket", "", true},
	}
	for _, c := range cases {
		got, err := NormalizeMCPTransport(c.in)
		if (err != nil) != c.err {
			t.Errorf("NormalizeMCPTransport(%q) err = %v, want err=%v", c.in, err, c.err)
			continue
		}
		if got != c.want {
			t.Errorf("NormalizeMCPTransport(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	if litellmMCPTransport("sse") != "sse" || litellmMCPTransport("streamable_http") != "http" {
		t.Error("litellmMCPTransport 映射错误")
	}
}

func TestMCPCostInfo(t *testing.T) {
	// free → 无 cost_info
	if got := MCPCostInfo(gateway.MCPBillingFree, floatPtr(1), nil, "d", 7); got == nil || got["mcp_server_cost_info"] != nil {
		t.Errorf("free 不应有 cost_info: %v", got)
	}
	// 按次+默认价：¥/汇率
	got := MCPCostInfo(gateway.MCPBillingPerCall, floatPtr(0.7), nil, "", 7.0)
	costInfo := got["mcp_server_cost_info"].(map[string]any)
	if costInfo["default_cost_per_query"].(float64) != 0.1 {
		t.Errorf("默认价换算错误: %v", costInfo)
	}
	// 工具级覆盖 + 无默认价
	got = MCPCostInfo(gateway.MCPBillingPerCall, nil, map[string]*float64{"search": floatPtr(7.0)}, "", 7.0)
	costInfo = got["mcp_server_cost_info"].(map[string]any)
	if _, ok := costInfo["default_cost_per_query"]; ok {
		t.Error("无默认价不应有 default_cost_per_query")
	}
	perTool := costInfo["tool_name_to_cost_per_query"].(map[string]any)
	if perTool["search"].(float64) != 1.0 {
		t.Errorf("工具价换算错误: %v", perTool)
	}
	// 免费但带描述：仍有 mcp_info(仅描述)
	got = MCPCostInfo(gateway.MCPBillingFree, nil, nil, "desc", 7)
	if got["description"] != "desc" || got["mcp_server_cost_info"] != nil {
		t.Errorf("仅描述投影错误: %v", got)
	}
	// 全空 → nil(不下发 mcp_info 键)
	if got := MCPCostInfo(gateway.MCPBillingFree, nil, nil, "", 7); got != nil {
		t.Errorf("全空应返回 nil: %v", got)
	}
}

func TestBuildMCPTools(t *testing.T) {
	remote := []map[string]any{
		{"name": "search", "description": "搜索", "inputSchema": map[string]any{"type": "object"}},
		{"name": "fetch", "display_name": "抓取"},
		{"name": "", "description": "残缺项应跳过"},
	}
	billing := map[string]MCPCostConfig{"search": {BillingType: gateway.MCPBillingPerCall, ExternalCostPerCall: floatPtr(0.5)}}
	tools := BuildMCPTools(42, "github", remote, billing)
	if len(tools) != 2 {
		t.Fatalf("工具数 = %d, want 2", len(tools))
	}
	if tools[0].NamespacedName != "github_search" || tools[0].DisplayName != "search" {
		t.Errorf("工具 0 命名错误: %+v", tools[0])
	}
	if tools[0].BillingType != gateway.MCPBillingPerCall || *tools[0].ExternalCostPerCall != 0.5 {
		t.Errorf("计费配置未保留: %+v", tools[0])
	}
	if tools[1].DisplayName != "抓取" || tools[1].NamespacedName != "github_fetch" {
		t.Errorf("工具 1 错误: %+v", tools[1])
	}
	if tools[1].BillingType != "" || tools[1].ExternalCostPerCall != nil {
		t.Error("无配置工具应继承服务器(空)")
	}
	if string(tools[0].InputSchema) != `{"type":"object"}` {
		t.Errorf("Schema 提取错误: %s", tools[0].InputSchema)
	}
}

func TestCollectMCPToolBilling(t *testing.T) {
	tools := []gateway.MCPTool{
		{ToolName: "a", BillingType: gateway.MCPBillingPerCall, ExternalCostPerCall: floatPtr(1)},
		{ToolName: "b"}, // 无配置不收录
	}
	got := CollectMCPToolBilling(tools)
	if len(got) != 1 || got["a"].BillingType != gateway.MCPBillingPerCall {
		t.Errorf("CollectMCPToolBilling = %v", got)
	}
}
