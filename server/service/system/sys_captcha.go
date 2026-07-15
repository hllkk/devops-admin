package system

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/golang/freetype/truetype"
	"github.com/google/uuid"
	"github.com/songzhibin97/gkit/cache/local_cache"
	"github.com/wenlng/go-captcha-assets/bindata/chars"
	fontAsset "github.com/wenlng/go-captcha-assets/resources/fonts/fzshengsksjw"
	"github.com/wenlng/go-captcha-assets/resources/imagesv2"
	"github.com/wenlng/go-captcha-assets/resources/tiles"
	"github.com/wenlng/go-captcha/v2/base/option"
	"github.com/wenlng/go-captcha/v2/click"
	"github.com/wenlng/go-captcha/v2/rotate"
	"github.com/wenlng/go-captcha/v2/slide"

	"github.com/hllkk/devops-admin/server/global"
)

const (
	captchaTypeClick  = "click"
	captchaTypeSlide  = "slide"
	captchaTypeRotate = "rotate"

	triggerModeThreshold = "threshold" // 失败达阈值才要求验证码
	triggerModeAlways    = "always"    // 始终要求验证码
	triggerModeOff       = "off"       // 关闭

	defaultFailThreshold = 3
	defaultFailWindow    = 600
	defaultExpireSeconds = 120
	defaultTolerance     = 5
	maxVerifyAttempts    = 3 // 单个验证码允许的最大校验失败次数，超出即作废，防暴力试答案
)

// CaptchaService go-captcha 行为验证码服务。
// 支持 click 点选 / slide 滑动 / rotate 旋转三种类型，经 config.captcha.go-captcha.type 切换。
// Service 层不依赖 gin.Context，所需客户端信息（IP）由 API 层传入。
type CaptchaService struct{}

// CaptchaResult 生成接口返回结构，对齐前端验证码组件所需数据。
type CaptchaResult struct {
	CaptchaEnabled bool   `json:"captchaEnabled"` // 当前是否要求验证码（false 时其余字段为空）
	Type           string `json:"type,omitempty"` // click | slide | rotate
	CaptchaId      string `json:"captchaId,omitempty"`
	MasterImage    string `json:"masterImage,omitempty"` // 主图 base64
	TileImage      string `json:"tileImage,omitempty"`   // 拼图块 base64（slide）
	ThumbImage     string `json:"thumbImage,omitempty"`  // 提示缩略图 base64（click/rotate）
	// slide 拼图块渲染参数（对应前端 gocaptcha-slide 的 data）
	ThumbX      int `json:"thumbX,omitempty"`
	ThumbY      int `json:"thumbY,omitempty"`
	ThumbWidth  int `json:"thumbWidth,omitempty"`
	ThumbHeight int `json:"thumbHeight,omitempty"`
	// rotate 旋转渲染参数（对应前端 gocaptcha-rotate 的 data）
	Angle     int `json:"angle,omitempty"`
	ThumbSize int `json:"thumbSize,omitempty"`
}

// generated provider 生成产物
type generated struct {
	master      string
	tile        string
	thumb       string
	answer      string // 序列化后的标准答案，存入缓存用于二次校验
	thumbX      int
	thumbY      int
	thumbWidth  int
	thumbHeight int
	angle       int
	thumbSize   int
}

// provider 抽象三种验证码的生成与校验
type provider interface {
	Generate() (generated, error)
	Verify(stored, userAnswer string, tolerance int) bool
}

type clickProvider struct{}
type slideProvider struct{}
type rotateProvider struct{}

// 资源单例：字体/背景/拼图加载昂贵，进程内只初始化一次
var (
	captchaAssetsOnce sync.Once
	clickCapt         click.Captcha
	slideCapt         slide.Captcha
	rotateCapt        rotate.Captcha
	assetsErr         error

	memCacheOnce sync.Once
	memCache     local_cache.Cache
)

