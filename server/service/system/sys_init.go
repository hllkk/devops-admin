package system

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"database/sql"
	"os"

	"github.com/hllkk/devops-admin/server/global"
	sysModel "github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/model/system/request"
	"gorm.io/gorm"
)

// dbReadyCallback 数据库就绪回调函数，由 initialize 包注入
var dbReadyCallback func()

// SetDBReadyCallback 设置数据库就绪回调
func SetDBReadyCallback(callback func()) {
	dbReadyCallback = callback
}

const (
	Mysql           = "mysql"
	Pgsql           = "pgsql"
	Sqlite          = "sqlite"
	Mssql           = "mssql"
	InitSuccess     = "\n[%v] --> 初始数据成功!\n"
	InitDataExist   = "\n[%v] --> %v 的初始数据已存在!\n"
	InitDataFailed  = "\n[%v] --> %v 初始数据失败! \nerr: %+v\n"
	InitDataSuccess = "\n[%v] --> %v 初始数据成功!\n"
)

const (
	InitOrderSystem   = 10
	InitOrderInternal = 1000
	InitOrderExternal = 100000
)

var (
	ErrMissingDBContext        = errors.New("missing db in context")
	ErrMissingDependentContext = errors.New("missing dependent value in context")
	ErrDBTypeMismatch          = errors.New("db type mismatch")
)

// SubInitializer 提供 source/*/init() 使用的接口，每个 initializer 完成一个初始化过程
type SubInitializer interface {
	InitializerName() string // 不一定代表单独一个表，所以改成了更宽泛的语义
	MigrateTable(ctx context.Context) (next context.Context, err error)
	InitializeData(ctx context.Context) (next context.Context, err error)
	TableCreated(ctx context.Context) bool
	DataInserted(ctx context.Context) bool
}

// TypedDBInitHandler 执行传入的 initializer
type TypedDBInitHandler interface {
	EnsureDB(ctx context.Context, conf *request.InitDB) (context.Context, error) // 建库，失败属于 fatal error，因此让它 panic
	WriteConfig(ctx context.Context) error                                       // 回写配置
	InitTables(ctx context.Context, inits initSlice) error                       // 建表 handler
	InitData(ctx context.Context, inits initSlice) error                         // 建数据 handler
}

// orderedInitializer 组合一个顺序字段，以供排序
type orderedInitializer struct {
	order int
	SubInitializer
}

// initSlice 供 initializer 排序依赖时使用
type initSlice []*orderedInitializer

var (
	initializers initSlice
	cache        map[string]*orderedInitializer
)

// RegisterInit 注册要执行的初始化过程，会在 InitDB() 时调用
func RegisterInit(order int, i SubInitializer) {
	if initializers == nil {
		initializers = initSlice{}
	}
	if cache == nil {
		cache = map[string]*orderedInitializer{}
	}
	name := i.InitializerName()
	if _, existed := cache[name]; existed {
		panic(fmt.Sprintf("Name conflict on %s", name))
	}
	ni := orderedInitializer{order, i}
	initializers = append(initializers, &ni)
	cache[name] = &ni
}

/* ---- * service * ---- */

type InitDBService struct{}

