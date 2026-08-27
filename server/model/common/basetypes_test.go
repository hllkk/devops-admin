package common

import (
	"encoding/json"
	"reflect"
	"testing"
)

// Int64StringSlice 入参兼容单测：字符串/数字/混合元素均解析为 []int64(雪花 id 前端 IdType=string|number)。

func TestInt64StringSlice_UnmarshalStringElems(t *testing.T) {
	var s Int64StringSlice
	if err := json.Unmarshal([]byte(`["1234567890123456789","42"]`), &s); err != nil {
		t.Fatalf("字符串数组应解析成功: %v", err)
	}
	if !reflect.DeepEqual([]int64(s), []int64{1234567890123456789, 42}) {
		t.Errorf("解析结果错误: %v", s)
	}
}

func TestInt64StringSlice_UnmarshalNumberElems(t *testing.T) {
	var s Int64StringSlice
	if err := json.Unmarshal([]byte(`[1,2,3]`), &s); err != nil {
		t.Fatalf("数字数组应解析成功: %v", err)
	}
	if !reflect.DeepEqual([]int64(s), []int64{1, 2, 3}) {
		t.Errorf("解析结果错误: %v", s)
	}
}

func TestInt64StringSlice_UnmarshalMixedElems(t *testing.T) {
	var s Int64StringSlice
	if err := json.Unmarshal([]byte(`["7",8]`), &s); err != nil {
		t.Fatalf("混合数组应解析成功: %v", err)
	}
	if !reflect.DeepEqual([]int64(s), []int64{7, 8}) {
		t.Errorf("解析结果错误: %v", s)
	}
}

func TestInt64StringSlice_UnmarshalRejectsGarbage(t *testing.T) {
	var s Int64StringSlice
	if err := json.Unmarshal([]byte(`["abc"]`), &s); err == nil {
		t.Error("非法元素应报错")
	}
}
