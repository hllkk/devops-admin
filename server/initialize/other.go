package initialize

import (
	"bufio"
	"log"
	"os"
	"strings"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/utils"
)

func OtherInit() {
	_, err := utils.ParseDuration(global.OPS_CONFIG.JWT.ExpiresTime)
	if err != nil {
		panic(err)
	}
	_, err = utils.ParseDuration(global.OPS_CONFIG.JWT.BufferTime)
	if err != nil {
		panic(err)
	}

	file, err := os.Open("go.mod")
	if err == nil && global.OPS_CONFIG.AutoCode.Module == "" {
		scanner := bufio.NewScanner(file)
		scanner.Scan()
		global.OPS_CONFIG.AutoCode.Module = strings.TrimPrefix(scanner.Text(), "module ")
	}

	// 加载 ip2region(登录/操作日志 IP→地点解析);失败仅告警不阻断,ParseIPLocation 将降级为"未知"。
	// 此处 OPS_LOG 尚未初始化(main.go 中 Zap 在 OtherInit 之后),故用标准 log 输出到 stderr。
	if err := utils.LoadIp2Region(global.OPS_CONFIG.System.Ip2RegionDbPath); err != nil {
		log.Printf("[WARN] ip2region 初始化失败, IP 地点解析将降级为\"未知\": %v", err)
	}
}
