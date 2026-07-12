package global

import (
	"time"

	"gorm.io/gorm"
)

type OPS_MODEL struct {
	ID        int64          `gorm:"primaryKey;autoIncrement:false" json:"ID,string"` // 雪花ID（字符串传输，防前端精度丢失）
	CreatedAt time.Time      // 创建时间
	UpdatedAt time.Time      // 更新时间
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // 删除时间
}
