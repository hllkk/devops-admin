package system

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"

	sysReq "github.com/hllkk/devops-admin/server/model/system/request"
)

// PingDB 测试数据库连接：只 ping 不建库、不落盘、无副作用。
func (initDBService *InitDBService) PingDB(conf sysReq.DBConnTest) error {
	if conf.DBType == "sqlite" {
		return pingSqliteDir(conf.DBPath)
	}
	// 复用 request.InitDB 上的 dsn 构造方法（DBConnTest 与 InitDB 的 DB 字段同构）
	ic := sysReq.InitDB{
		DBType: conf.DBType, Host: conf.Host, Port: conf.Port,
		UserName: conf.UserName, Password: conf.Password,
		DBName: conf.DBName, DBPath: conf.DBPath,
	}
	switch conf.DBType {
	case "mysql":
		return pingSQL("mysql", ic.MysqlEmptyDsn())
	case "pgsql":
		return pingSQL("pgx", ic.PgsqlEmptyDsn())
	case "mssql":
		return pingSQL("sqlserver", ic.MssqlEmptyDsn())
	default:
		return fmt.Errorf("不支持的数据库类型: %q", conf.DBType)
	}
}

// pingSQL 打开连接 → Ping → 关闭，不执行任何 SQL、不建库。
func pingSQL(driver, dsn string) error {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return fmt.Errorf("连接失败: %w", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		return fmt.Errorf("连接失败: %w", err)
	}
	return nil
}

// pingSqliteDir 无副作用校验父目录可写（不创建 .db 文件）。
func pingSqliteDir(dbPath string) error {
	if dbPath == "" {
		return errors.New("sqlite 数据库文件路径不能为空")
	}
	f, err := os.CreateTemp(dbPath, ".init-ping-*")
	if err != nil {
		return fmt.Errorf("sqlite 路径不可写: %w", err)
	}
	// 空文件 Close 失败无诊断价值，忽略以保证后续 Remove 清理临时文件。
	_ = f.Close()
	return os.Remove(f.Name())
}

// PingRedis 测试 Redis 连接：建客户端 → Ping → 关闭，不写 global、不落盘。
func (initDBService *InitDBService) PingRedis(conf sysReq.PingRedis) error {
	if conf.Addr == "" {
		return errors.New("redis 地址不能为空")
	}
	client := redis.NewClient(&redis.Options{Addr: conf.Addr, Password: conf.Password, DB: conf.DB})
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis 连接失败: %w", err)
	}
	return nil
}
