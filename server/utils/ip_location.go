package utils

import (
	"fmt"
	"net"
	"strings"

	"github.com/lionsoul2014/ip2region/binding/golang/service"
)

// 默认 ip2region xdb 路径(相对 server 运行目录)。
const defaultIp2RegionDbPath = "resource/ip2region.xdb"

// ipRegionSvc 全局 ip2region 查询服务(BufferCache 全内存,单实例 searcher 天生并发安全)。
// 由 LoadIp2Region 在启动阶段赋值,运行期只读;为 nil 表示加载失败或未初始化,ParseIPLocation 将降级。
var ipRegionSvc *service.Ip2Region

// LoadIp2Region 初始化 ip2region 查询服务(BufferCache 全内存策略:单实例内存 searcher,
// 天生并发安全,查询 μs 级,无文件 IO)。仅启用 IPv4(v6Config=nil),IPv6 地址查询返回空。
//
// 文件缺失/校验失败时返回 error,调用方应降级处理(记日志、不阻断启动):
// 此时 ipRegionSvc 为 nil,ParseIPLocation 对公网 IP 返回"未知"。
// 在 initialize.OtherInit 中调用一次。
func LoadIp2Region(dbPath string) error {
	if strings.TrimSpace(dbPath) == "" {
		dbPath = defaultIp2RegionDbPath
	}
	// BufferCache:全内存加载;该策略下 searchers 数量无效(单实例内存 searcher),传 1。
	v4Config, err := service.NewV4Config(service.BufferCache, dbPath, 1)
	if err != nil {
		return fmt.Errorf("加载 xdb 失败(%s): %w", dbPath, err)
	}
	svc, err := service.NewIp2Region(v4Config, nil) // 仅启用 IPv4
	if err != nil {
		return fmt.Errorf("创建查询服务失败: %w", err)
	}
	ipRegionSvc = svc
	return nil
}

// ParseIPLocation 解析 IP 为地点字符串,供登录日志 SysLoginLog.LoginLocation /
// 操作日志 SysOperLog.OperLocation 使用。
//
// 返回 ip2region 原生格式「国家|区域|城市|ISP|国家码」,如 "中国|广东省|深圳市|电信|CN"。
//   - 入参为空 → 返回 ""(调用方按需决定是否写库)
//   - 私有/回环/链路本地/未指定 IP → "内网IP"(不查库,对齐「内网IP不留空」诉求)
//   - 非法 IP、公网查询不到或服务未就绪 → "未知"(不留空)
//
// 并发安全:底层 service.Ip2Region.Search 天生并发安全,可直接在请求链路调用。
func ParseIPLocation(ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ""
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return "未知"
	}
	if parsed.IsLoopback() || parsed.IsPrivate() || parsed.IsLinkLocalUnicast() || parsed.IsUnspecified() {
		return "内网IP"
	}
	if ipRegionSvc == nil {
		return "未知" // 服务未就绪(加载失败降级),不留空
	}
	region, err := ipRegionSvc.Search(ip)
	if err != nil || region == "" {
		return "未知"
	}
	return region
}
