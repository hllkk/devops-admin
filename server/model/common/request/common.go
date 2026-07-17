package request

import (
	"strconv"

	"gorm.io/gorm"
)

const MaxPageSize = 100

// PageInfo Paging common input parameter structure
type PageInfo struct {
	PageNum  int    `json:"pageNum" form:"pageNum"`   // 页码
	PageSize int    `json:"pageSize" form:"pageSize"` // 每页大小
	Keyword  string `json:"keyword" form:"keyword"`   // 关键字
}

func (r *PageInfo) Paginate() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if r.PageNum <= 0 {
			r.PageNum = 1
		}
		switch {
		case r.PageSize > MaxPageSize:
			r.PageSize = MaxPageSize
		case r.PageSize <= 0:
			r.PageSize = 10
		}
		offset := (r.PageNum - 1) * r.PageSize
		return db.Offset(offset).Limit(r.PageSize)
	}
}

func (r *PageInfo) LimitOffset() (limit, offset int) {
	limit = r.PageSize
	if limit > MaxPageSize {
		limit = MaxPageSize
	}
	if limit <= 0 {
		return 0, 0
	}
	page := r.PageNum
	if page <= 0 {
		page = 1
	}
	return limit, limit * (page - 1)
}

// GetById Find by id structure (雪花 int64,string 对齐前端 IdType)
type GetById struct {
	ID int64 `json:"id,string" form:"id"` // 主键ID(雪花int64,JSON传输为string)
}

// Int64 返回 int64 主键,供 GORM / service 直接消费
func (r *GetById) Int64() int64 {
	return r.ID
}

// IdsReq 批量 ID 请求(string 对齐前端雪花 ID 传输契约,Int64s() 转 []int64 供 GORM)
type IdsReq struct {
	Ids []string `json:"ids" form:"ids"` // ID列表(string对齐前端,雪花ID传输为string)
}

// Int64s 将 string IDs 转为 []int64,跳过无法解析的项
func (r *IdsReq) Int64s() []int64 {
	result := make([]int64, 0, len(r.Ids))
	for _, id := range r.Ids {
		n, err := strconv.ParseInt(id, 10, 64)
		if err == nil {
			result = append(result, n)
		}
	}
	return result
}

// GetRoleId Get role by id structure (雪花 int64,string 对齐前端 IdType)
type GetRoleId struct {
	RoleId int64 `json:"roleId,string" form:"roleId"` // 角色ID(雪花int64,JSON传输为string)
}

type Empty struct{}