// InitDB 创建数据库并初始化 总入口
func (initDBService *InitDBService) InitDB(conf request.InitDB) (err error) {
	ctx := context.TODO()
	ctx = context.WithValue(ctx, "adminPassword", conf.AdminPassword)
	if len(initializers) == 0 {
		return errors.New("无可用初始化过程，请检查初始化是否已执行完成")
	}
	sort.Sort(&initializers) // 保证有依赖的 initializer 排在后面执行
	// Note: 若 initializer 只有单一依赖，可以写为 B=A+1, C=A+1; 由于 BC 之间没有依赖关系，所以谁先谁后并不影响初始化
	// 若存在多个依赖，可以写为 C=A+B, D=A+B+C, E=A+1;
	// C必然>A|B，因此在AB之后执行，D必然>A|B|C，因此在ABC后执行，而E只依赖A，顺序与CD无关，因此E与CD哪个先执行并不影响
	var initHandler TypedDBInitHandler
	switch conf.DBType {
	case "mysql":
		initHandler = NewMysqlInitHandler()
		ctx = context.WithValue(ctx, "dbtype", "mysql")
	case "pgsql":
		initHandler = NewPgsqlInitHandler()
		ctx = context.WithValue(ctx, "dbtype", "pgsql")
	case "sqlite":
		initHandler = NewSqliteInitHandler()
		ctx = context.WithValue(ctx, "dbtype", "sqlite")
	case "mssql":
		initHandler = NewMssqlInitHandler()
		ctx = context.WithValue(ctx, "dbtype", "mssql")
	default:
		initHandler = NewMysqlInitHandler()
		ctx = context.WithValue(ctx, "dbtype", "mysql")
	}
	ctx, err = initHandler.EnsureDB(ctx, &conf)
	if err != nil {
		return err
	}

	db := ctx.Value("db").(*gorm.DB)
	global.OPS_DB = db

	// 初始化雪花算法并注册 GORM 回调(service 包不能 import initialize,直接用 global 版)
	if err = global.InitSnowflake(global.OPS_CONFIG.System.WorkerID); err != nil {
		return err
	}
	if err = global.RegisterSnowflakeCallbacks(db); err != nil {
		return err
	}

	if err = initHandler.InitTables(ctx, initializers); err != nil {
		return err
	}
	if err = initHandler.InitData(ctx, initializers); err != nil {
		return err
	}

	// 把 Redis 配置落到运行时 OPS_CONFIG：各 per-DB handler 的 WriteConfig 会全量 StructToMap 回写，
	// 故 Redis 段与 system.use-redis 随本次落盘，4 个 handler 无需改动。
	applyRedisConfig(conf)

	// Docker 环境下配置由挂载的 config.yaml 管理（敏感项经 env 覆盖，见 initialize/other.go），
	// 跳过回写；本地部署仍回写 config 文件。
	if os.Getenv("DOCKER_ENV") != "true" {
		if err = initHandler.WriteConfig(ctx); err != nil {
			return err
		}
	} else {
		fmt.Println("Docker 环境，跳过配置文件回写（配置由挂载文件管理）")
	}
	// 不清空 initializers:保留供 RegisterTables/Reload 复用同一份建表清单;
	// 二次 /initdb 由各 initializer 的 TableCreated/DataInserted 探针幂等跳过。

	// 通知数据库已就绪，触发插件注册
	if dbReadyCallback != nil {
		dbReadyCallback()
	}

	return nil
}

// applyRedisConfig 把 Redis 配置落到运行时 OPS_CONFIG。
func applyRedisConfig(conf request.InitDB) {
	global.OPS_CONFIG.Redis = conf.ToRedisConfig()
	global.OPS_CONFIG.System.UseRedis = true
}

// createDatabase 创建数据库（ EnsureDB() 中调用 ）
func createDatabase(dsn string, driver string, createSql string) error {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return err
	}
	defer func(db *sql.DB) {
		err = db.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(db)
	if err = db.Ping(); err != nil {
		return err
	}
	_, err = db.Exec(createSql)
	return err
}

// extraTables 为不纳入 initializer 体系的孤儿表(JWT 黑名单 / 错误日志):
// 无种子数据、与角色菜单等无 context 依赖,故集中单独迁移。
// /initdb 的 createTables 末尾统一迁移; 常规重启则由 initialize.RegisterTables 独立清单覆盖。
var extraTables = []any{
	&sysModel.JwtBlacklist{},
	&sysModel.SysError{},
}

// MigrateExtraTables 迁移孤儿表,供 /initdb 的 createTables 调用。
func MigrateExtraTables(db *gorm.DB) error {
	return db.AutoMigrate(extraTables...)
}

// createTables 创建表（默认 dbInitHandler.initTables 行为）
func createTables(ctx context.Context, inits initSlice) error {
	next, cancel := context.WithCancel(ctx)
	defer cancel()
	for _, init := range inits {
		if init.TableCreated(next) {
			continue
		}
		if n, err := init.MigrateTable(next); err != nil {
			return err
		} else {
			next = n
		}
	}
	// 孤儿表不纳入 initializer 体系, 在 initializer 建表后统一迁移。
	// 注: 此为 /initdb 路径独有; 常规重启走 initialize.RegisterTables 的独立清单。
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ErrMissingDBContext
	}
	return MigrateExtraTables(db)
}

/* -- sortable interface -- */

func (a initSlice) Len() int {
	return len(a)
}

func (a initSlice) Less(i, j int) bool {
	return a[i].order < a[j].order
}

func (a initSlice) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
