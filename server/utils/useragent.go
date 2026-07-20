package utils

import (
	"strings"

	"github.com/mssola/user_agent"
)

// ParseUserAgent 解析 User-Agent 字符串,提取浏览器类型、操作系统、设备类型,
// 供登录日志 SysLoginLog 的 Browser/Os/DeviceType 字段使用。
//
// 返回值:
//   - browser     浏览器名(如 "Chrome",不带版本);解析不出浏览器名时回退为原始 UA,避免丢失信息
//   - os          操作系统(如 "Windows 10"/"Android 13"/"iOS");解析失败为空
//   - deviceType  设备类型,对齐前端 System.DeviceType: pc/android/ios/xcx;解析失败为空
//
// 入参为空字符串时三项均返回空。
func ParseUserAgent(ua string) (browser, os, deviceType string) {
	if strings.TrimSpace(ua) == "" {
		return "", "", ""
	}
	parsed := user_agent.New(ua)

	name, _ := parsed.Browser()
	name = strings.TrimSpace(name)
	if name == "" {
		browser = ua // 解析不出浏览器名,回退原始 UA
	} else {
		browser = name
	}

	os = strings.TrimSpace(parsed.OS())
	deviceType = classifyDevice(ua, parsed.Platform(), os, parsed.Mobile())
	return browser, os, deviceType
}

// classifyDevice 按前端契约 System.DeviceType(pc/android/ios/xcx)归类设备。
//   - xcx: 微信小程序(UA 含 miniprogram 特征)
//   - android: Android 手机/平板
//   - ios: iPhone/iPad(iOS 13+ 桌面态 UA platform 为 Macintosh 但仍判为移动)
//   - pc: 其余(Windows/macOS/Linux 桌面)
func classifyDevice(ua, platform, os string, mobile bool) string {
	uaLower := strings.ToLower(ua)
	platformLower := strings.ToLower(platform)
	osLower := strings.ToLower(os)

	switch {
	case strings.Contains(uaLower, "miniprogram"):
		return "xcx"
	case strings.Contains(osLower, "android") || strings.Contains(platformLower, "android"):
		return "android"
	case strings.Contains(platformLower, "iphone") || strings.Contains(platformLower, "ipad") ||
		strings.Contains(osLower, "ios") || strings.Contains(osLower, "iphone") || strings.Contains(osLower, "ipad"):
		return "ios"
	case mobile && strings.Contains(platformLower, "mac"):
		return "ios"
	default:
		return "pc"
	}
}
