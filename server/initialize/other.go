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
	"go.uber.org/zap"
)

// insecureDefaultJWTKey 为 conf/config.yaml 中的开发默认密钥（可能随仓库公开，生产环境禁止使用）。
// 启动校验会拒绝生产环境继续使用该密钥，强制通过环境变量 JWT_SIGNING_KEY 注入。
const insecureDefaultJWTKey = "2aa81975-ecfd-44bd-817b-809152efaafb"

func OtherInit() {
	// 应用环境变量覆盖（JWT/MySQL/Redis 等敏感项）
	// viper.AutomaticEnv 对 Unmarshal 不可靠，故在此手写覆盖，确保生产环境经环境变量注入的真实值生效
	applyEnvOverrides()

	// 校验签名密钥非空（占位符或环境变量值均可，禁止空密钥启动）
	if global.OPS_CONFIG.JWT.SigningKey == "" {
		zap.L().Fatal("JWT signing-key 为空，请设置环境变量 JWT_SIGNING_KEY 或在配置文件中提供")
	}

	// 生产环境（GIN_MODE=release）强制安全配置
	validateProductionSecurity()

	// 校验各时间字段为合法的 duration 字符串
	requireDuration("expires-time", global.OPS_CONFIG.JWT.ExpiresTime)
	requireDuration("buffer-time", global.OPS_CONFIG.JWT.BufferTime)
	requireDuration("refresh-ex-time", global.OPS_CONFIG.JWT.RefreshExTime)

	// JWT 黑名单本地缓存（仅作为 Redis 不可用时的进程级兜底；Redis 可用时主存储在 Redis+DB）
	dr, err := utils.ParseDuration(global.OPS_CONFIG.JWT.ExpiresTime)
	if err != nil {
		zap.L().Fatal("JWT expires-time 解析失败", zap.String("value", global.OPS_CONFIG.JWT.ExpiresTime), zap.Error(err))
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

// requireDuration 校验 duration 字符串合法，否则 Fatal 终止启动。
func requireDuration(field, value string) {
	if _, err := utils.ParseDuration(value); err != nil {
		zap.L().Fatal("JWT 配置无效: "+field, zap.String("value", value), zap.Error(err))
	}
}

// validateProductionSecurity 生产环境（GIN_MODE=release）强制安全配置：
//   - JWT 密钥不得为已泄露的开发默认值或短于 32 字符；
//   - 加密密钥应独立于 JWT 密钥；
//   - 未配置 TrustedProxies 时提醒。
func validateProductionSecurity() {
	isRelease := os.Getenv("GIN_MODE") == "release"

	// JWT 密钥校验
	if key := global.OPS_CONFIG.JWT.SigningKey; key == insecureDefaultJWTKey || len(key) < 32 {
		if isRelease {
			zap.L().Fatal("生产环境必须设置环境变量 JWT_SIGNING_KEY（至少 32 字符，禁止使用配置文件默认密钥）")
		}
		zap.L().Warn("正在使用配置文件默认/弱密钥，生产环境请设置环境变量 JWT_SIGNING_KEY")
	}

	// 加密密钥校验 — 生产环境应独立设置 SECRET_ENCRYPT_KEY
	if global.OPS_CONFIG.System.SecretEncryptKey == "" {
		if isRelease {
			zap.L().Warn("未设置 SECRET_ENCRYPT_KEY，回退使用 JWT_SIGNING_KEY 作为加密密钥；若 JWT 密钥变更，数据库中已加密的敏感配置将无法解密。建议通过环境变量 SECRET_ENCRYPT_KEY 注入独立密钥（生成: openssl rand -hex 32）")
		} else {
			zap.L().Info("未设置 SECRET_ENCRYPT_KEY，回退使用 JWT_SIGNING_KEY 作为加密密钥（开发环境可接受，生产环境建议独立设置）")
		}
	} else if len(global.OPS_CONFIG.System.SecretEncryptKey) < 32 {
		if isRelease {
			zap.L().Fatal("SECRET_ENCRYPT_KEY 至少需要 32 字符（AES-256 密钥派生要求）")
		}
		zap.L().Warn("SECRET_ENCRYPT_KEY 短于 32 字符，安全强度不足")
	}

	// 信任代理提醒
	if isRelease && len(global.OPS_CONFIG.System.TrustedProxies) == 0 {
		zap.L().Warn("生产环境未配置 TRUSTED_PROXIES，ClientIP 将取直连 RemoteAddr；反向代理部署时请配置内网网段以正确解析真实客户端 IP")
	}
}

// applyEnvOverrides 用环境变量覆盖敏感配置项（仅当对应 env 非空时覆盖，保留配置文件回退）。
func applyEnvOverrides() {
	// JWT 签名密钥
	if v := os.Getenv("JWT_SIGNING_KEY"); v != "" {
		global.OPS_CONFIG.JWT.SigningKey = v
		global.OPS_LOG.Info("配置项从环境变量加载", zap.String("key", "JWT_SIGNING_KEY"))
	}
	// 数据库敏感字段加密密钥
	if v := os.Getenv("SECRET_ENCRYPT_KEY"); v != "" {
		global.OPS_CONFIG.System.SecretEncryptKey = v
		global.OPS_LOG.Info("配置项从环境变量加载", zap.String("key", "SECRET_ENCRYPT_KEY"))
	}
	// MySQL 密码
	if v := os.Getenv("MYSQL_PASSWORD"); v != "" {
		global.OPS_CONFIG.Mysql.Password = v
		global.OPS_LOG.Info("配置项从环境变量加载", zap.String("key", "MYSQL_PASSWORD"))
	}
	// PostgreSQL 密码
	if v := os.Getenv("PG_PASSWORD"); v != "" {
		global.OPS_CONFIG.Pgsql.Password = v
		global.OPS_LOG.Info("配置项从环境变量加载", zap.String("key", "PG_PASSWORD"))
	}
	// Redis 密码（主实例 + redis-list 所有实例）
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		global.OPS_CONFIG.Redis.Password = v
		for i := range global.OPS_CONFIG.RedisList {
			global.OPS_CONFIG.RedisList[i].Password = v
		}
		global.OPS_LOG.Info("配置项从环境变量加载", zap.String("key", "REDIS_PASSWORD"))
	}
	// 初始化密钥
	if v := os.Getenv("INIT_KEY"); v != "" {
		global.OPS_CONFIG.System.InitKey = v
		global.OPS_LOG.Info("配置项从环境变量加载", zap.String("key", "INIT_KEY"))
	}
	// 严格设备绑定
	if v := os.Getenv("STRICT_DEVICE_BINDING"); v != "" {
		global.OPS_CONFIG.System.StrictDeviceBinding = (v == "true" || v == "1")
		global.OPS_LOG.Info("配置项从环境变量加载", zap.String("key", "STRICT_DEVICE_BINDING"), zap.Bool("value", global.OPS_CONFIG.System.StrictDeviceBinding))
	}
	// 可信反代列表（逗号分隔 CIDR/IP）
	if v := os.Getenv("TRUSTED_PROXIES"); v != "" {
		tp := make([]string, 0, 4)
		for _, p := range strings.Split(v, ",") {
			if s := strings.TrimSpace(p); s != "" {
				tp = append(tp, s)
			}
		}
		global.OPS_CONFIG.System.TrustedProxies = tp
		global.OPS_LOG.Info("配置项从环境变量加载", zap.String("key", "TRUSTED_PROXIES"), zap.Strings("value", tp))
	}
}
