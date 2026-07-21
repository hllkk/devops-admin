package global

import (
	"os"

	"github.com/gin-gonic/gin"
)

// ApplySensitiveEnvAndValidate 用环境变量覆盖 JWT 签名密钥并校验启动安全约束。
//
// DB/Redis/Mongo 等连接配置由初始化向导(/init/initdb → PgsqlInitHandler.WriteConfig 等)
// 回写 config.yaml 管理,每个环境各自初始化,不在此覆盖。
// JWT 签名密钥不进向导流程,且泄露可伪造任意用户 token,故单独由 OPS_JWT_SIGNING_KEY 注入。
//
// 在 core.Viper() 首次 Unmarshal 后、OnConfigChange 回调后、initialize.Reload 后各调用一次。
func ApplySensitiveEnvAndValidate() {
	if v := os.Getenv("OPS_JWT_SIGNING_KEY"); v != "" {
		OPS_CONFIG.JWT.SigningKey = v
	}
	if gin.Mode() == gin.ReleaseMode && OPS_CONFIG.JWT.SigningKey == "" {
		panic("生产环境必须配置 jwt.signing-key(或环境变量 OPS_JWT_SIGNING_KEY)")
	}
}
