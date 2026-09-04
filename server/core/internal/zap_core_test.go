package internal

import (
	"errors"
	"testing"

	"github.com/hllkk/devops-admin/server/utils/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// TestExtractErrorDetail 错误详情提取：builder 的 error_msg 字段与裸 zap.Error/error/err
// 键均须命中，否则 sys_error.info 丢失错误内容（曾因只匹配字面量 "error"/"err" 漏掉 error_msg）。
func TestExtractErrorDetail(t *testing.T) {
	cases := []struct {
		name   string
		fields []zapcore.Field
		want   string
	}{
		{"builder error_msg 字段", []zapcore.Field{zap.String(logger.FieldErrorMsg, "errcode=60020 not allow")}, "errcode=60020 not allow"},
		{"zap.Error 标准字段", []zapcore.Field{zap.Error(errors.New("dial tcp timeout"))}, "dial tcp timeout"},
		{"字符串 error 键", []zapcore.Field{zap.String("error", "boom")}, "boom"},
		{"字符串 err 键", []zapcore.Field{zap.String("err", "bang")}, "bang"},
		{"无错误字段返回空", []zapcore.Field{zap.String("mod", "wecom"), zap.String(logger.FieldRequestID, "r1")}, ""},
		{"多个错误字段取首个", []zapcore.Field{zap.String(logger.FieldErrorMsg, "first"), zap.String("error", "second")}, "first"},
		{"error_msg 空串不截胡后续字段", []zapcore.Field{zap.String(logger.FieldErrorMsg, ""), zap.String("err", "fallback")}, "fallback"},
	}
	for _, tc := range cases {
		if got := extractErrorDetail(tc.fields); got != tc.want {
			t.Errorf("%s: extractErrorDetail() = %q, want %q", tc.name, got, tc.want)
		}
	}
}
