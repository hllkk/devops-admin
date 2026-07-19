package system

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
)

// 验证登录日志 query 绑定(顶层字段 + 分页)。
func TestLoginLogSearchBinding(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = &http.Request{
		URL:    mustParseQuery("pageNum=2&pageSize=20&userName=admin&ipaddr=10.0.0.1&status=0"),
		Header: make(http.Header),
	}
	var q systemReq.LoginLogSearch
	if err := c.ShouldBindQuery(&q); err != nil {
		t.Fatalf("bind login log search: %v", err)
	}
	if q.UserName != "admin" || q.Ipaddr != "10.0.0.1" || q.Status != "0" {
		t.Fatalf("basic field mismatch: %+v", q)
	}
	if q.PageNum != 2 || q.PageSize != 20 {
		t.Fatalf("page mismatch: %+v", q)
	}
}

// 验证操作日志:
//  1. 顶层字段(title/businessType)走 struct binding;
//  2. 时间范围走 c.Query("params[beginTime]")(前端 qs.stringify bracket 序列化,
//     gin struct binding 不支持 bracket 嵌套, api 层显式取值)。
func TestOperLogSearchParamsQuery(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = &http.Request{
		URL:    mustParseQuery("pageNum=1&pageSize=10&title=%E7%94%A8%E6%88%B7&businessType=2&params[beginTime]=2026-07-01%2000:00:00&params[endTime]=2026-07-19%2023:59:59"),
		Header: make(http.Header),
	}
	var q systemReq.OperLogSearch
	if err := c.ShouldBindQuery(&q); err != nil {
		t.Fatalf("bind oper log search: %v", err)
	}
	if q.Title != "用户" || q.BusinessType != "2" {
		t.Fatalf("basic field mismatch: %+v", q)
	}
	// 模拟 api 层 GetOperLogList 的显式取值
	q.BeginTime = c.Query("params[beginTime]")
	q.EndTime = c.Query("params[endTime]")
	if q.BeginTime != "2026-07-01 00:00:00" || q.EndTime != "2026-07-19 23:59:59" {
		t.Fatalf("c.Query(params[...]) mismatch: begin=%q end=%q", q.BeginTime, q.EndTime)
	}
}

func mustParseQuery(raw string) *url.URL {
	u, err := url.Parse("http://localhost/log/operlog/list?" + raw)
	if err != nil {
		panic(err)
	}
	return u
}
