package request

import (
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

// GetById Find by id structure
type GetById struct {
	ID int `json:"id" form:"id"` // 主键ID
}

func (r *GetById) Uint() uint {
	return uint(r.ID)
}

type IdsReq struct {
	Ids []int `json:"ids" form:"ids"`
}

// GetRoleId Get role by id structure
type GetRoleId struct {
	RoleId uint `json:"roleId" form:"roleId"` // 角色ID
}

type Empty struct{}
