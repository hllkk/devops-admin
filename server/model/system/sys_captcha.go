package system

// CaptchaResult 验证码生成结果，严格对齐前端 Api.Auth.CaptchaResult
// （web/src/typings/api/auth.d.ts）。CaptchaEnabled=false 时其余字段为空。
//
// 各类型字段填充规则：
//   - image：MasterImage + CaptchaId（传统字母数字图形验证码，前端 UI 待补）
//   - click：MasterImage(jpeg) + ThumbImage(png)，用户答案 [{x,y}...]
//   - slide：MasterImage(jpeg) + TileImage(png) + ThumbX/Y/Width/Height，用户答案 {x,y}
//   - rotate：MasterImage(png) + ThumbImage(png) + Angle + ThumbSize，用户答案 {angle}
type CaptchaResult struct {
	CaptchaEnabled bool   `json:"captchaEnabled"`           // 当前是否要求验证码（触发策略决定）
	Type           string `json:"type,omitempty"`           // image | click | slide | rotate
	CaptchaId      string `json:"captchaId,omitempty"`      // 验证码会话 ID
	MasterImage    string `json:"masterImage,omitempty"`    // 主图 base64（含 data:image/*;base64, 前缀）
	TileImage      string `json:"tileImage,omitempty"`      // 拼图块 base64（slide）
	ThumbImage     string `json:"thumbImage,omitempty"`     // 缩略图 base64（click/rotate）
	ThumbX         int    `json:"thumbX,omitempty"`         // slide 拼图块初始 X
	ThumbY         int    `json:"thumbY,omitempty"`         // slide 拼图块初始 Y
	ThumbWidth     int    `json:"thumbWidth,omitempty"`     // slide 拼图块宽度
	ThumbHeight    int    `json:"thumbHeight,omitempty"`    // slide 拼图块高度
	Angle          int    `json:"angle,omitempty"`          // rotate 缩略图初始角度
	ThumbSize      int    `json:"thumbSize,omitempty"`      // rotate 缩略图尺寸
}
