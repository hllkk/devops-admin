package global

import (
	"time"

	"gorm.io/gorm"
)

type OPS_MODEL struct {
	ID          uint           `gorm:"primarykey" json:"ID"` // 主键ID
	CreatedTime time.Time      // 创建时间
	UpdatedTime time.Time      // 更新时间
	DeletedTime gorm.DeletedAt `gorm:"index" json:"-"` // 删除时间
}
