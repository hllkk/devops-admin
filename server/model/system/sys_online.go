package system

import "time"

// OnlineDevice 在线设备(对外响应,字段对齐前端 Api.Monitor.OnlineUser)。
// 数据来自 Redis 会话存储(online:session:<userId> hash),不入库。
type OnlineDevice struct {
	UserName      string `json:"userName"`      // 用户账号
	Ipaddr        string `json:"ipaddr"`        // 登录IP地址
	LoginLocation string `json:"loginLocation"` // 登录地点
	Browser       string `json:"browser"`       // 浏览器类型
	Os            string `json:"os"`            // 操作系统
	DeptName      string `json:"deptName"`      // 所在部门
	DeviceType    string `json:"deviceType"`    // 设备类型 pc/android/ios/xcx(对齐前端 System.DeviceType)
	LoginTime     int64  `json:"loginTime"`     // 登录时间(毫秒时间戳,对齐前端 number)
	TokenId       string `json:"tokenId"`       // 令牌ID(jti)
}

// OnlineSession 在线会话(Redis hash value,JSON 存储)。
// Token 明文留存供踢下线时入 jwt 黑名单;仅服务端可见,不对外暴露。
type OnlineSession struct {
	Token         string    `json:"token"`         // 完整 token(踢下线时入黑名单用)
	TokenId       string    `json:"tokenId"`       // jti
	UserName      string    `json:"userName"`      // 用户账号
	Ipaddr        string    `json:"ipaddr"`        // 登录IP
	LoginLocation string    `json:"loginLocation"` // 登录地点
	Browser       string    `json:"browser"`       // 浏览器
	Os            string    `json:"os"`            // 操作系统
	DeviceType    string    `json:"deviceType"`    // 设备类型
	LoginTime     time.Time `json:"loginTime"`     // 登录时间
	DeptName      string    `json:"deptName"`      // 所在部门(登录快照)
}
