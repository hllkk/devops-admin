package request

import (
	"net/url"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
)

// NormalizeEmptyBoolQuery 修正 Gin 对 query 空串 *bool 字段的绑定。
//
// 背景: Gin 的 form binding 见到 query 里存在的 key(即便值为空串)会把 *bool
// 绑成 &false —— setBoolField 把空串当 "false", 再由 Ptr 分支先 reflect.New
// 分配非 nil 指针后递归写入。这击穿了 *bool 用 nil 区分"未传筛选 vs 传 false"
// 的设计意图, 导致列表默认带上 WHERE is_active = false, 查不到启用(true)数据。
//
// 用法: 在 API 层 c.ShouldBindQuery(&q) 成功后调用本函数。它遍历 ptr 指向的
// 结构体(含嵌入字段)中每个带 form tag 的 *bool 字段, 若其 query key 存在且值
// 为空串, 则把该 *bool 置 nil, 还原"未传筛选"语义; 显式 true/false 不动。
func NormalizeEmptyBoolQuery(c *gin.Context, ptr any) {
	if c == nil || c.Request == nil || ptr == nil {
		return
	}
	rv := reflect.ValueOf(ptr)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return
	}
	query := c.Request.URL.Query()
	normalizeEmptyBoolStruct(rv.Elem(), query)
}

// normalizeEmptyBoolStruct 递归遍历结构体(含嵌入), 归一空串 *bool 字段。
func normalizeEmptyBoolStruct(v reflect.Value, q url.Values) {
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		fv := v.Field(i)
		if sf.Anonymous {
			// 嵌入字段递归: 指针先解引用, 结构体直接进
			switch fv.Kind() {
			case reflect.Ptr:
				if !fv.IsNil() {
					normalizeEmptyBoolStruct(fv.Elem(), q)
				}
			case reflect.Struct:
				normalizeEmptyBoolStruct(fv, q)
			}
			continue
		}
		if !fv.CanSet() {
			continue
		}
		// 只处理 *bool(指针指向 bool)
		if fv.Kind() != reflect.Ptr || fv.Type().Elem().Kind() != reflect.Bool {
			continue
		}
		key := formTagKey(sf)
		if key == "" || key == "-" {
			continue
		}
		// query 里该 key 存在且值为空串 → 置 nil
		if vs, ok := q[key]; ok && (len(vs) == 0 || vs[0] == "") {
			fv.Set(reflect.Zero(fv.Type()))
		}
	}
}

// formTagKey 取 form tag 逗号前的 key; 无 form tag 回退字段名(对齐 Gin 默认)。
func formTagKey(sf reflect.StructField) string {
	raw := sf.Tag.Get("form")
	key := strings.SplitN(raw, ",", 2)[0]
	if key == "" {
		key = sf.Name
	}
	return key
}