// ensureCaptchaAssets 懒加载三种验证码的 builder（资源来自 go-captcha-assets）。
// 单例 + sync.Once，避免每次生成都重新读取内嵌资源。
func ensureCaptchaAssets() {
	captchaAssetsOnce.Do(func() {
		// 背景图（click/slide/rotate 共用）
		bgs, err := imagesv2.GetImages()
		if err != nil {
			assetsErr = fmt.Errorf("加载验证码背景图失败: %w", err)
			return
		}

		// click 文字点选
		font, err := fontAsset.GetFont()
		if err != nil {
			assetsErr = fmt.Errorf("加载验证码字体失败: %w", err)
			return
		}
		clickBuilder := click.NewBuilder(
			click.WithRangeLen(option.RangeVal{Min: 4, Max: 6}),
			click.WithRangeVerifyLen(option.RangeVal{Min: 2, Max: 4}),
			click.WithDisabledRangeVerifyLen(true),
			click.WithIsThumbNonDeformAbility(false),
		)
		clickBuilder.SetResources(
			click.WithChars(chars.GetChineseChars()),
			click.WithFonts([]*truetype.Font{font}),
			click.WithBackgrounds(bgs),
		)
		clickCapt = clickBuilder.Make()

		// slide 滑动拼图
		graphAssets, err := tiles.GetTiles()
		if err != nil {
			assetsErr = fmt.Errorf("加载验证码拼图资源失败: %w", err)
			return
		}
		graphImgs := make([]*slide.GraphImage, 0, len(graphAssets))
		for _, g := range graphAssets {
			graphImgs = append(graphImgs, &slide.GraphImage{
				OverlayImage: g.OverlayImage,
				MaskImage:    g.MaskImage,
				ShadowImage:  g.ShadowImage,
			})
		}
		slideBuilder := slide.NewBuilder()
		slideBuilder.SetResources(
			slide.WithGraphImages(graphImgs),
			slide.WithBackgrounds(bgs),
		)
		slideCapt = slideBuilder.Make()

		// rotate 旋转
		rotateBuilder := rotate.NewBuilder()
		rotateBuilder.SetResources(rotate.WithImages(bgs))
		rotateCapt = rotateBuilder.Make()
	})
}

// GetCaptcha 生成验证码。按触发策略决定是否需要验证码：
// 不需要时返回 CaptchaEnabled=false；需要时生成并缓存答案，返回组件数据。
func (s CaptchaService) GetCaptcha(username, ip string) (CaptchaResult, error) {
	res := CaptchaResult{CaptchaEnabled: false}
	cfg := global.OPS_CONFIG.Captcha.GoCaptcha
	if !cfg.Enabled || !s.NeedCaptcha(username, ip) {
		return res, nil
	}
	ensureCaptchaAssets()
	if assetsErr != nil {
		return res, assetsErr
	}
	p, err := s.providerFor(cfg.Type)
	if err != nil {
		return res, err
	}
	g, err := p.Generate()
	if err != nil {
		return res, err
	}
	captchaId := uuid.NewString()
	if err := s.storeAnswer(captchaId, g.answer); err != nil {
		return res, err
	}
	res.CaptchaEnabled = true
	res.Type = cfg.Type
	res.CaptchaId = captchaId
	res.MasterImage = g.master
	res.TileImage = g.tile
	res.ThumbImage = g.thumb
	res.ThumbX = g.thumbX
	res.ThumbY = g.thumbY
	res.ThumbWidth = g.thumbWidth
	res.ThumbHeight = g.thumbHeight
	res.Angle = g.angle
	res.ThumbSize = g.thumbSize
	return res, nil
}

// VerifyCaptcha 二次校验用户答案，校验后立即删除（一次性，防重放）。
func (s CaptchaService) VerifyCaptcha(captchaId, userAnswer string) error {
	if captchaId == "" {
		return fmt.Errorf("验证码ID不能为空")
	}
	cfg := global.OPS_CONFIG.Captcha.GoCaptcha
	stored, ok := s.loadAnswer(captchaId)
	if !ok {
		return fmt.Errorf("验证码已失效，请刷新重试")
	}
	p, err := s.providerFor(cfg.Type)
	if err != nil {
		return err
	}
	if !p.Verify(stored, userAnswer, s.toleranceFor(cfg.Type)) {
		// 失败累计尝试次数；达上限即作废该验证码，防止同一 captchaId 在有效期内暴力试答案
		if s.incrCaptchaAttempt(captchaId) >= maxVerifyAttempts {
			s.deleteAnswer(captchaId)
			s.delCaptchaAttempt(captchaId)
			return fmt.Errorf("验证码错误次数过多，请刷新重试")
		}
		return fmt.Errorf("验证码校验未通过")
	}
	// 校验通过删除答案，保证一次性防重放
	s.deleteAnswer(captchaId)
	s.delCaptchaAttempt(captchaId)
	return nil
}

