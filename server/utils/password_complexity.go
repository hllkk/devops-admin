package utils

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/hllkk/devops-admin/server/model/system"
)

// ValidatePasswordComplexity 按安全配置校验密码复杂度 不满足返回可读错误
func ValidatePasswordComplexity(pwd string, cfg system.SysSecurityConfig) error {
	if cfg.PasswordMinLength > 0 && utf8.RuneCountInString(pwd) < cfg.PasswordMinLength {
		return fmt.Errorf("密码长度不能少于 %d 位", cfg.PasswordMinLength)
	}
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range pwd {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}
	var missing []string
	if cfg.PasswordRequireUpper && !hasUpper {
		missing = append(missing, "大写字母")
	}
	if cfg.PasswordRequireLower && !hasLower {
		missing = append(missing, "小写字母")
	}
	if cfg.PasswordRequireDigit && !hasDigit {
		missing = append(missing, "数字")
	}
	if cfg.PasswordRequireSpecial && !hasSpecial {
		missing = append(missing, "特殊字符")
	}
	if len(missing) > 0 {
		return fmt.Errorf("密码必须包含%s", strings.Join(missing, "、"))
	}
	return nil
}
