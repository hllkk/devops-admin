package initialize

import (
	"testing"

	"github.com/hllkk/devops-admin/server/config"
)

func TestDialRedis_WrongAddr(t *testing.T) {
	_, err := DialRedis(config.Redis{Addr: "127.0.0.1:33999"}) // 33999 几乎必然无人监听
	if err == nil {
		t.Fatal("错误地址应返回连接错误，得到 nil")
	}
}
