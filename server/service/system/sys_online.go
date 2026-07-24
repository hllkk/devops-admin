package system

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// OnlineService 在线设备/会话管理。
// 会话存 Redis hash(key=online:session:<userId>,field=tokenId,value=OnlineSession JSON),
// 不入库;token 失效(过期/黑名单)靠列表查询兜底清理 + 登出/踢下线主动删除。
type OnlineService struct{}

// onlineSessionKey 拼接当前用户的会话 hash key。
func onlineSessionKey(userId int64) string {
	return "online:session:" + strconv.FormatInt(userId, 10)
}

// RecordSession 登录成功时记录一条在线会话到 Redis。
// deptName 按 deptId 查部门名补齐(登录快照);整 key 的 TTL 刷新为 jwt 过期时间。
// 失败仅记日志,不阻断登录(会话记录是辅助能力)。
func (s *OnlineService) RecordSession(ctx context.Context, userId, deptId int64, sess system.OnlineSession) error {
	if userId == 0 || sess.TokenId == "" {
		return errors.New("userId/tokenId 不能为空")
	}
	if deptId > 0 && sess.DeptName == "" {
		// 独立 tx,不污染外层 db;只取部门名快照
		var dept system.SysDepartment
		if err := global.OPS_DB.WithContext(ctx).Select("dept_name").Where("dept_id = ?", deptId).First(&dept).Error; err == nil {
			sess.DeptName = dept.DeptName
		}
	}
	data, err := json.Marshal(sess)
	if err != nil {
		return err
	}
	key := onlineSessionKey(userId)
	if err := global.OPS_REDIS.HSet(ctx, key, sess.TokenId, string(data)).Err(); err != nil {
		logger.WithCtx(ctx).Mod("biz").Err(err).Error("记录在线会话失败")
		return err
	}
	if dr, _ := utils.ParseDuration(global.OPS_CONFIG.JWT.ExpiresTime); dr > 0 {
		global.OPS_REDIS.Expire(ctx, key, dr)
	}
	return nil
}

// ListSessions 列出当前用户的有效在线会话。
// 遍历 hash,对每条会话:JSON 损坏 / ParseToken 失败(过期) / 在 jwt 黑名单(OPS_CACHE)
// → 视为失效,HDEL 清理并跳过;通过的按 ipaddr 模糊过滤,按 LoginTime 降序返回。
func (s *OnlineService) ListSessions(ctx context.Context, userId int64, ipaddr string) (list []system.OnlineSession, err error) {
	list = make([]system.OnlineSession, 0)
	if userId == 0 {
		return
	}
	key := onlineSessionKey(userId)
	fields, err := global.OPS_REDIS.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	j := utils.NewJWT()
	for tokenId, raw := range fields {
		var sess system.OnlineSession
		if e := json.Unmarshal([]byte(raw), &sess); e != nil {
			global.OPS_REDIS.HDel(ctx, key, tokenId)
			continue
		}
		// token 过期/解析失败 → 清理
		if _, e := j.ParseToken(sess.Token); e != nil {
			global.OPS_REDIS.HDel(ctx, key, tokenId)
			continue
		}
		// 已入 jwt 黑名单 → 清理(复用 LoadAll + JsonInBlacklist 写入 OPS_CACHE 的机制)
		if _, ok := global.OPS_CACHE.Get(sess.Token); ok {
			global.OPS_REDIS.HDel(ctx, key, tokenId)
			continue
		}
		if ipaddr != "" && !strings.Contains(sess.Ipaddr, ipaddr) {
			continue
		}
		list = append(list, sess)
	}
	sort.Slice(list, func(i, k int) bool {
		return list[i].LoginTime.After(list[k].LoginTime)
	})
	return
}

