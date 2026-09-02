package system

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/freetype/truetype"
	"github.com/google/uuid"
	"github.com/mojocn/base64Captcha"
	"github.com/wenlng/go-captcha-assets/bindata/chars"
	fontAsset "github.com/wenlng/go-captcha-assets/resources/fonts/fzshengsksjw"
	"github.com/wenlng/go-captcha-assets/resources/imagesv2"
	"github.com/wenlng/go-captcha-assets/resources/tiles"
	"github.com/wenlng/go-captcha/v2/base/option"
	"github.com/wenlng/go-captcha/v2/click"
	"github.com/wenlng/go-captcha/v2/rotate"
	"github.com/wenlng/go-captcha/v2/slide"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
)

const (
	captchaTypeImage     = "image"      // 传统字母数字图形验证码(base64Captcha)
	captchaTypeClick     = "click"      // go-captcha 文字点选
	captchaTypeSlide     = "slide"      // go-captcha 滑动拼图
	captchaTypeRotate    = "rotate"     // go-captcha 旋转
	captchaKeyPrefix     = "gocaptcha:" // OPS_CACHE key 前缀
	defaultExpireSeconds = 120          // 验证码答案默认有效期(秒)，配置 CaptchaTimeout<=0 时兜底
	defaultTolerance     = 5            // 命中容差默认值(click/slide 像素、rotate 角度)
	defaultKeyLong       = 6            // 默认验证码长度/点选字符数
	maxVerifyAttempts    = 3            // 单个验证码最大校验失败次数，超出作废防暴力试答案
)

// CaptchaService 验证码服务：双引擎(image=传统图形 / click|slide|rotate=go-captcha)，
// 经 SysSecurityConfig.CaptchaType 切换，CaptchaEnabled 为总开关。Service 层不依赖 gin.Context，
// IP 由 API 层传入。答案与失败计数统一存 global.OPS_CACHE(Redis 优先、不可用自动降级 memory)，
// 校验一次性消费防重放。
type CaptchaService struct{}

// generated provider 生成产物
type generated struct {
	master      string
	tile        string
	thumb       string
	answer      string // 序列化标准答案，存缓存用于二次校验
	thumbX      int
	thumbY      int
	thumbWidth  int
	thumbHeight int
	angle       int
	thumbSize   int
}

// provider 抽象各验证码类型的生成与校验
type provider interface {
	Generate(cfg system.SysSecurityConfig) (generated, error)
	Verify(stored, userAnswer string, tolerance int) bool
}

type imageProvider struct{}
type clickProvider struct{}
type slideProvider struct{}
type rotateProvider struct{}

// go-captcha 资源单例：字体/背景/拼图加载昂贵，进程内只加载一次。
// builder 按 当前配置(key-long 等)每次现场构建，保证安全配置热更新即时生效。
var (
	captchaResourcesOnce sync.Once
	captchaChars         []string
	captchaFont          *truetype.Font
	captchaBackgrounds   []image.Image
	captchaTiles         []*slide.GraphImage
	resourcesErr         error
)

// ensureCaptchaResources 懒加载 go-captcha 内嵌素材(背景/字体/字符/拼图)，仅加载一次。
func ensureCaptchaResources() {
	captchaResourcesOnce.Do(func() {
		bgs, err := imagesv2.GetImages()
		if err != nil {
			resourcesErr = fmt.Errorf("加载验证码背景图失败: %w", err)
			return
		}
		captchaBackgrounds = bgs

		font, err := fontAsset.GetFont()
		if err != nil {
			resourcesErr = fmt.Errorf("加载验证码字体失败: %w", err)
			return
		}
		captchaFont = font
		captchaChars = chars.GetChineseChars()

		graphAssets, err := tiles.GetTiles()
		if err != nil {
			resourcesErr = fmt.Errorf("加载验证码拼图资源失败: %w", err)
			return
		}
		captchaTiles = make([]*slide.GraphImage, 0, len(graphAssets))
		for _, g := range graphAssets {
			captchaTiles = append(captchaTiles, &slide.GraphImage{
				OverlayImage: g.OverlayImage,
				MaskImage:    g.MaskImage,
				ShadowImage:  g.ShadowImage,
			})
		}
	})
}

// config 读取当前生效的安全配置，DB 未就绪时返回默认。
func (s CaptchaService) config(ctx context.Context) system.SysSecurityConfig {
	svc := SecurityConfigService{}
	return svc.Current(ctx)
}

