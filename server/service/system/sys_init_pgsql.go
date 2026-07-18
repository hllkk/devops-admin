package system

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/gookit/color"
	"github.com/hllkk/devops-admin/server/config"
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system/request"
	"github.com/hllkk/devops-admin/server/utils"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PgsqlInitHandler struct{}

func NewPgsqlInitHandler() *PgsqlInitHandler {
	return &PgsqlInitHandler{}
}

// WriteConfig pgsql 回写配置
func (h PgsqlInitHandler) WriteConfig(ctx context.Context) error {
	c, ok := ctx.Value("config").(config.Pgsql)
	if !ok {
		return errors.New("postgresql config invalid")
	}
	global.OPS_CONFIG.System.DbType = "pgsql"
	global.OPS_CONFIG.Pgsql = c
	global.OPS_CONFIG.JWT.SigningKey = uuid.New().String()
	cs := utils.StructToMap(global.OPS_CONFIG)
	for k, v := range cs {
		global.OPS_VP.Set(k, v)
	}
	global.OPS_ACTIVE_DBNAME = &c.Dbname
	return global.OPS_VP.WriteConfig()
}

// EnsureDB 创建数据库并初始化 pg
func (h PgsqlInitHandler) EnsureDB(ctx context.Context, conf *request.InitDB) (next context.Context, err error) {
	if s, ok := ctx.Value("dbtype").(string); !ok || s != "pgsql" {
		return ctx, ErrDBTypeMismatch
	}

	c := conf.ToPgsqlConfig()
	next = context.WithValue(ctx, "config", c)
	if c.Dbname == "" {
		return ctx, nil
	} // 如果没有数据库名, 则跳出初始化数据

	dsn := conf.PgsqlEmptyDsn()
	// PostgreSQL 不支持 CREATE DATABASE IF NOT EXISTS, 先查 pg_database 判断库是否已存在;
	// 已存在则跳过建库(幂等), 与建表/建数据的探针一致, 避免二次 /initdb 撞 42P04 失败。
	existed, err := pgsqlDatabaseExists(dsn, c.Dbname)
	if err != nil {
		return nil, err
	}
	if !existed {
		var createSql string
		if conf.Template != "" {
			createSql = fmt.Sprintf("CREATE DATABASE %s WITH TEMPLATE %s;", c.Dbname, conf.Template)
		} else {
			createSql = fmt.Sprintf("CREATE DATABASE %s;", c.Dbname)
		}
		if err = createDatabase(dsn, "pgx", createSql); err != nil {
			return nil, err
		} // 创建数据库
	}

	var db *gorm.DB
	if db, err = gorm.Open(postgres.New(postgres.Config{
		DSN:                  c.Dsn(), // DSN data source name
		PreferSimpleProtocol: false,
	}), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true}); err != nil {
		return ctx, err
	}
	global.OPS_CONFIG.AutoCode.Root, _ = filepath.Abs("..")
	next = context.WithValue(next, "db", db)
	return next, err
}

func (h PgsqlInitHandler) InitTables(ctx context.Context, inits initSlice) error {
	return createTables(ctx, inits)
}

func (h PgsqlInitHandler) InitData(ctx context.Context, inits initSlice) error {
	next, cancel := context.WithCancel(ctx)
	defer cancel()
	for i := 0; i < len(inits); i++ {
		if inits[i].DataInserted(next) {
			color.Info.Printf(InitDataExist, Pgsql, inits[i].InitializerName())
			continue
		}
		if n, err := inits[i].InitializeData(next); err != nil {
			color.Info.Printf(InitDataFailed, Pgsql, inits[i].InitializerName(), err)
			return err
		} else {
			next = n
			color.Info.Printf(InitDataSuccess, Pgsql, inits[i].InitializerName())
		}
	}
	color.Info.Printf(InitSuccess, Pgsql)
	return nil
}

// pgsqlDatabaseExists 查询 PostgreSQL 目标库是否已存在。
// 连到默认 postgres 库读取 pg_database: PostgreSQL 不支持 CREATE DATABASE IF NOT EXISTS,
// 故由调用方在建库前用本函数做幂等判断。
func pgsqlDatabaseExists(dsn, dbname string) (bool, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return false, err
	}
	defer func() { _ = db.Close() }()
	if err = db.Ping(); err != nil {
		return false, err
	}
	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbname).Scan(&exists)
	return exists, err
}
