package system

import (
	"strings"
	"testing"

	"github.com/hllkk/devops-admin/server/model/system"
)

// TestCaptchaProviders 验证四类验证码引擎都能生成合法 base64 主图与非空标准答案，
// 覆盖 go-captcha-assets 资源加载与 base64Captcha 字体绘制两条最易运行时翻车的链路。
func TestCaptchaProviders(t *testing.T) {
	cfg := system.SysSecurityConfig{
		CaptchaEnabled: true,
		CaptchaType:    "click",
		KeyLong:        6,
		ImgWidth:       240,
		ImgHeight:      80,
	}
	ensureCaptchaResources()
	if resourcesErr != nil {
		t.Fatalf("加载 go-captcha 资源失败: %v", resourcesErr)
	}
	cases := []struct {
		name string
		p    provider
	}{
		{"image", imageProvider{}},
		{"click", clickProvider{}},
		{"slide", slideProvider{}},
		{"rotate", rotateProvider{}},
	}
	for _, c := range cases {
		g, err := c.p.Generate(cfg)
		if err != nil {
			t.Fatalf("%s 生成失败: %v", c.name, err)
		}
		if !strings.HasPrefix(g.master, "data:image/") {
			t.Errorf("%s 主图非 base64 data url: %q", c.name, trunc(g.master))
		}
		if g.answer == "" {
			t.Errorf("%s 标准答案为空", c.name)
		}
		switch c.name {
		case "slide":
			if g.tile == "" || g.thumbWidth == 0 {
				t.Errorf("slide 缺少 tileImage/thumbWidth")
			}
		case "click", "rotate":
			if g.thumb == "" {
				t.Errorf("%s 缺少 thumbImage", c.name)
			}
		}
		t.Logf("%s: master=%d字符 answer=%s", c.name, len(g.master), trunc(g.answer))
	}
}

func trunc(s string) string {
	if len(s) > 60 {
		return s[:60] + "..."
	}
	return s
}