// GetOnlineList 当前用户在线设备分页列表(对齐前端 GET /monitor/online)。
// 过滤后总数=total,分页走 PageInfo.LimitOffset(MaxPageSize=100 截断);limit<=0 时返回全部。
func (s *OnlineService) GetOnlineList(ctx context.Context, userId int64, q systemReq.OnlineSearch) (devices []system.OnlineDevice, total int64, err error) {
	sessions, err := s.ListSessions(ctx, userId, q.Ipaddr)
	if err != nil {
		return nil, 0, err
	}
	total = int64(len(sessions))
	limit, offset := q.LimitOffset()
	devices = make([]system.OnlineDevice, 0)
	if limit == 0 {
		for _, ss := range sessions {
			devices = append(devices, toDevice(ss))
		}
		return devices, total, nil
	}
	end := offset + limit
	if end > len(sessions) {
		end = len(sessions)
	}
	if offset > len(sessions) {
		offset = len(sessions)
	}
	for _, ss := range sessions[offset:end] {
		devices = append(devices, toDevice(ss))
	}
	return devices, total, nil
}

// RemoveSession 删除当前用户的指定会话(登出/互踢用)。tokenId 不存在时静默返回。
func (s *OnlineService) RemoveSession(ctx context.Context, userId int64, tokenId string) error {
	if userId == 0 || tokenId == "" {
		return nil
	}
	return global.OPS_REDIS.HDel(ctx, onlineSessionKey(userId), tokenId).Err()
}

// UpdateSessionToken 续签后同步更新会话里的 token,使踢下线能命中当前有效 token。
// 会话不存在/解析失败时静默返回(不阻断续签)。
func (s *OnlineService) UpdateSessionToken(ctx context.Context, userId int64, tokenId, newToken string) error {
	if userId == 0 || tokenId == "" || newToken == "" {
		return nil
	}
	key := onlineSessionKey(userId)
	raw, err := global.OPS_REDIS.HGet(ctx, key, tokenId).Result()
	if err != nil {
		return nil
	}
	var sess system.OnlineSession
	if e := json.Unmarshal([]byte(raw), &sess); e != nil {
		return nil
	}
	sess.Token = newToken
	data, e := json.Marshal(sess)
	if e != nil {
		return e
	}
	return global.OPS_REDIS.HSet(ctx, key, tokenId, string(data)).Err()
}

// KickSession 踢下线指定会话:校验属于当前用户后,token 入 jwt 黑名单 + 删会话记录。
// tokenId 不属于该用户(hash 查不到)→ 返回错误,天然防止踢他人设备。
func (s *OnlineService) KickSession(ctx context.Context, userId int64, tokenId string) error {
	if userId == 0 || tokenId == "" {
		return errors.New("参数不能为空")
	}
	key := onlineSessionKey(userId)
	raw, err := global.OPS_REDIS.HGet(ctx, key, tokenId).Result()
	if err != nil {
		return errors.New("设备已不在线")
	}
	var sess system.OnlineSession
	if e := json.Unmarshal([]byte(raw), &sess); e != nil {
		return errors.New("会话数据异常")
	}
	if sess.Token != "" {
		if e := JwtServiceApp.JsonInBlacklist(ctx, system.JwtBlacklist{Jwt: sess.Token}); e != nil {
			return e
		}
	}
	return global.OPS_REDIS.HDel(ctx, key, tokenId).Err()
}

// RevokeUserSessions 吊销指定用户的所有活跃会话(改密 / 重置密码成功后调用)。
// 复用 KickSession 逐个踢单会话:token 入黑名单 + 删会话记录,旧 token 立即失效。
// 当前调用方若持新 token(不同 jti)不受影响;单条失败不中断其余会话的吊销。
func (s *OnlineService) RevokeUserSessions(ctx context.Context, userId int64) error {
	if userId == 0 {
		return nil
	}
	sessions, err := s.ListSessions(ctx, userId, "")
	if err != nil {
		return err
	}
	for _, sess := range sessions {
		_ = s.KickSession(ctx, userId, sess.TokenId)
	}
	return nil
}

// toDevice 会话 → 对外设备视图(loginTime 转毫秒时间戳,对齐前端 number)。
func toDevice(s system.OnlineSession) system.OnlineDevice {
	return system.OnlineDevice{
		UserName:      s.UserName,
		Ipaddr:        s.Ipaddr,
		LoginLocation: s.LoginLocation,
		Browser:       s.Browser,
		Os:            s.Os,
		DeptName:      s.DeptName,
		DeviceType:    s.DeviceType,
		LoginTime:     s.LoginTime.UnixMilli(),
		TokenId:       s.TokenId,
	}
}
