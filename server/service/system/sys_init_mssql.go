package system

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/gookit/color"
	"github.com/hllkk/devops-admin/server/config"
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system/request"
	"github.com/hllkk/devops-admin/server/utils"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
)

type MssqlInitHandler struct{}

func NewMssqlInitHandler() *MssqlInitHandler {
	return &MssqlInitHandler{}
}

// WriteConfig mssql回写配置
func (h MssqlInitHandler) WriteConfig(ctx context.Context) error {
	c, ok := ctx.Value("config").(config.Mssql)
	if !ok {
		return errors.New("mssql config invalid")
	}
	global.OPS_CONFIG.System.DbType = "mssql"
	global.OPS_CONFIG.Mssql = c
	global.OPS_CONFIG.JWT.SigningKey = uuid.New().String()
	cs := utils.StructToMap(global.OPS_CONFIG)
	for k, v := range cs {
		global.OPS_VP.Set(k, v)
	}
	global.OPS_ACTIVE_DBNAME = &c.Dbname
	return global.OPS_VP.WriteConfig()
}

// EnsureDB 创建数据库并初始化 mssql
func (h MssqlInitHandler) EnsureDB(ctx context.Context, conf *request.InitDB) (next context.Context, err error) {
	if s, ok := ctx.Value("dbtype").(string); !ok || s != "mssql" {
		return ctx, ErrDBTypeMismatch
	}

	c := conf.ToMssqlConfig()
	next = context.WithValue(ctx, "config", c)
	if c.Dbname == "" {
		return ctx, nil
	} // 如果没有数据库名, 则跳出初始化数据

	dsn := conf.MssqlEmptyDsn()

	mssqlConfig := sqlserver.Config{
		DSN:               dsn, // DSN data source name
		DefaultStringSize: 191, // string 类型字段的默认长度
	}

	var db *gorm.DB

	if db, err = gorm.Open(sqlserver.New(mssqlConfig), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true}); err != nil {
		return nil, err
	}

	global.OPS_CONFIG.AutoCode.Root, _ = filepath.Abs("..")
	next = context.WithValue(next, "db", db)
	return next, err
}

func (h MssqlInitHandler) InitTables(ctx context.Context, inits initSlice) error {
	return createTables(ctx, inits)
}

func (h MssqlInitHandler) InitData(ctx context.Context, inits initSlice) error {
	next, cancel := context.WithCancel(ctx)
	defer cancel()
	for _, init := range inits {
		if init.DataInserted(next) {
			color.Info.Printf(InitDataExist, Mssql, init.InitializerName())
			continue
		}
		if n, err := init.InitializeData(next); err != nil {
			color.Info.Printf(InitDataFailed, Mssql, init.InitializerName(), err)
			return err
		} else {
			next = n
			color.Info.Printf(InitDataSuccess, Mssql, init.InitializerName())
		}
	}
	color.Info.Printf(InitSuccess, Mssql)
	return nil
}
