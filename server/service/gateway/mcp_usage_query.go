package gateway

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/gateway"
	gatewayReq "github.com/hllkk/devops-admin/server/model/gateway/request"
	gatewayResp "github.com/hllkk/devops-admin/server/model/gateway/response"
	"github.com/hllkk/devops-admin/server/model/system"
)

// GetMcpLogList MCP 调用日志分页明细(管理员视角,挂 /gateway/usage/mcp/list,
// 与 LLM 调用日志同菜单 casbin 零改动)。
func (s *UsageSyncService) GetMcpLogList(ctx context.Context, q *gatewayReq.McpLogSearch) ([]gatewayResp.McpLogView, int64, error) {
	db := global.OPS_DB.WithContext(ctx).Model(&gateway.McpLog{})
	if q.UserId != 0 {
		db = db.Where("user_id = ?", q.UserId)
	}
	if q.AiKeyId != 0 {
		db = db.Where("ai_key_id = ?", q.AiKeyId)
	}
	if q.McpServerId != 0 {
		db = db.Where("mcp_server_id = ?", q.McpServerId)
	}
	if q.ToolName != "" {
		db = db.Where("tool_name LIKE ?", "%"+q.ToolName+"%")
	}
	if q.Status != "" {
		db = db.Where("status = ?", q.Status)
	}
	if t, err := time.Parse(time.RFC3339, q.StartTime); err == nil {
		db = db.Where("started_at >= ?", t.UTC())
	}
	if t, err := time.Parse(time.RFC3339, q.EndTime); err == nil {
		db = db.Where("started_at <= ?", t.UTC())
	}

	var total int64
	var rows []gateway.McpLog
	limit, offset := q.LimitOffset()
	if limit > 0 {
		err := db.Count(&total).Order("started_at DESC").Limit(limit).Offset(offset).Find(&rows).Error
		if err != nil {
			return nil, 0, err
		}
	} else {
		err := db.Count(&total).Order("started_at DESC").Find(&rows).Error
		if err != nil {
			return nil, 0, err
		}
	}
	return fillMcpLogNames(ctx, rows), total, nil
}

// fillMcpLogNames 批量回填归因实体可读名(每页两次 IN 查询,避免逐行 N+1)。
func fillMcpLogNames(ctx context.Context, rows []gateway.McpLog) []gatewayResp.McpLogView {
	userIds, keyIds := make([]int64, 0, len(rows)), make([]int64, 0, len(rows))
	for i := range rows {
		if rows[i].UserId != 0 {
			userIds = append(userIds, rows[i].UserId)
		}
		if rows[i].AiKeyId != 0 {
			keyIds = append(keyIds, rows[i].AiKeyId)
		}
	}
	userNames, keyNames := map[int64]string{}, map[int64]string{}
	db := global.OPS_DB.WithContext(ctx)
	if len(userIds) > 0 {
		var users []system.SysUser
		if err := db.Select("id, nick_name").Where("id IN ?", userIds).Find(&users).Error; err == nil {
			for i := range users {
				userNames[users[i].UserId] = users[i].NickName
			}
		}
	}
	if len(keyIds) > 0 {
		var keys []gateway.AiKey
		if err := db.Select("ai_key_id, name").Where("ai_key_id IN ?", keyIds).Find(&keys).Error; err == nil {
			for i := range keys {
				keyNames[keys[i].AiKeyId] = keys[i].Name
			}
		}
	}
	list := make([]gatewayResp.McpLogView, 0, len(rows))
	for i := range rows {
		list = append(list, gatewayResp.McpLogView{
			McpLog:    rows[i],
			UserName:  userNames[rows[i].UserId],
			AiKeyName: keyNames[rows[i].AiKeyId],
		})
	}
	return list
}

// ───────────────── 成本分析·MCP 维(扫 gateway_mcp_log 实时聚合,MCP 调用量远小于 LLM 暂不进聚合表) ─────────────────

// costMcpDeptJoins/anchor MCP 日志表(m)的部门读时归因(与聚合表同口径,规避 LLM JOIN 常量绑定别名 s)。
const costMcpDeptJoins = `LEFT JOIN sys_users u ON u.id = m.user_id AND m.user_id <> 0
	LEFT JOIN gateway_ai_key k ON k.ai_key_id = m.ai_key_id AND m.ai_key_id <> 0`

const costMcpDeptAnchor = `CASE WHEN k.owner_type = 'dept' THEN k.owner_id ELSE COALESCE(u.dept_id, 0) END`

// costBizDayRange 业务日闭区间(Asia/Shanghai) → UTC 时间下界/上界(timestamptz 过滤走 started_at 索引)。
func costBizDayRange(start, end string) (time.Time, time.Time) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.UTC
	}
	st, _ := time.ParseInLocation("2006-01-02", start, loc)
	et, _ := time.ParseInLocation("2006-01-02", end, loc)
	return st.UTC(), et.AddDate(0, 0, 1).UTC() // 上界=次日零点(开区间)
}

// mcpCostBaseQuery MCP 日志 + 部门归因 JOIN + 公共筛选(userId/aiKeyId/部门锚点/业务日)。
func mcpCostBaseQuery(ctx context.Context, f *gatewayReq.CostAnalysisSearch, start, end string, deptIds []int64) *gorm.DB {
	db := global.OPS_DB.WithContext(ctx).Table("gateway_mcp_log m").Joins(costMcpDeptJoins)
	lo, hi := costBizDayRange(start, end)
	db = db.Where("m.started_at >= ? AND m.started_at < ?", lo, hi)
	if len(deptIds) > 0 {
		db = db.Where(costMcpDeptAnchor+" IN ?", deptIds)
	}
	if f.UserId != 0 {
		db = db.Where("m.user_id = ?", f.UserId)
	}
	if f.AiKeyId != 0 {
		db = db.Where("m.ai_key_id = ?", f.AiKeyId)
	}
	return db
}

// sumMcpCost MCP 期间成本/调用数合计(成本分析 overview 合并口径用)。
func sumMcpCost(ctx context.Context, f *gatewayReq.CostAnalysisSearch, start, end string) (internal, external float64, calls int, err error) {
	type agg struct {
		Internal float64
		External float64
		Calls    int
	}
	var a agg
	if err = mcpCostBaseQuery(ctx, f, start, end, expandDeptSubtree(ctx, f.DepartmentId)).
		Select(`COALESCE(SUM(m.internal_cost),0) AS internal,
			COALESCE(SUM(m.external_cost),0) AS external,
			COUNT(*) AS calls`).Scan(&a).Error; err != nil {
		return
	}
	return a.Internal, a.External, a.Calls, nil
}

// mcpTrend MCP 按业务日聚合(Asia/Shanghai 切桶,与聚合表同口径)。
func mcpTrend(ctx context.Context, f *gatewayReq.CostAnalysisSearch, start, end string) (map[string]gatewayResp.CostTrendItem, error) {
	var rows []gatewayResp.CostTrendItem
	if err := mcpCostBaseQuery(ctx, f, start, end, expandDeptSubtree(ctx, f.DepartmentId)).
		Select(`to_char((m.started_at AT TIME ZONE 'Asia/Shanghai')::date,'YYYY-MM-DD') AS date,
			COALESCE(SUM(m.internal_cost),0) AS internal_cost,
			COALESCE(SUM(m.external_cost),0) AS external_cost,
			COUNT(*) AS requests`).
		Group(`(m.started_at AT TIME ZONE 'Asia/Shanghai')::date`).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]gatewayResp.CostTrendItem, len(rows))
	for _, r := range rows {
		out[r.Date] = r
	}
	return out, nil
}
