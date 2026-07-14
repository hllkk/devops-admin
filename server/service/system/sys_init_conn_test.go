package system

import (
	"path/filepath"
	"testing"

	sysReq "github.com/hllkk/devops-admin/server/model/system/request"
)

func TestPingRedis_WrongAddr(t *testing.T) {
	svc := &InitDBService{}
	if err := svc.PingRedis(sysReq.PingRedis{Addr: "127.0.0.1:33999"}); err == nil {
		t.Fatal("错误 Redis 地址应返回连接错误")
	}
}

func TestPingRedis_Local(t *testing.T) {
	svc := &InitDBService{}
	if err := svc.PingRedis(sysReq.PingRedis{Addr: "127.0.0.1:6379"}); err != nil {
		t.Skipf("本地 Redis 不可用，跳过: %v", err)
	}
}

func TestPingDB_Sqlite_ValidDir(t *testing.T) {
	svc := &InitDBService{}
	dir := t.TempDir()
	if err := svc.PingDB(sysReq.DBConnTest{DBType: "sqlite", DBPath: dir, DBName: "x"}); err != nil {
		t.Fatalf("合法目录应通过: %v", err)
	}
}

func TestPingDB_Sqlite_NotExistDir(t *testing.T) {
	svc := &InitDBService{}
	if err := svc.PingDB(sysReq.DBConnTest{DBType: "sqlite", DBPath: filepath.Join(t.TempDir(), "no-such-subdir"), DBName: "x"}); err == nil {
		t.Fatal("不存在的目录应失败")
	}
}

func TestPingDB_Mysql_WrongAddr(t *testing.T) {
	svc := &InitDBService{}
	err := svc.PingDB(sysReq.DBConnTest{DBType: "mysql", Host: "127.0.0.1", Port: "33999", UserName: "x", Password: "x", DBName: "x"})
	if err == nil {
		t.Fatal("错误 mysql 地址应失败")
	}
}
