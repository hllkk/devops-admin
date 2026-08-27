package common

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type JSONMap map[string]any

type JSONSlice[T any] []T

var mysqlJSONTypeCache sync.Map

func (JSONMap) GormDataType() string {
	return "json"
}

func (JSONMap) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	return resolveJSONDBDataType(db)
}

func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

func (m *JSONMap) Scan(value any) error {
	if value == nil {
		*m = make(map[string]any)
		return nil
	}
	var err error
	switch typed := value.(type) {
	case []byte:
		err = json.Unmarshal(typed, m)
	case string:
		err = json.Unmarshal([]byte(typed), m)
	default:
		err = errors.New("basetypes.JSONMap.Scan: invalid value type")
	}
	if err != nil {
		return err
	}
	return nil
}

func (JSONSlice[T]) GormDataType() string {
	return "json"
}

func (JSONSlice[T]) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	return resolveJSONDBDataType(db)
}

func (s JSONSlice[T]) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

func (s *JSONSlice[T]) Scan(value any) error {
	if value == nil {
		*s = JSONSlice[T]{}
		return nil
	}
	var err error
	switch typed := value.(type) {
	case []byte:
		err = json.Unmarshal(typed, s)
	case string:
		err = json.Unmarshal([]byte(typed), s)
	default:
		err = errors.New("basetypes.JSONSlice.Scan: invalid value type")
	}
	if err != nil {
		return err
	}
	return nil
}

func resolveJSONDBDataType(db *gorm.DB) string {
	switch db.Dialector.Name() {
	case "mysql":
		if mysqlSupportsJSON(db) {
			return "JSON"
		}
		return "LONGTEXT"
	case "postgres":
		return "JSONB"
	default:
		return "JSON"
	}
}

func mysqlSupportsJSON(db *gorm.DB) bool {
	sqlDB, err := db.DB()
	if err != nil {
		return true
	}

	cacheKey := sqlDB
	if cached, ok := mysqlJSONTypeCache.Load(cacheKey); ok {
		return cached.(bool)
	}

	supports := true
	var version string
	if err := db.Raw("SELECT VERSION()").Scan(&version).Error; err == nil {
		lowerVersion := strings.ToLower(version)
		if strings.Contains(lowerVersion, "mariadb") {
			supports = false
		} else if strings.HasPrefix(version, "5.5.") || strings.HasPrefix(version, "5.6.") {
			supports = false
		}
	}

	mysqlJSONTypeCache.Store(cacheKey, supports)
	return supports
}

type TreeNode[T any] interface {
	GetChildren() []T
	SetChildren(children T)
	GetID() int
	GetParentID() int
}

// Int64String 兼容前端 IdType(string|number)与空串的 int64 字段(雪花 id 入参适配)。
// 容忍 "" / null / 数字 / "字符串数字",统一解析为 int64(空/null→0);用于请求体绑定,
// 避免前端空串(如顶层 parentId "")或 number/string 混合导致普通 int64 字段绑定失败。
type Int64String int64

// UnmarshalJSON 容忍空串/null/数字/带引号数字串。
func (v *Int64String) UnmarshalJSON(data []byte) error {
	s := strings.TrimSpace(strings.Trim(string(data), `"`))
	if s == "" || s == "null" {
		*v = 0
		return nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	*v = Int64String(n)
	return nil
}

// Int64 取 int64 值。
func (v Int64String) Int64() int64 { return int64(v) }

// Int64StringSlice 将 []int64 序列化为 JSON 字符串数组(雪花 id 出参适配)。
// 项目约定雪花 id 输出 string(树节点 id 等用 ",string" tag);但 Go encoding/json 的 ",string" tag 对 slice 无效,
// 故自定义 MarshalJSON 逐元素转字符串,保证响应内 checkedKeys 等与同响应节点 id 类型一致,
// 避免前端 NTree 用 string key 匹配 number 失败导致勾选不回显。
type Int64StringSlice []int64

// MarshalJSON 逐元素转字符串输出(如 [1,2] → ["1","2"])。
func (s Int64StringSlice) MarshalJSON() ([]byte, error) {
	strs := make([]string, len(s))
	for i, v := range s {
		strs[i] = strconv.FormatInt(v, 10)
	}
	return json.Marshal(strs)
}

// UnmarshalJSON 兼容字符串/数字/混合元素的数组入参(雪花 id 前端 IdType=string|number,
// 批量传参常为字符串数组;null 元素按 0 处理,查询无副作用)。
func (s *Int64StringSlice) UnmarshalJSON(data []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	out := make([]int64, 0, len(raw))
	for _, r := range raw {
		v := Int64String(0)
		if err := v.UnmarshalJSON(r); err != nil {
			return err
		}
		out = append(out, v.Int64())
	}
	*s = out
	return nil
}
