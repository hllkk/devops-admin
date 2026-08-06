package initialize

import (
	"bufio"
	"log"
	"os"
	"strings"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/utils"
)

// insecureDefaultJWTKey 为 config.yaml 中的开发默认密钥（已随仓库公开，生产环境禁止使用）。
// 启动校验会拒绝生产环境（GIN_MODE=release）继续使用该密钥，强制经环境变量 JWT_SIGNING_KEY 注入。
const insecureDefaultJWTKey = "b51e4e42-0a1f-402b-9146-1697908c54b3"

func OtherInit() {
	// 应用环境变量覆盖（JWT/DB/Redis/对象存储 等敏感项），让 docker-compose 注入的密钥生效
	// 必须在 Gorm()/Redis() 连接之前完成（main.go 中 OtherInit 先于 Gorm 调用）
	applyEnvOverrides()

	// 校验签名密钥非空（占位符或环境变量值均可，禁止空密钥启动）
	if global.OPS_CONFIG.JWT.SigningKey == "" {
		log.Fatal("[FATAL] JWT signing-key 为空，请设置环境变量 JWT_SIGNING_KEY 或在配置文件中提供")
	}

	// 生产环境（GIN_MODE=release）强制安全配置，避免误用开发默认密钥
	validateProductionSecurity()

	// 校验各时间字段为合法的 duration 字符串
	if _, err := utils.ParseDuration(global.OPS_CONFIG.JWT.ExpiresTime); err != nil {
		panic(err)
	}
	if _, err := utils.ParseDuration(global.OPS_CONFIG.JWT.BufferTime); err != nil {
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

// applyEnvOverrides 用环境变量覆盖敏感配置项（仅当对应 env 非空时覆盖，保留配置文件回退）。
//
// 背景：core/viper.go 的 AutomaticEnv 仅对显式 v.Get 生效，对一次性 Unmarshal 到 global.OPS_CONFIG
// 不可靠；而项目所有配置读取都走 global.OPS_CONFIG，故生产环境必须在此手写覆盖，才能让
// docker-compose 注入的密钥/密码真正生效，避免使用配置文件中的开发占位值。
// 这样用户只需维护 .env 一份机密，config.yaml 保持静态入库。
func applyEnvOverrides() {
	if v := os.Getenv("JWT_SIGNING_KEY"); v != "" {
		global.OPS_CONFIG.JWT.SigningKey = v
		log.Println("[INFO] 配置项从环境变量加载: JWT_SIGNING_KEY")
	}
	if v := os.Getenv("PG_PASSWORD"); v != "" {
		global.OPS_CONFIG.Pgsql.Password = v
		log.Println("[INFO] 配置项从环境变量加载: PG_PASSWORD")
	}
	if v := os.Getenv("MYSQL_PASSWORD"); v != "" {
		global.OPS_CONFIG.Mysql.Password = v
		log.Println("[INFO] 配置项从环境变量加载: MYSQL_PASSWORD")
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		global.OPS_CONFIG.Redis.Password = v
		for i := range global.OPS_CONFIG.RedisList {
			global.OPS_CONFIG.RedisList[i].Password = v
		}
		log.Println("[INFO] 配置项从环境变量加载: REDIS_PASSWORD")
	}
	// 对象存储（oss-type=minio 直连 RustFS）：用 .env 的 RUSTFS 凭据覆盖 minio 段
	if v := os.Getenv("RUSTFS_ROOT_USER"); v != "" {
		global.OPS_CONFIG.Minio.AccessKeyId = v
		log.Println("[INFO] 配置项从环境变量加载: RUSTFS_ROOT_USER (minio.access-key-id)")
	}
	if v := os.Getenv("RUSTFS_ROOT_PASSWORD"); v != "" {
		global.OPS_CONFIG.Minio.AccessKeySecret = v
		log.Println("[INFO] 配置项从环境变量加载: RUSTFS_ROOT_PASSWORD (minio.access-key-secret)")
	}
	// 可信反代列表（逗号分隔 CIDR/IP），多层反代下正确解析真实 ClientIP；空则不信任任何代理
	if v := os.Getenv("TRUSTED_PROXIES"); v != "" {
		tp := make([]string, 0, 4)
		for _, p := range strings.Split(v, ",") {
			if s := strings.TrimSpace(p); s != "" {
				tp = append(tp, s)
			}
		}
		global.OPS_CONFIG.System.TrustedProxies = tp
		log.Println("[INFO] 配置项从环境变量加载: TRUSTED_PROXIES")
	}
	// AI 网关 LiteLLM 底座：master-key 敏感，生产由 .env 注入；public-url 按部署域名（经 nginx /llm/ 反代或直暴露 4000）
	if v := os.Getenv("LITELLM_MASTER_KEY"); v != "" {
		global.OPS_CONFIG.Litellm.MasterKey = v
		log.Println("[INFO] 配置项从环境变量加载: LITELLM_MASTER_KEY")
	}
	if v := os.Getenv("LITELLM_PUBLIC_URL"); v != "" {
		global.OPS_CONFIG.Litellm.PublicURL = v
		log.Println("[INFO] 配置项从环境变量加载: LITELLM_PUBLIC_URL")
	}
}

// validateProductionSecurity 生产环境（GIN_MODE=release）强制安全配置：
// JWT 密钥不得为已泄露的开发默认值或短于 32 字符。
// 测试/开发环境仅告警，不阻断启动。
func validateProductionSecurity() {
	isRelease := os.Getenv("GIN_MODE") == "release"
	if key := global.OPS_CONFIG.JWT.SigningKey; key == insecureDefaultJWTKey || len(key) < 32 {
		if isRelease {
			log.Fatal("[FATAL] 生产环境必须设置环境变量 JWT_SIGNING_KEY（至少 32 字符，禁止使用配置文件默认密钥）")
		}
		log.Println("[WARN] 正在使用配置文件默认/弱密钥，生产环境请设置环境变量 JWT_SIGNING_KEY")
	}
	if isRelease && len(global.OPS_CONFIG.System.TrustedProxies) == 0 {
		log.Println("[WARN] 生产环境未配置 TRUSTED_PROXIES，反代部署时 ClientIP 将取直连地址")
	}
}
