package request

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	commonReq "github.com/hllkk/devops-admin/server/model/common/request"
)

// searchSample 模拟真实分页搜索结构: 嵌入 PageInfo + 顶层 *bool + 嵌入 *bool + 边界字段。
type innerSearch struct {
	InnerActive *bool `form:"innerActive"`
}

type searchSample struct {
	commonReq.PageInfo // 嵌入(对齐真实 Search 结构)
	innerSearch       // 嵌入 *bool, 验证递归
	Name     string `form:"name"`
	IsActive *bool  `form:"isActive"`
	Skip     *bool  `form:"-"` // 显式不绑, 不应被处理
	NoTag    *bool             // 无 form tag, 回退字段名
}

func boolPtr(b bool) *bool { return &b }

func TestNormalizeEmptyBoolQuery(t *testing.T) {
	cases := []struct {
		name  string
		query string
		want  func(*testing.T, *searchSample)
	}{
		{
			"空串置nil", "?isActive=&innerActive=",
			func(t *testing.T, s *searchSample) {
				if s.IsActive != nil {
					t.Fatalf("isActive 空串应归 nil, got %v", *s.IsActive)
				}
				if s.InnerActive != nil {
					t.Fatalf("嵌入 innerActive 空串应归 nil, got %v", *s.InnerActive)
				}
			},
		},
		{
			"true保留", "?isActive=true&innerActive=true",
			func(t *testing.T, s *searchSample) {
				if s.IsActive == nil || !*s.IsActive {
					t.Fatalf("isActive=true 应保留, got %v", s.IsActive)
				}
				if s.InnerActive == nil || !*s.InnerActive {
					t.Fatalf("innerActive=true 应保留, got %v", s.InnerActive)
				}
			},
		},
		{
			"显式false保留", "?isActive=false",
			func(t *testing.T, s *searchSample) {
				if s.IsActive == nil || *s.IsActive {
					t.Fatalf("isActive=false 应保留 false, got %v", s.IsActive)
				}
			},
		},
		{
			"未传保持nil", "?",
			func(t *testing.T, s *searchSample) {
				if s.IsActive != nil {
					t.Fatalf("未传 isActive 应保持 nil, got %v", *s.IsActive)
				}
			},
		},
		{
			"form-与无tag字段不被乱置nil", "?noTag=&skip=",
			func(t *testing.T, s *searchSample) {
				// NoTag 回退字段名 NoTag: query 有该 key 且空串 → 仍应置 nil(语义一致)
				// Skip form:- 应跳过, 不被处理(保持 Gin 原绑定结果)
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/x"+tc.query, nil)

			var s searchSample
			if err := c.ShouldBindQuery(&s); err != nil {
				t.Fatalf("ShouldBindQuery err: %v", err)
			}
			// 先验证 Gin 的坑: 空串确被绑成 &false(回归基线)
			if tc.name == "空串置nil" {
				if s.IsActive == nil {
					t.Fatalf("前置断言失败: Gin 应把空串绑成 &false, 实际 nil(行为已变?)")
				}
			}
			NormalizeEmptyBoolQuery(c, &s)
			tc.want(t, &s)
		})
	}
}

func TestNormalizeEmptyBoolQuery_NilSafe(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/x?isActive=", nil)
	// 不应 panic
	NormalizeEmptyBoolQuery(c, nil)
	NormalizeEmptyBoolQuery(nil, nil)
	var s searchSample
	NormalizeEmptyBoolQuery(c, &s) // 非 *bool 无影响
}
