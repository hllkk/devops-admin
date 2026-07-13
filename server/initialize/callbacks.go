package initialize

import (
	"context"
	"reflect"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/utils/snowflake"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const snowflakeCallbackName = "ops:snowflake_id"

// RegisterCallbacks 在给定 DB 上注册雪花主键生成回调：创建记录时若整型主键为 0，
// 则自动生成雪花 ID（填充业务模型自定义的整型主键）。幂等——已注册则跳过，热重载等重复调用安全。
func RegisterCallbacks(db *gorm.DB) {
	if db == nil {
		return
	}
	cb := db.Callback().Create()
	if cb.Get(snowflakeCallbackName) != nil {
		return
	}
	if err := cb.Before("gorm:create").Register(snowflakeCallbackName, assignSnowflakeID); err != nil {
		if global.OPS_LOG != nil {
			global.OPS_LOG.Error("注册雪花主键回调失败", zap.Error(err))
		}
	}
}

// assignSnowflakeID 在 gorm:create 之前为整型主键为 0 的记录填充雪花 ID。
func assignSnowflakeID(tx *gorm.DB) {
	if tx.Statement == nil || tx.Statement.Schema == nil {
		return
	}
	pf := tx.Statement.Schema.PrioritizedPrimaryField
	if pf == nil {
		return
	}
	// 仅处理整型主键（业务模型自定义的 int64 主键，回调不依赖具体基座）
	if pf.DataType != schema.Int && pf.DataType != schema.Uint {
		return
	}

	rv := tx.Statement.ReflectValue
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			assignSnowflakeToRecord(pf, tx.Statement.Context, rv.Index(i))
		}
	case reflect.Struct:
		assignSnowflakeToRecord(pf, tx.Statement.Context, rv)
	}
}

// assignSnowflakeToRecord 若记录主键为 0 则生成雪花 ID 写入。
func assignSnowflakeToRecord(pf *schema.Field, ctx context.Context, rv reflect.Value) {
	if !rv.IsValid() {
		return
	}
	fv := pf.ReflectValueOf(ctx, rv)
	if !fv.IsValid() || !fv.IsZero() {
		return
	}
	if err := pf.Set(ctx, rv, snowflake.MustNextID()); err != nil && global.OPS_LOG != nil {
		global.OPS_LOG.Warn("雪花主键赋值失败", zap.Error(err))
	}
}
