package global

import (
	"fmt"
	"reflect"
	"time"

	"github.com/yitter/idgenerator-go/idgen"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// 雪花算法(yitter IdGenerator)全局配置常量。
//
//	WorkerIdBitLength=6 -> 单进程可分配 workerID 范围 [0,63];
//	SeqBitLength=6      -> 单毫秒单节点序列号 64 个,足够后台/网盘/AI网关等业务写入。
//
// 多副本部署时各实例的 workerID 必须互异,否则会产生重复 ID(见 config.System.WorkerID)。
const (
	snowflakeWorkerBitLen = uint8(6)
	snowflakeSeqBitLen    = uint8(6)
	snowflakeMaxWorkerID  = (1 << snowflakeWorkerBitLen) - 1 // 63
)

// InitSnowflake 初始化 yitter IdGenerator(进程级全局单例,只需调用一次)。
// workerID 取自 config.System.WorkerID,范围 [0,63]。
// yitter 相比经典 Twitter 雪花内置了时钟回拨/漂移处理,单线程每秒可达百万级。
func InitSnowflake(workerID int) error {
	if workerID < 0 || workerID > snowflakeMaxWorkerID {
		return fmt.Errorf("snowflake workerID must be in [0,%d], got %d", snowflakeMaxWorkerID, workerID)
	}

	opts := idgen.NewIdGeneratorOptions(uint16(workerID))
	opts.BaseTime = time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	opts.WorkerIdBitLength = snowflakeWorkerBitLen
	opts.SeqBitLength = snowflakeSeqBitLen
	idgen.SetIdGenerator(opts) // 全局生效,必须调用
	return nil
}

// NextSnowflakeID 手动获取一个雪花 ID,供非 GORM 写入场景(如手动构造记录)使用。
func NextSnowflakeID() int64 {
	return idgen.NextId()
}

// RegisterSnowflakeCallbacks 为指定 db 注册 Create 前的雪花 ID 回调(钩子名 ops:snowflake_id)。
// 仅当主键(PrioritizedPrimaryField)为 int64 且当前为零值时才填充:
//   - 业务实体(SysUser.UserId 等自定义命名主键)被自动填入雪花 ID;
//   - uint 主键(OPS_MODEL 内部系统表)跳过,继续走 DB 自增。
//
// 需在 InitSnowflake 之后、AutoMigrate/RegisterTables 之前对每个 db 实例调用。
func RegisterSnowflakeCallbacks(db *gorm.DB) error {
	return db.Callback().Create().Before("gorm:create").
		Register("ops:snowflake_id", assignSnowflakeID)
}

// assignSnowflakeID GORM Create 前回调:为零值的 int64 主键分配雪花 ID。
func assignSnowflakeID(db *gorm.DB) {
	if db.Statement == nil || db.Statement.Schema == nil {
		return
	}
	pk := db.Statement.Schema.PrioritizedPrimaryField
	if pk == nil || pk.FieldType.Kind() != reflect.Int64 {
		return // 无主键或非 int64(如 uint 内部表):不干预
	}

	rv := db.Statement.ReflectValue
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			fillSnowflakeID(db, rv.Index(i), pk)
		}
	case reflect.Struct:
		fillSnowflakeID(db, rv, pk)
	}
}

// fillSnowflakeID 为单个结构体元素的零值主键填充雪花 ID。
func fillSnowflakeID(db *gorm.DB, elem reflect.Value, pk *schema.Field) {
	if elem.Kind() == reflect.Ptr {
		elem = elem.Elem()
	}
	if elem.Kind() != reflect.Struct {
		return
	}

	cur := pk.ReflectValueOf(db.Statement.Context, elem)
	if !cur.IsValid() || cur.Int() != 0 {
		return // 非零值(含显式指定的种子数据 ID)不覆盖
	}

	if err := pk.Set(db.Statement.Context, elem, idgen.NextId()); err != nil && OPS_LOG != nil {
		OPS_LOG.Error("snowflake: assign id failed", zap.Error(err))
	}
}
