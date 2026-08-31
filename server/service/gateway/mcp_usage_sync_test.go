package gateway

import (
	"testing"

	"github.com/hllkk/devops-admin/server/model/gateway"
)

func f64(v float64) *float64 { return &v }

// calcMcpCosts：工具级优先(含未声明 billing_type 但配了单价的覆盖)→服务器级→免费 0；
// internal 单价 nil 回落 external
func TestCalcMcpCosts(t *testing.T) {
	// nil ref：未归因 0
	if e, i := calcMcpCosts(nil); e != 0 || i != 0 {
		t.Fatalf("nil ref 应为 0, got %v/%v", e, i)
	}
	// 工具级 per_call 双价
	toolRef := &mcpToolRef{
		tool:    &gateway.MCPTool{BillingType: gateway.MCPBillingPerCall, ExternalCostPerCall: f64(0.5), InternalCostPerCall: f64(0.3)},
		server:  &gateway.MCPServer{BillingType: gateway.MCPBillingPerCall, ExternalCostPerCall: f64(9)},
	}
	if e, i := calcMcpCosts(toolRef); e != 0.5 || i != 0.3 {
		t.Fatalf("工具级双价失败: %v/%v", e, i)
	}
	// 工具级 internal nil → 回落 external
	toolRef = &mcpToolRef{
		tool:   &gateway.MCPTool{BillingType: gateway.MCPBillingPerCall, ExternalCostPerCall: f64(0.5)},
		server: &gateway.MCPServer{BillingType: gateway.MCPBillingPerCall, InternalCostPerCall: f64(9)},
	}
	if e, i := calcMcpCosts(toolRef); e != 0.5 || i != 0.5 {
		t.Fatalf("工具级 internal 回落失败: %v/%v", e, i)
	}
	// 工具级 free 覆盖服务器计费
	toolRef = &mcpToolRef{
		tool:   &gateway.MCPTool{BillingType: gateway.MCPBillingFree},
		server: &gateway.MCPServer{BillingType: gateway.MCPBillingPerCall, ExternalCostPerCall: f64(9)},
	}
	if e, i := calcMcpCosts(toolRef); e != 0 || i != 0 {
		t.Fatalf("工具级 free 应为 0, got %v/%v", e, i)
	}
	// 工具未声明 billing_type 但配了单价 → 视为 per_call 覆盖
	toolRef = &mcpToolRef{
		tool:   &gateway.MCPTool{ExternalCostPerCall: f64(0.2)},
		server: &gateway.MCPServer{BillingType: gateway.MCPBillingPerCall, ExternalCostPerCall: f64(9)},
	}
	if e, i := calcMcpCosts(toolRef); e != 0.2 || i != 0.2 {
		t.Fatalf("工具隐式 per_call 覆盖失败: %v/%v", e, i)
	}
	// 工具完全继承(BillingType 空+无单价) → 服务器级
	toolRef = &mcpToolRef{
		tool:   &gateway.MCPTool{},
		server: &gateway.MCPServer{BillingType: gateway.MCPBillingPerCall, ExternalCostPerCall: f64(1.5), InternalCostPerCall: f64(1.2)},
	}
	if e, i := calcMcpCosts(toolRef); e != 1.5 || i != 1.2 {
		t.Fatalf("服务器级计费失败: %v/%v", e, i)
	}
	// 服务器级 internal nil 回落 external
	toolRef = &mcpToolRef{
		server: &gateway.MCPServer{BillingType: gateway.MCPBillingPerCall, ExternalCostPerCall: f64(1.5)},
	}
	if e, i := calcMcpCosts(toolRef); e != 1.5 || i != 1.5 {
		t.Fatalf("服务器级 internal 回落失败: %v/%v", e, i)
	}
	// 服务器 free / 无 ref.server
	toolRef = &mcpToolRef{server: &gateway.MCPServer{BillingType: gateway.MCPBillingFree, ExternalCostPerCall: f64(1)}}
	if e, i := calcMcpCosts(toolRef); e != 0 || i != 0 {
		t.Fatalf("服务器 free 应为 0, got %v/%v", e, i)
	}
	toolRef = &mcpToolRef{}
	if e, i := calcMcpCosts(toolRef); e != 0 || i != 0 {
		t.Fatalf("无 server 应为 0, got %v/%v", e, i)
	}
}