// Get 生成验证码。总开关关闭或触发策略未达时返回 CaptchaEnabled=false。
func (s CaptchaService) Get(ctx context.Context, username, ip string) (system.CaptchaResult, error) {
	res := system.CaptchaResult{}
	cfg := s.config(ctx)
	if !s.needCaptcha(cfg, username, ip) {
		return res, nil
	}
	if cfg.CaptchaType != captchaTypeImage {
		ensureCaptchaResources()
		if resourcesErr != nil {
			return res, resourcesErr
		}
	}
	p, err := s.providerFor(cfg.CaptchaType)
	if err != nil {
		return res, err
	}
	g, err := p.Generate(cfg)
	if err != nil {
		return res, err
	}
	captchaId := uuid.NewString()
	s.storeAnswer(captchaId, g.answer, s.ttl(cfg))
	res.CaptchaEnabled = true
	res.Type = cfg.CaptchaType
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

// Verify 二次校验用户答案，校验后立即删除(一次性防重放)。供 login 在密码校验前调用。
func (s CaptchaService) Verify(ctx context.Context, captchaId, userAnswer string) error {
	if captchaId == "" {
		return fmt.Errorf("验证码ID不能为空")
	}
	cfg := s.config(ctx)
	stored, ok := s.loadAnswer(captchaId)
	if !ok {
		return fmt.Errorf("验证码已失效，请刷新重试")
	}
	p, err := s.providerFor(cfg.CaptchaType)
	if err != nil {
		return err
	}
	if !p.Verify(stored, userAnswer, toleranceOf(cfg)) {
		// 累计尝试次数；达上限即作废该验证码，防止同一 captchaId 在有效期内暴力试答案
		if n, _ := s.incrAttempt(captchaId, s.ttl(cfg)); n >= maxVerifyAttempts {
			s.deleteAnswer(captchaId)
			s.delAttempt(captchaId)
			return fmt.Errorf("验证码错误次数过多，请刷新重试")
		}
		return fmt.Errorf("验证码校验未通过")
	}
	s.deleteAnswer(captchaId)
	s.delAttempt(captchaId)
	return nil
}

// NeedCaptcha 是否要求提交验证码(供 login 校验入口判断) 导出封装
func (s CaptchaService) NeedCaptcha(ctx context.Context, username, ip string) bool {
	return s.needCaptcha(s.config(ctx), username, ip)
}

// needCaptcha 总开关 CaptchaEnabled 关闭则永不要求；否则按 CaptchaOpen 阈值触发
// (0=每次都要 / N=失败N次后触发)。
func (s CaptchaService) needCaptcha(cfg system.SysSecurityConfig, username, ip string) bool {
	if !cfg.CaptchaEnabled {
		return false
	}
	if cfg.CaptchaOpen == 0 {
		return true
	}
	if username != "" && s.getFailCount(username) >= cfg.CaptchaOpen {
		return true
	}
	if ip != "" && s.getFailCount(ipKey(ip)) >= cfg.CaptchaOpen {
		return true
	}
	return false
}

// RecordLoginFail 登录失败累计账号与 IP 双维度计数(供 login 重建接入)。
func (s CaptchaService) RecordLoginFail(ctx context.Context, username, ip string) {
	window := s.ttl(s.config(ctx))
	if username != "" {
		global.OPS_CACHE.IncrementWithExpire(s.failKey(username), 1, window)
	}
	if ip != "" {
		global.OPS_CACHE.IncrementWithExpire(s.failKey(ipKey(ip)), 1, window)
	}
}

// ResetLoginFail 登录成功清零失败计数(供 login 重建接入)。
func (s CaptchaService) ResetLoginFail(ctx context.Context, username, ip string) {
	if username != "" {
		global.OPS_CACHE.Delete(s.failKey(username))
	}
	if ip != "" {
		global.OPS_CACHE.Delete(s.failKey(ipKey(ip)))
	}
}

func (s CaptchaService) providerFor(t string) (provider, error) {
	switch t {
	case captchaTypeImage:
		return imageProvider{}, nil
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

func (s CaptchaService) ttl(cfg system.SysSecurityConfig) time.Duration {
	sec := cfg.CaptchaTimeout
	if sec <= 0 {
		sec = defaultExpireSeconds
	}
	return time.Duration(sec) * time.Second
}

func toleranceOf(cfg system.SysSecurityConfig) int {
	if cfg.CaptchaTolerance > 0 {
		return cfg.CaptchaTolerance
	}
	return defaultTolerance
}

func ipKey(ip string) string { return "ip:" + ip }

// ===== image 传统图形验证码(base64Captcha，复用 KeyLong/ImgWidth/ImgHeight) =====

func (imageProvider) Generate(cfg system.SysSecurityConfig) (generated, error) {
	drv := &base64Captcha.DriverString{
		Height:          orPositive(cfg.ImgHeight, 80),
		Width:           orPositive(cfg.ImgWidth, 240),
		NoiseCount:      50,
		ShowLineOptions: base64Captcha.OptionShowHollowLine | base64Captcha.OptionShowSlimeLine | base64Captcha.OptionShowSineLine,
		Length:          orPositive(cfg.KeyLong, defaultKeyLong),
		Source:          "ABCDEFGHJKMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789",
		BgColor:         &color.RGBA{R: 240, G: 240, B: 240, A: 255},
		Fonts:           []string{"wqy-microhei.ttc"},
	}
	driver := drv.ConvertFonts()
	_, b64, answer, err := base64Captcha.NewCaptcha(driver, base64Captcha.DefaultMemStore).Generate()
	if err != nil {
		return generated{}, err
	}
	return generated{master: b64, answer: strings.ToLower(strings.TrimSpace(answer))}, nil
}

func (imageProvider) Verify(stored, userAnswer string, tolerance int) bool {
	return strings.EqualFold(stored, strings.TrimSpace(userAnswer))
}

// ===== click 文字点选(主图字符数=KeyLong，校验数与主图数一致) =====

func (clickProvider) Generate(cfg system.SysSecurityConfig) (generated, error) {
	n := orPositive(cfg.KeyLong, defaultKeyLong)
	builder := click.NewBuilder(
		click.WithRangeLen(option.RangeVal{Min: n, Max: n}),
		click.WithRangeVerifyLen(option.RangeVal{Min: 2, Max: 4}), // 被 DisabledRangeVerifyLen 禁用，值无意义
		click.WithDisabledRangeVerifyLen(true),
		click.WithIsThumbNonDeformAbility(false),
	)
	builder.SetResources(
		click.WithChars(captchaChars),
		click.WithFonts([]*truetype.Font{captchaFont}),
		click.WithBackgrounds(captchaBackgrounds),
	)
	data, err := builder.Make().Generate()
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
	return generated{master: master, thumb: thumb, answer: string(b)}, nil
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

func (slideProvider) Generate(cfg system.SysSecurityConfig) (generated, error) {
	builder := slide.NewBuilder()
	builder.SetResources(
		slide.WithGraphImages(captchaTiles),
		slide.WithBackgrounds(captchaBackgrounds),
	)
	data, err := builder.Make().Generate()
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
		thumbX:      b.DX, // 拼图块初始显示位置(basic mode 下位于主图左侧)
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

func (rotateProvider) Generate(cfg system.SysSecurityConfig) (generated, error) {
	builder := rotate.NewBuilder()
	builder.SetResources(rotate.WithImages(captchaBackgrounds))
	data, err := builder.Make().Generate()
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

// ===== 存储层：统一 global.OPS_CACHE(Redis 优先，不可用自动降级 memory) =====

func (s CaptchaService) storeAnswer(captchaId, answer string, ttl time.Duration) {
	global.OPS_CACHE.Set(s.answerKey(captchaId), answer, ttl)
}

func (s CaptchaService) loadAnswer(captchaId string) (string, bool) {
	v, ok := global.OPS_CACHE.Get(s.answerKey(captchaId))
	if !ok {
		return "", false
	}
	str, _ := v.(string)
	return str, str != ""
}

func (s CaptchaService) deleteAnswer(captchaId string) {
	global.OPS_CACHE.Delete(s.answerKey(captchaId))
}

func (s CaptchaService) incrAttempt(captchaId string, ttl time.Duration) (int64, error) {
	return global.OPS_CACHE.IncrementWithExpire(s.attemptKey(captchaId), 1, ttl)
}

func (s CaptchaService) delAttempt(captchaId string) {
	global.OPS_CACHE.Delete(s.attemptKey(captchaId))
}

// getFailCount 读取失败计数。Redis 后端 Get 返回 string、memory 后端返回 int64/int，均兼容。
func (s CaptchaService) getFailCount(scope string) int {
	v, ok := global.OPS_CACHE.Get(s.failKey(scope))
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case string:
		if i, err := strconv.Atoi(n); err == nil {
			return i
		}
	}
	return 0
}

func (s CaptchaService) answerKey(captchaId string) string {
	return captchaKeyPrefix + "answer:" + captchaId
}
func (s CaptchaService) attemptKey(captchaId string) string {
	return captchaKeyPrefix + "attempt:" + captchaId
}
func (s CaptchaService) failKey(scope string) string { return captchaKeyPrefix + "fail:" + scope }

func orPositive(v, def int) int {
	if v > 0 {
		return v
	}
	return def
}