// NeedCaptcha 判断是否需要验证码（触发策略）。
// 阈值模式下，账号或 IP 任一失败计数达阈值即要求。
func (s CaptchaService) NeedCaptcha(username, ip string) bool {
	cfg := global.OPS_CONFIG.Captcha.GoCaptcha
	if !cfg.Enabled {
		return false
	}
	switch cfg.Trigger.Mode {
	case triggerModeOff:
		return false
	case triggerModeAlways:
		return true
	case triggerModeThreshold:
		threshold := cfg.Trigger.FailThreshold
		if threshold <= 0 {
			threshold = defaultFailThreshold
		}
		if username != "" && s.getFailCount(username) >= threshold {
			return true
		}
		if ip != "" && s.getFailCount("ip:"+ip) >= threshold {
			return true
		}
		return false
	default:
		return false
	}
}

// RecordLoginResult 记录登录结果：失败则账号与 IP 计数各 +1；成功则清零。
func (s CaptchaService) RecordLoginResult(username, ip string, success bool) {
	if success {
		if username != "" {
			s.delFailCount(username)
		}
		if ip != "" {
			s.delFailCount("ip:" + ip)
		}
		return
	}
	window := global.OPS_CONFIG.Captcha.GoCaptcha.Trigger.FailWindow
	if window <= 0 {
		window = defaultFailWindow
	}
	if username != "" {
		s.incrFailCount(username, window)
	}
	if ip != "" {
		s.incrFailCount("ip:"+ip, window)
	}
}

// UnlockUser 清除该账号的登录失败计数（取消强制验证码触发），供登录日志“解锁”调用。
func (s CaptchaService) UnlockUser(username string) {
	if username == "" {
		return
	}
	s.delFailCount(username)
}

func (s CaptchaService) providerFor(t string) (provider, error) {
	switch t {
	case captchaTypeClick:
		return clickProvider{}, nil
	case captchaTypeSlide:
		return slideProvider{}, nil
	case captchaTypeRotate:
		return rotateProvider{}, nil
	default:
		return nil, fmt.Errorf("不支持的验证码类型: %s", t)
	}
}

func (s CaptchaService) toleranceFor(t string) int {
	cfg := global.OPS_CONFIG.Captcha.GoCaptcha
	switch t {
	case captchaTypeClick:
		return orDefault(cfg.Click.Padding, defaultTolerance)
	case captchaTypeSlide:
		return orDefault(cfg.Slide.Tolerance, defaultTolerance)
	case captchaTypeRotate:
		return orDefault(cfg.Rotate.Tolerance, defaultTolerance)
	default:
		return defaultTolerance
	}
}

func orDefault(v, def int) int {
	if v > 0 {
		return v
	}
	return def
}

// ===== click 文字点选 =====

