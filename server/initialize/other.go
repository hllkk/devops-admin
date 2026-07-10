package initialize

import (
	"bufio"
	"os"
	"strings"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/utils"

	"github.com/songzhibin97/gkit/cache/local_cache"
)

func OtherInit() {
	dr, err := utils.ParseDuration(global.OPS_CONFIG.JWT.ExpiresTime)
	if err != nil {
		panic(err)
	}
	_, err = utils.ParseDuration(global.OPS_CONFIG.JWT.BufferTime)
	if err != nil {
		panic(err)
	}

	global.BlackCache = local_cache.NewCache(
		local_cache.SetDefaultExpire(dr),
	)
	file, err := os.Open("go.mod")
	if err == nil && global.OPS_CONFIG.AutoCode.Module == "" {
		scanner := bufio.NewScanner(file)
		scanner.Scan()
		global.OPS_CONFIG.AutoCode.Module = strings.TrimPrefix(scanner.Text(), "module ")
	}
}
