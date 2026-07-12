package initialize

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/snowflake"

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

	// 初始化雪花算法节点（MustInit 幂等，热重载重入安全）
	epoch, err := time.Parse(time.RFC3339, global.OPS_CONFIG.Snowflake.Epoch)
	if err != nil {
		panic(fmt.Errorf("解析 snowflake.epoch 失败（需 RFC3339 格式）: %w", err))
	}
	snowflake.MustInit(global.OPS_CONFIG.Snowflake.Node, epoch)
}