func (clickProvider) Generate() (generated, error) {
	data, err := clickCapt.Generate()
	if err != nil {
		return generated{}, err
	}
	dots := data.GetData()
	// 按 index 排序得到有序目标序列，前端按提示顺序点击
	keys := make([]int, 0, len(dots))
	for k := range dots {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	type dotPos struct {
		X, Y, Width, Height int
	}
	targets := make([]dotPos, 0, len(keys))
	for _, k := range keys {
		d := dots[k]
		targets = append(targets, dotPos{X: d.X, Y: d.Y, Width: d.Width, Height: d.Height})
	}
	b, _ := json.Marshal(targets)
	master, err := data.GetMasterImage().ToBase64()
	if err != nil {
		return generated{}, err
	}
	thumb, err := data.GetThumbImage().ToBase64()
	if err != nil {
		return generated{}, err
	}
	return generated{
		master: master,
		thumb:  thumb,
		answer: string(b),
	}, nil
}

func (clickProvider) Verify(stored, userAnswer string, tolerance int) bool {
	type dotPos struct {
		X, Y, Width, Height int
	}
	type point struct {
		X, Y int
	}
	var targets []dotPos
	if json.Unmarshal([]byte(stored), &targets) != nil {
		return false
	}
	var points []point
	if json.Unmarshal([]byte(userAnswer), &points) != nil {
		return false
	}
	if len(points) != len(targets) || len(points) == 0 {
		return false
	}
	for i, p := range points {
		t := targets[i]
		if !click.Validate(p.X, p.Y, t.X, t.Y, t.Width, t.Height, tolerance) {
			return false
		}
	}
	return true
}

// ===== slide 滑动拼图 =====

func (slideProvider) Generate() (generated, error) {
	data, err := slideCapt.Generate()
	if err != nil {
		return generated{}, err
	}
	b := data.GetData()
	// 校验目标 = 缺口位置 (X, Y)，而非拼图块初始显示位置 (DX, DY)
	ans, _ := json.Marshal(struct {
		X, Y int
	}{X: b.X, Y: b.Y})
	master, err := data.GetMasterImage().ToBase64()
	if err != nil {
		return generated{}, err
	}
	tile, err := data.GetTileImage().ToBase64()
	if err != nil {
		return generated{}, err
	}
	return generated{
		master:      master,
		tile:        tile,
		answer:      string(ans),
		thumbX:      b.DX, // 拼图块初始显示位置（go-captcha basic mode 下位于主图左侧）
		thumbY:      b.DY,
		thumbWidth:  b.Width,
		thumbHeight: b.Height,
	}, nil
}

func (slideProvider) Verify(stored, userAnswer string, tolerance int) bool {
	var tgt struct {
		X, Y int
	}
	if json.Unmarshal([]byte(stored), &tgt) != nil {
		return false
	}
	var u struct {
		X, Y int
	}
	if json.Unmarshal([]byte(userAnswer), &u) != nil {
		return false
	}
	return slide.Validate(u.X, u.Y, tgt.X, tgt.Y, tolerance)
}

// ===== rotate 旋转 =====

func (rotateProvider) Generate() (generated, error) {
	data, err := rotateCapt.Generate()
	if err != nil {
		return generated{}, err
	}
	b := data.GetData()
	ans, _ := json.Marshal(struct {
		Angle int
	}{Angle: b.Angle})
	master, err := data.GetMasterImage().ToBase64()
	if err != nil {
		return generated{}, err
	}
	thumb, err := data.GetThumbImage().ToBase64()
	if err != nil {
		return generated{}, err
	}
	return generated{
		master:    master,
		thumb:     thumb,
		answer:    string(ans),
		angle:     0, // 缩略图初始旋转角度
		thumbSize: b.Width,
	}, nil
}

func (rotateProvider) Verify(stored, userAnswer string, tolerance int) bool {
	var tgt struct {
		Angle int
	}
	if json.Unmarshal([]byte(stored), &tgt) != nil {
		return false
	}
	// 兼容 {"angle":N} 对象或裸数字两种提交形式
	var uobj struct {
		Angle int
	}
	if json.Unmarshal([]byte(userAnswer), &uobj) != nil {
		var unum int
		if json.Unmarshal([]byte(userAnswer), &unum) != nil {
			return false
		}
		uobj.Angle = unum
	}
	return rotate.Validate(uobj.Angle, tgt.Angle, tolerance)
}

// ===== 存储层：Redis 优先，不可用时降级进程内 local_cache =====

func (s CaptchaService) redisAvailable() bool {
	return global.OPS_REDIS != nil
}

func (s CaptchaService) ensureMemCache() local_cache.Cache {
	memCacheOnce.Do(func() {
		ttl := global.OPS_CONFIG.Captcha.GoCaptcha.ExpireSeconds
		if ttl <= 0 {
			ttl = defaultExpireSeconds
		}
		memCache = local_cache.NewCache(local_cache.SetDefaultExpire(time.Duration(ttl) * time.Second))
	})
	return memCache
}

func (s CaptchaService) expireTTL() time.Duration {
	sec := global.OPS_CONFIG.Captcha.GoCaptcha.ExpireSeconds
	if sec <= 0 {
		sec = defaultExpireSeconds
	}
	return time.Duration(sec) * time.Second
}

func (s CaptchaService) answerKey(captchaId string) string {
	return global.OPS_CONFIG.Captcha.GoCaptcha.KeyPrefix + "answer:" + captchaId
}

func (s CaptchaService) failKey(scope string) string {
	return global.OPS_CONFIG.Captcha.GoCaptcha.KeyPrefix + "fail:" + scope
}

func (s CaptchaService) storeAnswer(captchaId, answer string) error {
	if s.redisAvailable() {
		return global.OPS_REDIS.Set(context.Background(), s.answerKey(captchaId), answer, s.expireTTL()).Err()
	}
	s.ensureMemCache().Set(s.answerKey(captchaId), answer, s.expireTTL())
	return nil
}

func (s CaptchaService) loadAnswer(captchaId string) (string, bool) {
	key := s.answerKey(captchaId)
	if s.redisAvailable() {
		val, err := global.OPS_REDIS.Get(context.Background(), key).Result()
		if err == nil && val != "" {
			return val, true
		}
		return "", false
	}
	if v, ok := s.ensureMemCache().Get(key); ok {
		if str, _ := v.(string); str != "" {
			return str, true
		}
	}
	return "", false
}

func (s CaptchaService) deleteAnswer(captchaId string) {
	key := s.answerKey(captchaId)
	if s.redisAvailable() {
		global.OPS_REDIS.Del(context.Background(), key)
		return
	}
	s.ensureMemCache().Delete(key)
}

// attemptKey 单个验证码的失败尝试计数键。
func (s CaptchaService) attemptKey(captchaId string) string {
	return global.OPS_CONFIG.Captcha.GoCaptcha.KeyPrefix + "attempt:" + captchaId
}

// incrCaptchaAttempt 累计该验证码的失败尝试次数，返回当前累计值（与答案同 TTL，随验证码过期而清零）。
func (s CaptchaService) incrCaptchaAttempt(captchaId string) int {
	key := s.attemptKey(captchaId)
	ttl := s.expireTTL()
	if s.redisAvailable() {
		ctx := context.Background()
		if err := global.OPS_REDIS.Incr(ctx, key).Err(); err == nil {
			global.OPS_REDIS.Expire(ctx, key, ttl)
		}
		if v, err := global.OPS_REDIS.Get(ctx, key).Int(); err == nil {
			return v
		}
		return 1
	}
	c := s.ensureMemCache()
	n := 1
	if v, ok := c.Get(key); ok {
		if iv, _ := v.(int); iv > 0 {
			n = iv + 1
		}
	}
	c.Set(key, n, ttl)
	return n
}

// delCaptchaAttempt 清除失败尝试计数。
func (s CaptchaService) delCaptchaAttempt(captchaId string) {
	key := s.attemptKey(captchaId)
	if s.redisAvailable() {
		global.OPS_REDIS.Del(context.Background(), key)
		return
	}
	s.ensureMemCache().Delete(key)
}

func (s CaptchaService) incrFailCount(scope string, windowSec int) {
	key := s.failKey(scope)
	ttl := time.Duration(windowSec) * time.Second
	if s.redisAvailable() {
		ctx := context.Background()
		if err := global.OPS_REDIS.Incr(ctx, key).Err(); err == nil {
			global.OPS_REDIS.Expire(ctx, key, ttl)
		}
		return
	}
	c := s.ensureMemCache()
	var n int
	if v, ok := c.Get(key); ok {
		if iv, _ := v.(int); iv > 0 {
			n = iv
		}
	}
	n++
	c.Set(key, n, ttl)
}

func (s CaptchaService) getFailCount(scope string) int {
	key := s.failKey(scope)
	if s.redisAvailable() {
		if v, err := global.OPS_REDIS.Get(context.Background(), key).Int(); err == nil {
			return v
		}
		return 0
	}
	if v, ok := s.ensureMemCache().Get(key); ok {
		if iv, _ := v.(int); iv > 0 {
			return iv
		}
	}
	return 0
}

func (s CaptchaService) delFailCount(scope string) {
	key := s.failKey(scope)
	if s.redisAvailable() {
		global.OPS_REDIS.Del(context.Background(), key)
		return
	}
	s.ensureMemCache().Delete(key)
}
